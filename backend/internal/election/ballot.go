package election

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrElectionNotDraft = errors.New("the ballot can only be edited while the election is a draft")
	ErrUnknownOrderItem = errors.New("one of the items doesn't belong here")
)

const maxPhotoURLLength = 2048

// A candidate is the one votable unit a portfolio offers - votes.candidate_id
// always points at exactly one of these, win or lose, regardless of how many
// people are actually on it. RunningMates exists so a ticket (e.g. President
// running with a Vice President) is still a single choice on the ballot and
// a single row here, not two candidates competing against each other for
// the same slot. There's one shared photo for the whole ticket (PhotoURL) -
// running mates are names only, matching how a real ballot typically shows
// a joint ticket as one photographed pair under one set of names.
type Candidate struct {
	ID           string    `json:"id"`
	PortfolioID  string    `json:"portfolioId"`
	FullName     string    `json:"fullName"`
	RunningMates []string  `json:"runningMates"`
	PhotoURL     *string   `json:"photoUrl"`
	DisplayOrder int       `json:"displayOrder"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Portfolio struct {
	ID          string  `json:"id"`
	ElectionID  string  `json:"electionId"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
	// RunningMateRequired makes every candidate under this portfolio a
	// fixed two-person ticket (one running mate, no more, no less) rather
	// than the free-form 0-4 a candidate could carry before - a portfolio
	// either always runs as tickets or never does, not candidate by
	// candidate.
	RunningMateRequired bool        `json:"runningMateRequired"`
	DisplayOrder        int         `json:"displayOrder"`
	CreatedAt           time.Time   `json:"createdAt"`
	Candidates          []Candidate `json:"candidates"`
}

// OrderEntry is one item in a reorder request - an id and the position it
// should move to.
type OrderEntry struct {
	ID    string `json:"id"`
	Order int    `json:"order"`
}

// requireDraft is the one rule every ballot-editing action shares: the
// election must exist and still be a draft, or the ballot a Approver is
// about to review could change out from under them.
func (s *Service) requireDraft(ctx context.Context, electionID string) error {
	e, err := s.Get(ctx, electionID)
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if e.Status != "DRAFT" {
		return ErrElectionNotDraft
	}
	return nil
}

