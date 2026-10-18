// Package pulse serves the public, no-login turnout-transparency page for
// an election: how many eligible voters have voted, a turnout-over-time
// series, and per-portfolio abstention rates. It deliberately never
// exposes candidate-level standings while an election is contestable - not
// even indirectly. A single-candidate ("uncontested") portfolio's ballot
// count IS that candidate's live vote count, so those numbers are
// suppressed entirely rather than computed and shown.
//
// Isolation is the other half of the design: this package is the only
// code that ever queries votes/voter_credentials/electorate_voters for
// this feature, and it does so on a TTL-gated refresh, never per-request.
// The public HTTP handler (handlers.go) reads only the pre-computed
// election_pulse_snapshots row - never the real voting tables directly -
// so public traffic volume never translates into load on, or an attack
// surface against, the tables that matter for actually casting votes.
package pulse

import (
	"context"
	"errors"
	"fmt"
	"time"

	"backend/internal/election"
	"backend/internal/electorate"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/singleflight"
)

var ErrNotPublic = errors.New("this election isn't public yet")

const snapshotTTL = 15 * time.Second

type ElectionInfo struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	Status      string    `json:"status"`
	StartTime   time.Time `json:"startTime"`
	EndTime     time.Time `json:"endTime"`
	AccentColor *string   `json:"accentColor"`
	LogoURL     *string   `json:"logoUrl"`
}

type TurnoutPoint struct {
	BucketStart time.Time `json:"bucketStart"`
	Count       int       `json:"count"`
}

// PortfolioAbstention is nil-Ballots/nil-Abstentions for an uncontested
// portfolio - see the package doc for why those numbers are never
// computed for one, not just hidden after the fact.
type PortfolioAbstention struct {
	PortfolioID string `json:"portfolioId"`
	Title       string `json:"title"`
	Uncontested bool   `json:"uncontested"`
	Ballots     *int   `json:"ballots"`
	Abstentions *int   `json:"abstentions"`
}

// ScheduleChange is the actor-free public projection of
// election.ScheduleChange - who made the call stays internal, but if the
// posted voting window changed, anyone watching should see that it did
// and why.
type ScheduleChange struct {
	PreviousEndTime time.Time `json:"previousEndTime"`
	NewEndTime      time.Time `json:"newEndTime"`
	Reason          string    `json:"reason"`
	CreatedAt       time.Time `json:"createdAt"`
}

type Snapshot struct {
	Election            ElectionInfo          `json:"election"`
	EligibleCount       int                   `json:"eligibleCount"`
	VotedCount          int                   `json:"votedCount"`
	TurnoutSeries       []TurnoutPoint        `json:"turnoutSeries"`
	PortfolioAbstention []PortfolioAbstention `json:"portfolioAbstention"`
	ScheduleChanges     []ScheduleChange      `json:"scheduleChanges"`
	RefreshedAt         time.Time             `json:"refreshedAt"`
}

type snapshotRow struct {
	EligibleCount       int
	VotedCount          int
	TurnoutSeries       []TurnoutPoint
	PortfolioAbstention []PortfolioAbstention
	RefreshedAt         time.Time
}

type Service struct {
	pool       *pgxpool.Pool
	election   *election.Service
	electorate *electorate.Service
	sf         singleflight.Group
}

func NewService(pool *pgxpool.Pool, electionService *election.Service, electorateService *electorate.Service) *Service {
	return &Service{pool: pool, election: electionService, electorate: electorateService}
}

