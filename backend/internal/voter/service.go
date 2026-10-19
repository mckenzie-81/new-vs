// Package voter implements the voter-facing half of an election: issuing
// each eligible voter a single-use credential, verifying their identity
// beyond just inbox access, letting them cast a secret ballot, and
// tallying results once voting closes.
//
// The one property every design choice here answers to: a "votes" row
// must never be joinable back to a voter's identity, even by someone with
// full database access. voter_credentials knows who a voter is
// (electorate_voter_id) and stores token_hash = sha256(rawToken). votes
// stores a different, domain-separated hash - credential_hash =
// sha256(rawToken + ":vote:" + electionID) - computed in memory during the
// single authenticated vote-casting request and never persisted anywhere
// alongside token_hash. Neither hash can be derived from the other without
// the raw token itself, so a stolen dump containing both tables still
// can't be joined.
package voter

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"backend/internal/audit"
	"backend/internal/election"
	"backend/internal/electorate"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const verificationLockoutThreshold = 5

var (
	ErrElectionNotIssuable   = errors.New("credentials can only be issued once an election is approved or live, with an electorate database linked")
	ErrInvalidOrExpiredToken = errors.New("this voting link is invalid or has expired")
	ErrCredentialLocked      = errors.New("too many failed attempts - this link has been locked")
	ErrVerificationFailed    = errors.New("the details provided don't match our records")
	ErrAlreadyVoted          = errors.New("you have already voted in this election")
	ErrElectionNotLive       = errors.New("voting is not currently open for this election")
	ErrSessionInvalid        = errors.New("your session has expired - use your voting link again")
	ErrInvalidSelection      = errors.New("invalid ballot selection")
	ErrResultsNotReady       = errors.New("results aren't available until this election has closed")
)

type Service struct {
	pool       *pgxpool.Pool
	election   *election.Service
	electorate *electorate.Service
}

func NewService(pool *pgxpool.Pool, electionService *election.Service, electorateService *electorate.Service) *Service {
	return &Service{pool: pool, election: electionService, electorate: electorateService}
}

func containsFold(list []string, target string) bool {
	for _, item := range list {
		if strings.EqualFold(item, target) {
			return true
		}
	}
	return false
}

func generateVoterToken() (rawToken, tokenHash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	rawToken = hex.EncodeToString(buf)
	return rawToken, hashVoterToken(rawToken), nil
}

func hashVoterToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

// deriveVoteHash produces a value that's cryptographically distinct from
// hashVoterToken's output for the same raw token - see the package comment.
func deriveVoteHash(rawToken, electionID string) string {
	sum := sha256.Sum256([]byte(rawToken + ":vote:" + electionID))
	return hex.EncodeToString(sum[:])
}

// IssuedCredential is a single newly-issued voter's raw token, handed back
// so the caller (the HTTP handler, which owns the base URL) can build an
// invite link. The raw token exists only here and in the voter's own
// browser/inbox afterward - it's never logged or persisted anywhere beyond
// its hash.
type IssuedCredential struct {
	Email    string
	RawToken string
}

type IssueResult struct {
	Eligible      int
	Issued        []IssuedCredential
	AlreadyIssued int
}

// IssueAndInviteCredentials generates one credential per eligible voter
// who doesn't already have one for this election. Safe to call repeatedly:
// already-issued voters are simply skipped, so re-clicking "issue" after
// the electorate grows just invites the new arrivals.
func (s *Service) IssueAndInviteCredentials(ctx context.Context, electionID string) (*IssueResult, error) {
	e, err := s.election.Get(ctx, electionID)
	if err != nil {
		return nil, err
	}
	if e.ElectorateDatabaseID == nil || (e.Status != "APPROVED" && e.Status != "LIVE") {
		return nil, ErrElectionNotIssuable
	}

	eligible, err := s.electorate.ListEligibleVoters(ctx, *e.ElectorateDatabaseID, e.ElectorateFilter)
	if err != nil {
		return nil, fmt.Errorf("listing eligible voters: %w", err)
	}

	result := &IssueResult{Eligible: len(eligible)}
	for _, v := range eligible {
		rawToken, tokenHash, err := generateVoterToken()
		if err != nil {
			return nil, fmt.Errorf("generating voter token: %w", err)
		}

		tag, err := s.pool.Exec(ctx, `
			INSERT INTO voter_credentials (election_id, electorate_voter_id, email, token_hash, expires_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (election_id, electorate_voter_id) DO NOTHING
		`, electionID, v.ID, v.Email, tokenHash, e.EndTime)
		if err != nil {
			return nil, fmt.Errorf("issuing credential: %w", err)
		}
		if tag.RowsAffected() == 0 {
			result.AlreadyIssued++
			continue
		}
		result.Issued = append(result.Issued, IssuedCredential{Email: v.Email, RawToken: rawToken})
	}

	return result, nil
}