func (s *Service) CreatePortfolio(ctx context.Context, electionID, title, description string, runningMateRequired bool) (*Portfolio, error) {
	if err := s.requireDraft(ctx, electionID); err != nil {
		return nil, err
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	if len(title) > maxTitleLength {
		return nil, fmt.Errorf("%w: title must be %d characters or fewer", ErrInvalidInput, maxTitleLength)
	}

	var desc *string
	if trimmed := strings.TrimSpace(description); trimmed != "" {
		desc = &trimmed
	}

	p := Portfolio{ElectionID: electionID, Candidates: []Candidate{}}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO portfolios (election_id, title, description, running_mate_required, display_order)
		SELECT $1, $2, $3, $4, coalesce(max(display_order) + 1, 0) FROM portfolios WHERE election_id = $1
		RETURNING id, title, description, running_mate_required, display_order, created_at
	`, electionID, title, desc, runningMateRequired).Scan(
		&p.ID, &p.Title, &p.Description, &p.RunningMateRequired, &p.DisplayOrder, &p.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("creating portfolio: %w", err)
	}

	return &p, nil
}

func (s *Service) DeletePortfolio(ctx context.Context, electionID, portfolioID string) error {
	if err := s.requireDraft(ctx, electionID); err != nil {
		return err
	}

	tag, err := s.pool.Exec(ctx, `
		DELETE FROM portfolios WHERE id = $1 AND election_id = $2
	`, portfolioID, electionID)
	if err != nil {
		return fmt.Errorf("deleting portfolio: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// ReorderPortfolios applies a full new ordering in one go. Every id in the
// request must belong to this election, or none of it is applied - a
// partial reorder would leave the ballot in a state nobody asked for.
func (s *Service) ReorderPortfolios(ctx context.Context, electionID string, order []OrderEntry) error {
	if err := s.requireDraft(ctx, electionID); err != nil {
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, entry := range order {
		tag, err := tx.Exec(ctx, `
			UPDATE portfolios SET display_order = $1 WHERE id = $2 AND election_id = $3
		`, entry.Order, entry.ID, electionID)
		if err != nil {
			return fmt.Errorf("reordering portfolio: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrUnknownOrderItem
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing reorder: %w", err)
	}
	return nil
}

// validateCandidateInput enforces the portfolio's own ticket shape: when
// runningMateRequired is true, a candidate must carry exactly one running
// mate (a two-person ticket, never more); when false, none at all. This
// replaced an earlier "0 to 4, candidate's own choice" rule - a portfolio
// either runs as tickets or it doesn't, uniformly for every candidate in
// it.
func validateCandidateInput(fullName, photoURL string, runningMates []string, runningMateRequired bool) (string, *string, []string, error) {
	fullName = strings.TrimSpace(fullName)
	if fullName == "" {
		return "", nil, nil, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if len(fullName) > maxTitleLength {
		return "", nil, nil, fmt.Errorf("%w: name must be %d characters or fewer", ErrInvalidInput, maxTitleLength)
	}

	var photoURLVal *string
	if trimmed := strings.TrimSpace(photoURL); trimmed != "" {
		if len(trimmed) > maxPhotoURLLength {
			return "", nil, nil, fmt.Errorf("%w: photo URL is too long", ErrInvalidInput)
		}
		photoURLVal = &trimmed
	}

	mates := make([]string, 0, len(runningMates))
	for _, m := range runningMates {
		trimmed := strings.TrimSpace(m)
		if trimmed == "" {
			continue
		}
		if len(trimmed) > maxTitleLength {
			return "", nil, nil, fmt.Errorf("%w: a running mate's name must be %d characters or fewer", ErrInvalidInput, maxTitleLength)
		}
		mates = append(mates, trimmed)
	}

	if runningMateRequired && len(mates) != 1 {
		return "", nil, nil, fmt.Errorf("%w: this portfolio requires exactly one running mate per ticket", ErrInvalidInput)
	}
	if !runningMateRequired && len(mates) != 0 {
		return "", nil, nil, fmt.Errorf("%w: this portfolio doesn't use running mates", ErrInvalidInput)
	}

	return fullName, photoURLVal, mates, nil
}

// portfolioRunningMateRequired looks up just the one flag needed to
// validate a candidate against its portfolio's ticket shape, scoped to
// the given election so a candidate can never be attached to a portfolio
// from a different election via a mismatched id.
func (s *Service) portfolioRunningMateRequired(ctx context.Context, electionID, portfolioID string) (bool, error) {
	var required bool
	err := s.pool.QueryRow(ctx, `
		SELECT running_mate_required FROM portfolios WHERE id = $1 AND election_id = $2
	`, portfolioID, electionID).Scan(&required)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrNotFound
	}
	if err != nil {
		return false, fmt.Errorf("looking up portfolio: %w", err)
	}
	return required, nil
}

func (s *Service) CreateCandidate(ctx context.Context, electionID, portfolioID, fullName, photoURL string, runningMates []string) (*Candidate, error) {
	if err := s.requireDraft(ctx, electionID); err != nil {
		return nil, err
	}

	runningMateRequired, err := s.portfolioRunningMateRequired(ctx, electionID, portfolioID)
	if err != nil {
		return nil, err
	}
	fullName, photoURLVal, mates, err := validateCandidateInput(fullName, photoURL, runningMates, runningMateRequired)
	if err != nil {
		return nil, err
	}

	c := Candidate{PortfolioID: portfolioID}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO candidates (portfolio_id, full_name, photo_url, running_mates, display_order)
		SELECT p.id, $2, $3, $4, coalesce((SELECT max(display_order) + 1 FROM candidates WHERE portfolio_id = p.id), 0)
		FROM portfolios p WHERE p.id = $1 AND p.election_id = $5
		RETURNING id, full_name, photo_url, running_mates, display_order, created_at
	`, portfolioID, fullName, photoURLVal, mates, electionID).Scan(
		&c.ID, &c.FullName, &c.PhotoURL, &c.RunningMates, &c.DisplayOrder, &c.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("creating candidate: %w", err)
	}

	return &c, nil
}