// GetSnapshot is the only entry point this package exposes to a handler -
// everything else is a private implementation detail of how the snapshot
// gets computed and cached.
func (s *Service) GetSnapshot(ctx context.Context, electionID string) (*Snapshot, error) {
	e, err := s.election.Get(ctx, electionID)
	if err != nil {
		return nil, err
	}
	// A draft or pending election isn't legitimate to show publicly yet -
	// mapped to the same 404 as "doesn't exist" by the handler, so an
	// anonymous caller can't distinguish the two cases.
	if e.Status == "DRAFT" || e.Status == "PENDING_APPROVAL" {
		return nil, ErrNotPublic
	}

	row, err := s.readRow(ctx, electionID)
	if err != nil {
		return nil, err
	}
	if row == nil || time.Since(row.RefreshedAt) > snapshotTTL {
		row, err = s.refresh(ctx, electionID, e)
		if err != nil {
			return nil, err
		}
	}

	// Read live, not cached - this is specifically about disclosing an
	// in-progress change to the posted rules, so it shouldn't lag behind
	// by the same TTL that's fine for turnout numbers. The underlying
	// table isn't one of the three this package otherwise never touches
	// directly (votes/voter_credentials/electorate_voters) - it holds no
	// vote content or voter PII, just the election's own schedule history.
	changes, err := s.election.ListScheduleChanges(ctx, electionID)
	if err != nil {
		return nil, fmt.Errorf("listing schedule changes: %w", err)
	}
	publicChanges := make([]ScheduleChange, 0, len(changes))
	for _, c := range changes {
		publicChanges = append(publicChanges, ScheduleChange{
			PreviousEndTime: c.PreviousEndTime,
			NewEndTime:      c.NewEndTime,
			Reason:          c.Reason,
			CreatedAt:       c.CreatedAt,
		})
	}

	return &Snapshot{
		Election: ElectionInfo{
			ID:          e.ID,
			Title:       e.Title,
			Description: e.Description,
			Status:      e.Status,
			StartTime:   e.StartTime,
			EndTime:     e.EndTime,
			AccentColor: e.AccentColor,
			LogoURL:     e.LogoURL,
		},
		EligibleCount:       row.EligibleCount,
		VotedCount:          row.VotedCount,
		TurnoutSeries:       row.TurnoutSeries,
		PortfolioAbstention: row.PortfolioAbstention,
		ScheduleChanges:     publicChanges,
		RefreshedAt:         row.RefreshedAt,
	}, nil
}

func (s *Service) readRow(ctx context.Context, electionID string) (*snapshotRow, error) {
	var row snapshotRow
	err := s.pool.QueryRow(ctx, `
		SELECT eligible_count, voted_count, turnout_series, portfolio_abstention, refreshed_at
		FROM election_pulse_snapshots WHERE election_id = $1
	`, electionID).Scan(
		&row.EligibleCount, &row.VotedCount, &row.TurnoutSeries, &row.PortfolioAbstention, &row.RefreshedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading pulse snapshot: %w", err)
	}
	return &row, nil
}

// refresh collapses concurrent cache-miss recomputes for the same
// election into one query+upsert via singleflight - many simultaneous
// public viewers hitting a stale cache at the same instant cause one
// recompute, not N. The shared computation runs on a context detached
// from any single caller's cancellation, so one viewer closing their tab
// mid-request can't abort the recompute for everyone else piggybacking on
// it.
func (s *Service) refresh(ctx context.Context, electionID string, e *election.Election) (*snapshotRow, error) {
	v, err, _ := s.sf.Do(electionID, func() (any, error) {
		return s.computeAndStore(context.WithoutCancel(ctx), electionID, e)
	})
	if err != nil {
		return nil, err
	}
	return v.(*snapshotRow), nil
}