// ElectionSummary is the deliberately trimmed, voter-safe projection of
// election.Election - omits CreatedBy/ApprovedBy/RejectionReason and
// everything else admin-internal, which has no business in a voter-facing
// response.
type ElectionSummary struct {
	ID                   string    `json:"id"`
	Title                string    `json:"title"`
	Description          *string   `json:"description"`
	StartTime            time.Time `json:"startTime"`
	EndTime              time.Time `json:"endTime"`
	AccentColor          *string   `json:"accentColor"`
	AccentColorSecondary *string   `json:"accentColorSecondary"`
	AccentColorTertiary  *string   `json:"accentColorTertiary"`
	LogoURL              *string   `json:"logoUrl"`
	BallotStyle          string    `json:"ballotStyle"`
}

type TokenStatus struct {
	Status              string           `json:"status"` // needs_verification | authenticated | voted | locked | invalid
	Election            *ElectionSummary `json:"election,omitempty"`
	VerificationColumns []string         `json:"verificationColumns,omitempty"`
}

// GetTokenStatus is a pure read with no side effects - safe for an email
// client's link-prefetch scanner to hit, the same concern this codebase's
// bootstrap.Confirm already documents for its own tokens. Only the
// explicit AuthenticateVoter call (which requires the verification fields
// a scanner wouldn't have) can change anything.
func (s *Service) GetTokenStatus(ctx context.Context, rawToken string) (*TokenStatus, error) {
	if rawToken == "" {
		return &TokenStatus{Status: "invalid"}, nil
	}
	tokenHash := hashVoterToken(rawToken)

	var (
		electionID, status string
		expiresAt          time.Time
	)
	err := s.pool.QueryRow(ctx, `
		SELECT election_id, status, expires_at FROM voter_credentials WHERE token_hash = $1
	`, tokenHash).Scan(&electionID, &status, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return &TokenStatus{Status: "invalid"}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("looking up credential: %w", err)
	}
	if status != "VOTED" && time.Now().After(expiresAt) {
		return &TokenStatus{Status: "invalid"}, nil
	}

	e, err := s.election.Get(ctx, electionID)
	if err != nil {
		return nil, fmt.Errorf("looking up election: %w", err)
	}
	summary := &ElectionSummary{
		ID: e.ID, Title: e.Title, Description: e.Description, StartTime: e.StartTime, EndTime: e.EndTime,
		AccentColor: e.AccentColor, AccentColorSecondary: e.AccentColorSecondary, AccentColorTertiary: e.AccentColorTertiary,
		LogoURL: e.LogoURL, BallotStyle: e.BallotStyle,
	}

	switch status {
	case "VOTED":
		return &TokenStatus{Status: "voted", Election: summary}, nil
	case "LOCKED":
		return &TokenStatus{Status: "locked"}, nil
	case "AUTHENTICATED":
		return &TokenStatus{Status: "authenticated", Election: summary}, nil
	default: // ISSUED
		return &TokenStatus{Status: "needs_verification", Election: summary, VerificationColumns: e.VoterVerificationColumns}, nil
	}
}