// UpdateCandidate changes a candidate's name, photo, and running mates.
// Same draft-only rule as every other ballot edit.
func (s *Service) UpdateCandidate(ctx context.Context, electionID, portfolioID, candidateID, fullName, photoURL string, runningMates []string) (*Candidate, error) {
	if err := s.requireDraft(ctx, electionID); err != nil {
		return nil, err
	}

	runningMateRequired, err := s.portfolioRunningMateRequired(ctx, electionID, portfolioID)
	if err != nil {
		return nil, err
	}
	fullName, photoURLVal, mates, err := validateCandidateInput(fullName, photoURL, runningMates, runningMateRequired)
	if err != nil {
		return nil, err
	}

	c := Candidate{ID: candidateID, PortfolioID: portfolioID}
	err = s.pool.QueryRow(ctx, `
		UPDATE candidates SET full_name = $1, photo_url = $2, running_mates = $3
		WHERE id = $4 AND portfolio_id IN (SELECT id FROM portfolios WHERE id = $5 AND election_id = $6)
		RETURNING full_name, photo_url, running_mates, display_order, created_at
	`, fullName, photoURLVal, mates, candidateID, portfolioID, electionID).Scan(
		&c.FullName, &c.PhotoURL, &c.RunningMates, &c.DisplayOrder, &c.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("updating candidate: %w", err)
	}

	return &c, nil
}

func (s *Service) DeleteCandidate(ctx context.Context, electionID, portfolioID, candidateID string) error {
	if err := s.requireDraft(ctx, electionID); err != nil {
		return err
	}

	tag, err := s.pool.Exec(ctx, `
		DELETE FROM candidates
		WHERE id = $1
		AND portfolio_id IN (SELECT id FROM portfolios WHERE id = $2 AND election_id = $3)
	`, candidateID, portfolioID, electionID)
	if err != nil {
		return fmt.Errorf("deleting candidate: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// ReorderCandidates applies a full new ordering within one portfolio. Same
// all-or-nothing rule as ReorderPortfolios.
func (s *Service) ReorderCandidates(ctx context.Context, electionID, portfolioID string, order []OrderEntry) error {
	if err := s.requireDraft(ctx, electionID); err != nil {
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, entry := range order {
		tag, err := tx.Exec(ctx, `
			UPDATE candidates SET display_order = $1
			WHERE id = $2
			AND portfolio_id IN (SELECT id FROM portfolios WHERE id = $3 AND election_id = $4)
		`, entry.Order, entry.ID, portfolioID, electionID)
		if err != nil {
			return fmt.Errorf("reordering candidate: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrUnknownOrderItem
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing reorder: %w", err)
	}
	return nil
}

// ListPortfolios returns an election's portfolios, each with its candidates
// attached - the whole ballot shape in one call, which is what every caller
// actually wants it for.
func (s *Service) ListPortfolios(ctx context.Context, electionID string) ([]Portfolio, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, election_id, title, description, running_mate_required, display_order, created_at
		FROM portfolios
		WHERE election_id = $1
		ORDER BY display_order ASC
	`, electionID)
	if err != nil {
		return nil, fmt.Errorf("listing portfolios: %w", err)
	}

	portfolios := []Portfolio{}
	byID := map[string]*Portfolio{}
	for rows.Next() {
		var p Portfolio
		p.Candidates = []Candidate{}
		if err := rows.Scan(
			&p.ID, &p.ElectionID, &p.Title, &p.Description, &p.RunningMateRequired, &p.DisplayOrder, &p.CreatedAt,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scanning portfolio: %w", err)
		}
		portfolios = append(portfolios, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating portfolios: %w", err)
	}
	for i := range portfolios {
		byID[portfolios[i].ID] = &portfolios[i]
	}
	if len(portfolios) == 0 {
		return portfolios, nil
	}

	candidateRows, err := s.pool.Query(ctx, `
		SELECT c.id, c.portfolio_id, c.full_name, c.photo_url, c.running_mates, c.display_order, c.created_at
		FROM candidates c
		JOIN portfolios p ON p.id = c.portfolio_id
		WHERE p.election_id = $1
		ORDER BY c.display_order ASC
	`, electionID)
	if err != nil {
		return nil, fmt.Errorf("listing candidates: %w", err)
	}
	defer candidateRows.Close()

	for candidateRows.Next() {
		var c Candidate
		if err := candidateRows.Scan(
			&c.ID, &c.PortfolioID, &c.FullName, &c.PhotoURL, &c.RunningMates, &c.DisplayOrder, &c.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning candidate: %w", err)
		}
		if p, ok := byID[c.PortfolioID]; ok {
			p.Candidates = append(p.Candidates, c)
		}
	}
	if err := candidateRows.Err(); err != nil {
		return nil, fmt.Errorf("iterating candidates: %w", err)
	}

	return portfolios, nil
}