func (s *Service) computeAndStore(ctx context.Context, electionID string, e *election.Election) (*snapshotRow, error) {
	var eligible int
	if e.ElectorateDatabaseID != nil {
		var err error
		eligible, err = s.electorate.CountEligible(ctx, *e.ElectorateDatabaseID, e.ElectorateFilter)
		if err != nil {
			return nil, fmt.Errorf("counting eligible voters: %w", err)
		}
	}

	var votedCount int
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM voter_credentials WHERE election_id = $1 AND status = 'VOTED'
	`, electionID).Scan(&votedCount); err != nil {
		return nil, fmt.Errorf("counting voted: %w", err)
	}

	series, err := s.turnoutSeries(ctx, electionID, e.StartTime, e.EndTime)
	if err != nil {
		return nil, err
	}

	abstention, err := s.portfolioAbstention(ctx, electionID)
	if err != nil {
		return nil, err
	}

	row := &snapshotRow{
		EligibleCount:       eligible,
		VotedCount:          votedCount,
		TurnoutSeries:       series,
		PortfolioAbstention: abstention,
	}

	if err := s.pool.QueryRow(ctx, `
		INSERT INTO election_pulse_snapshots (election_id, eligible_count, voted_count, turnout_series, portfolio_abstention, refreshed_at)
		VALUES ($1, $2, $3, $4, $5, now())
		ON CONFLICT (election_id) DO UPDATE SET
			eligible_count = EXCLUDED.eligible_count,
			voted_count = EXCLUDED.voted_count,
			turnout_series = EXCLUDED.turnout_series,
			portfolio_abstention = EXCLUDED.portfolio_abstention,
			refreshed_at = EXCLUDED.refreshed_at
		RETURNING refreshed_at
	`, electionID, row.EligibleCount, row.VotedCount, row.TurnoutSeries, row.PortfolioAbstention,
	).Scan(&row.RefreshedAt); err != nil {
		return nil, fmt.Errorf("storing pulse snapshot: %w", err)
	}

	return row, nil
}

type bucketWidth struct {
	sql string
	dur time.Duration
}

// Candidate bucket widths, smallest first - chooseBucketWidth picks the
// smallest one that keeps the series to a readable number of points
// regardless of how long the election runs.
var bucketWidths = []bucketWidth{
	{"1 minute", time.Minute},
	{"2 minutes", 2 * time.Minute},
	{"5 minutes", 5 * time.Minute},
	{"10 minutes", 10 * time.Minute},
	{"15 minutes", 15 * time.Minute},
	{"30 minutes", 30 * time.Minute},
	{"1 hour", time.Hour},
	{"2 hours", 2 * time.Hour},
	{"3 hours", 3 * time.Hour},
	{"6 hours", 6 * time.Hour},
	{"12 hours", 12 * time.Hour},
	{"1 day", 24 * time.Hour},
	{"2 days", 48 * time.Hour},
	{"7 days", 7 * 24 * time.Hour},
}

const targetBuckets = 30

func chooseBucketWidth(span time.Duration) bucketWidth {
	for _, w := range bucketWidths {
		if span <= 0 || span/w.dur <= targetBuckets {
			return w
		}
	}
	return bucketWidths[len(bucketWidths)-1]
}

// turnoutSeries buckets on voter_credentials.voted_at - one row per
// ballot - never votes.cast_at, which has one row per portfolio per
// ballot and would overcount turnout by the number of portfolios.
// Buckets are anchored to the election's own start time (via date_bin's
// origin), not the wall clock, so a 6-hour election starting at 14:37
// doesn't get a lopsided first bucket.
func (s *Service) turnoutSeries(ctx context.Context, electionID string, startTime, endTime time.Time) ([]TurnoutPoint, error) {
	end := endTime
	if now := time.Now(); now.Before(end) {
		end = now
	}
	span := end.Sub(startTime)
	if span <= 0 {
		return []TurnoutPoint{}, nil
	}

	width := chooseBucketWidth(span)

	rows, err := s.pool.Query(ctx, `
		SELECT date_bin($1::interval, voted_at, $2) AS bucket, count(*)
		FROM voter_credentials
		WHERE election_id = $3 AND status = 'VOTED'
		GROUP BY bucket
	`, width.sql, startTime, electionID)
	if err != nil {
		return nil, fmt.Errorf("querying turnout series: %w", err)
	}
	defer rows.Close()

	counts := map[time.Time]int{}
	for rows.Next() {
		var bucket time.Time
		var count int
		if err := rows.Scan(&bucket, &count); err != nil {
			return nil, fmt.Errorf("scanning turnout bucket: %w", err)
		}
		counts[bucket] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating turnout series: %w", err)
	}

	// Fill every bucket in range, not just the ones with votes, so a
	// quiet period shows as a real zero rather than a gap in the chart.
	series := []TurnoutPoint{}
	for t := startTime; !t.After(end); t = t.Add(width.dur) {
		series = append(series, TurnoutPoint{BucketStart: t, Count: counts[t]})
	}
	return series, nil
}

func (s *Service) portfolioAbstention(ctx context.Context, electionID string) ([]PortfolioAbstention, error) {
	portfolios, err := s.election.ListPortfolios(ctx, electionID)
	if err != nil {
		return nil, fmt.Errorf("listing portfolios: %w", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT portfolio_id, count(*) AS ballots, count(*) FILTER (WHERE candidate_id IS NULL) AS abstentions
		FROM votes WHERE election_id = $1 GROUP BY portfolio_id
	`, electionID)
	if err != nil {
		return nil, fmt.Errorf("querying portfolio abstention: %w", err)
	}
	defer rows.Close()

	type counts struct{ ballots, abstentions int }
	byPortfolio := map[string]counts{}
	for rows.Next() {
		var portfolioID string
		var c counts
		if err := rows.Scan(&portfolioID, &c.ballots, &c.abstentions); err != nil {
			return nil, fmt.Errorf("scanning portfolio abstention: %w", err)
		}
		byPortfolio[portfolioID] = c
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating portfolio abstention: %w", err)
	}

	result := make([]PortfolioAbstention, 0, len(portfolios))
	for _, p := range portfolios {
		pa := PortfolioAbstention{PortfolioID: p.ID, Title: p.Title}
		if len(p.Candidates) == 1 {
			// Uncontested - ballots cast for this portfolio would be
			// exactly that one candidate's live vote count. Never
			// compute or expose it, not even "just the count".
			pa.Uncontested = true
		} else {
			c := byPortfolio[p.ID]
			ballots, abstentions := c.ballots, c.abstentions
			pa.Ballots = &ballots
			pa.Abstentions = &abstentions
		}
		result = append(result, pa)
	}
	return result, nil
}