// AuthenticateVoter checks the token and, if the election requires them,
// every verification field - always evaluating all of them regardless of
// an earlier mismatch, so neither the response nor its timing reveals
// which field was wrong. A wrong attempt counts toward the lockout
// threshold even though the overall call fails; success flips the
// credential to AUTHENTICATED, which is what GetBallot/CastVote check for.
func (s *Service) AuthenticateVoter(ctx context.Context, rawToken string, verification map[string]string, ip, userAgent string) (time.Time, error) {
	if rawToken == "" {
		return time.Time{}, ErrInvalidOrExpiredToken
	}
	tokenHash := hashVoterToken(rawToken)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return time.Time{}, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var (
		credentialID, electionID, electorateVoterID, email, status string
		failedAttempts                                             int
		expiresAt                                                  time.Time
	)
	err = tx.QueryRow(ctx, `
		SELECT id, election_id, electorate_voter_id, email, status, failed_attempts, expires_at
		FROM voter_credentials WHERE token_hash = $1 FOR UPDATE
	`, tokenHash).Scan(&credentialID, &electionID, &electorateVoterID, &email, &status, &failedAttempts, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrInvalidOrExpiredToken
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("looking up credential: %w", err)
	}

	switch status {
	case "LOCKED":
		return time.Time{}, ErrCredentialLocked
	case "VOTED":
		return time.Time{}, ErrAlreadyVoted
	}
	if time.Now().After(expiresAt) {
		return time.Time{}, ErrInvalidOrExpiredToken
	}

	e, err := s.election.Get(ctx, electionID)
	if err != nil {
		return time.Time{}, fmt.Errorf("looking up election: %w", err)
	}

	allMatched := true
	if len(e.VoterVerificationColumns) > 0 {
		var raw []byte
		if err := tx.QueryRow(ctx, `SELECT data FROM electorate_voters WHERE id = $1`, electorateVoterID).Scan(&raw); err != nil {
			return time.Time{}, fmt.Errorf("looking up voter data: %w", err)
		}
		var data map[string]string
		if err := json.Unmarshal(raw, &data); err != nil {
			return time.Time{}, fmt.Errorf("decoding voter data: %w", err)
		}

		// A verification column may be a hashed/sensitive one (bcrypt
		// compare) or, when an electorate has nothing marked sensitive, a
		// plain metadata column an admin chose anyway - weaker proof since
		// its value is visible to any systemadmin already, but that's the
		// admin's informed call, not a bug. Either way every column is
		// still checked regardless of an earlier mismatch.
		var sensitiveColumns []string
		if e.ElectorateDatabaseID != nil {
			db, err := s.electorate.GetDatabase(ctx, *e.ElectorateDatabaseID)
			if err != nil {
				return time.Time{}, fmt.Errorf("looking up electorate database: %w", err)
			}
			sensitiveColumns = db.SensitiveColumns
		}

		for _, col := range e.VoterVerificationColumns {
			if containsFold(sensitiveColumns, col) {
				if bcrypt.CompareHashAndPassword([]byte(data[col]), []byte(verification[col])) != nil {
					allMatched = false
				}
			} else if !strings.EqualFold(strings.TrimSpace(data[col]), strings.TrimSpace(verification[col])) {
				allMatched = false
			}
		}
	}

	if !allMatched {
		failedAttempts++
		newStatus := status
		action := "VOTER_VERIFICATION_FAILED"
		if failedAttempts >= verificationLockoutThreshold {
			newStatus = "LOCKED"
			action = "VOTER_CREDENTIAL_LOCKED"
		}
		if _, err := tx.Exec(ctx, `
			UPDATE voter_credentials SET failed_attempts = $1, status = $2, updated_at = now() WHERE id = $3
		`, failedAttempts, newStatus, credentialID); err != nil {
			return time.Time{}, fmt.Errorf("recording failed attempt: %w", err)
		}
		if err := audit.Log(ctx, tx, audit.Entry{
			Action:    action,
			Metadata:  map[string]any{"election_id": electionID, "electorate_voter_id": electorateVoterID, "email": email},
			IPAddress: ip,
			UserAgent: userAgent,
		}); err != nil {
			return time.Time{}, fmt.Errorf("writing audit log: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return time.Time{}, fmt.Errorf("committing failed attempt: %w", err)
		}
		if newStatus == "LOCKED" {
			return time.Time{}, ErrCredentialLocked
		}
		return time.Time{}, ErrVerificationFailed
	}

	if _, err := tx.Exec(ctx, `
		UPDATE voter_credentials SET status = 'AUTHENTICATED', authenticated_at = now(), failed_attempts = 0, updated_at = now()
		WHERE id = $1
	`, credentialID); err != nil {
		return time.Time{}, fmt.Errorf("marking credential authenticated: %w", err)
	}
	if err := audit.Log(ctx, tx, audit.Entry{
		Action:    "VOTER_AUTHENTICATED",
		Metadata:  map[string]any{"election_id": electionID, "electorate_voter_id": electorateVoterID, "email": email},
		IPAddress: ip,
		UserAgent: userAgent,
	}); err != nil {
		return time.Time{}, fmt.Errorf("writing audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return time.Time{}, fmt.Errorf("committing authentication: %w", err)
	}

	return expiresAt, nil
}

type BallotResult struct {
	Election   ElectionSummary      `json:"election"`
	Portfolios []election.Portfolio `json:"portfolios"`
}

// GetBallot doesn't lock the credential row the way CastVote does - it's a
// read that grants no new authority, so there's nothing racy to guard
// against here.
func (s *Service) GetBallot(ctx context.Context, rawToken string) (*BallotResult, error) {
	if rawToken == "" {
		return nil, ErrSessionInvalid
	}
	tokenHash := hashVoterToken(rawToken)

	var electionID, status string
	var expiresAt time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT election_id, status, expires_at FROM voter_credentials WHERE token_hash = $1
	`, tokenHash).Scan(&electionID, &status, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSessionInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("looking up credential: %w", err)
	}
	if status == "VOTED" {
		return nil, ErrAlreadyVoted
	}
	if status != "AUTHENTICATED" {
		return nil, ErrSessionInvalid
	}
	if time.Now().After(expiresAt) {
		return nil, ErrSessionInvalid
	}

	e, err := s.election.Get(ctx, electionID)
	if err != nil {
		return nil, fmt.Errorf("looking up election: %w", err)
	}
	if e.Status != "LIVE" {
		return nil, ErrElectionNotLive
	}

	portfolios, err := s.election.ListPortfolios(ctx, electionID)
	if err != nil {
		return nil, fmt.Errorf("listing portfolios: %w", err)
	}

	return &BallotResult{
		Election: ElectionSummary{
			ID: e.ID, Title: e.Title, Description: e.Description, StartTime: e.StartTime, EndTime: e.EndTime,
			AccentColor: e.AccentColor, AccentColorSecondary: e.AccentColorSecondary, AccentColorTertiary: e.AccentColorTertiary,
			LogoURL: e.LogoURL, BallotStyle: e.BallotStyle,
		},
		Portfolios: portfolios,
	}, nil
}

// CastVote never trusts the submitted selection keys directly - it
// iterates the election's own authoritative portfolio list and looks up
// each one in the submission, so a portfolio missing from the map (or the
// whole map being empty) is simply an abstain on every portfolio, exactly
// like leaving every choice blank in the UI.
func (s *Service) CastVote(ctx context.Context, rawToken string, selections map[string]*string, explicitNo map[string]bool) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if rawToken == "" {
		return ErrSessionInvalid
	}
	tokenHash := hashVoterToken(rawToken)

	var credentialID, electionID, status string
	var expiresAt time.Time
	err = tx.QueryRow(ctx, `
		SELECT id, election_id, status, expires_at FROM voter_credentials WHERE token_hash = $1 FOR UPDATE
	`, tokenHash).Scan(&credentialID, &electionID, &status, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrSessionInvalid
	}
	if err != nil {
		return fmt.Errorf("looking up credential: %w", err)
	}
	if status == "VOTED" {
		return ErrAlreadyVoted
	}
	if status != "AUTHENTICATED" {
		return ErrSessionInvalid
	}
	if time.Now().After(expiresAt) {
		return ErrSessionInvalid
	}

	e, err := s.election.Get(ctx, electionID)
	if err != nil {
		return fmt.Errorf("looking up election: %w", err)
	}
	if e.Status != "LIVE" {
		return ErrElectionNotLive
	}

	portfolios, err := s.election.ListPortfolios(ctx, electionID)
	if err != nil {
		return fmt.Errorf("listing portfolios: %w", err)
	}

	voteHash := deriveVoteHash(rawToken, electionID)

	for _, p := range portfolios {
		var candidateID *string
		if chosen, ok := selections[p.ID]; ok && chosen != nil {
			valid := false
			for _, c := range p.Candidates {
				if c.ID == *chosen {
					valid = true
					break
				}
			}
			if !valid {
				return ErrInvalidSelection
			}
			candidateID = chosen
		}

		// An explicit "No" only means anything for an uncontested
		// portfolio's Yes/No ballot, and never alongside a candidate
		// choice - both would be a contradictory ballot for the same
		// portfolio.
		isNo := explicitNo[p.ID]
		if isNo && (len(p.Candidates) != 1 || candidateID != nil) {
			return ErrInvalidSelection
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO votes (election_id, portfolio_id, candidate_id, credential_hash, is_explicit_no)
			VALUES ($1, $2, $3, $4, $5)
		`, electionID, p.ID, candidateID, voteHash, isNo); err != nil {
			return fmt.Errorf("casting vote: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE voter_credentials SET status = 'VOTED', voted_at = now(), updated_at = now() WHERE id = $1
	`, credentialID); err != nil {
		return fmt.Errorf("marking credential voted: %w", err)
	}

	// Deliberately no electorate_voter_id/email here, and IP/UA blank -
	// unlike VOTER_AUTHENTICATED above, this entry sits right next to the
	// vote's content and must carry nothing that could re-identify whose
	// ballot it was.
	if err := audit.Log(ctx, tx, audit.Entry{
		Action:    "VOTE_CAST",
		Metadata:  map[string]any{"election_id": electionID, "portfolio_count": len(portfolios)},
		IPAddress: "",
		UserAgent: "",
	}); err != nil {
		return fmt.Errorf("writing audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing vote: %w", err)
	}

	return nil
}

type CandidateResult struct {
	CandidateID  string   `json:"candidateId"`
	FullName     string   `json:"fullName"`
	PhotoURL     *string  `json:"photoUrl"`
	RunningMates []string `json:"runningMates"`
	VoteCount    int      `json:"voteCount"`
}

type PortfolioResult struct {
	PortfolioID string            `json:"portfolioId"`
	Title       string            `json:"title"`
	Candidates  []CandidateResult `json:"candidates"`
	// NoCount is only set for an uncontested portfolio (exactly one
	// candidate/ticket) - a deliberate "No" on its Yes/No ballot, kept
	// separate from AbstainCount so a real rejection signal never gets
	// silently lumped in with people who just skipped that portfolio. Nil
	// for any portfolio with more than one candidate.
	NoCount      *int `json:"noCount"`
	AbstainCount int  `json:"abstainCount"`
	TotalBallots int  `json:"totalBallots"`
}

type Results struct {
	ElectionID string            `json:"electionId"`
	Portfolios []PortfolioResult `json:"portfolios"`
}

// GetResults ties "is voting over" to the same election status the
// dashboard's results page already gates on (CLOSED/ARCHIVED), rather than
// a separate wall-clock check - one source of truth for both, kept
// accurate by election.Service's own lazy status fix-up.
func (s *Service) GetResults(ctx context.Context, electionID string) (*Results, error) {
	e, err := s.election.Get(ctx, electionID)
	if err != nil {
		return nil, err
	}
	if e.Status != "CLOSED" && e.Status != "ARCHIVED" {
		return nil, ErrResultsNotReady
	}

	portfolios, err := s.election.ListPortfolios(ctx, electionID)
	if err != nil {
		return nil, fmt.Errorf("listing portfolios: %w", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT portfolio_id, candidate_id, is_explicit_no, count(*)
		FROM votes WHERE election_id = $1 GROUP BY portfolio_id, candidate_id, is_explicit_no
	`, electionID)
	if err != nil {
		return nil, fmt.Errorf("tallying votes: %w", err)
	}
	defer rows.Close()

	// portfolioID -> tallyKey -> count. candidate == "" is either an
	// abstain (no == false) or an explicit No (no == true) - a candidate
	// row is never also explicit-No, CastVote guarantees that.
	type tallyKey struct {
		candidate string
		no        bool
	}
	counts := map[string]map[tallyKey]int{}
	for rows.Next() {
		var portfolioID string
		var candidateID *string
		var isNo bool
		var count int
		if err := rows.Scan(&portfolioID, &candidateID, &isNo, &count); err != nil {
			return nil, fmt.Errorf("scanning tally: %w", err)
		}
		key := tallyKey{no: isNo}
		if candidateID != nil {
			key.candidate = *candidateID
		}
		if counts[portfolioID] == nil {
			counts[portfolioID] = map[tallyKey]int{}
		}
		counts[portfolioID][key] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating tally: %w", err)
	}

	results := &Results{ElectionID: electionID}
	for _, p := range portfolios {
		pr := PortfolioResult{PortfolioID: p.ID, Title: p.Title, Candidates: []CandidateResult{}}
		total := 0
		for _, c := range p.Candidates {
			count := counts[p.ID][tallyKey{candidate: c.ID}]
			pr.Candidates = append(pr.Candidates, CandidateResult{
				CandidateID: c.ID, FullName: c.FullName, PhotoURL: c.PhotoURL, RunningMates: c.RunningMates, VoteCount: count,
			})
			total += count
		}
		pr.AbstainCount = counts[p.ID][tallyKey{}]
		total += pr.AbstainCount
		if len(p.Candidates) == 1 {
			no := counts[p.ID][tallyKey{no: true}]
			pr.NoCount = &no
			total += no
		}
		pr.TotalBallots = total
		results.Portfolios = append(results.Portfolios, pr)
	}

	return results, nil
}
