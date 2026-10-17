package election

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrElectionNotDraft = errors.New("the ballot can only be edited while the election is a draft")

type Candidate struct {
	ID          string    `json:"id"`
	PortfolioID string    `json:"portfolioId"`
	FullName    string    `json:"fullName"`
	Manifesto   *string   `json:"manifesto"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Portfolio struct {
	ID          string      `json:"id"`
	ElectionID  string      `json:"electionId"`
	Title       string      `json:"title"`
	Description *string     `json:"description"`
	CreatedAt   time.Time   `json:"createdAt"`
	Candidates  []Candidate `json:"candidates"`
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

func (s *Service) CreatePortfolio(ctx context.Context, electionID, title, description string) (*Portfolio, error) {
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
		INSERT INTO portfolios (election_id, title, description)
		VALUES ($1, $2, $3)
		RETURNING id, title, description, created_at
	`, electionID, title, desc).Scan(&p.ID, &p.Title, &p.Description, &p.CreatedAt)
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

func (s *Service) CreateCandidate(ctx context.Context, electionID, portfolioID, fullName, manifesto string) (*Candidate, error) {
	if err := s.requireDraft(ctx, electionID); err != nil {
		return nil, err
	}

	fullName = strings.TrimSpace(fullName)
	if fullName == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if len(fullName) > maxTitleLength {
		return nil, fmt.Errorf("%w: name must be %d characters or fewer", ErrInvalidInput, maxTitleLength)
	}

	var manifestoVal *string
	if trimmed := strings.TrimSpace(manifesto); trimmed != "" {
		manifestoVal = &trimmed
	}

	c := Candidate{PortfolioID: portfolioID}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO candidates (portfolio_id, full_name, manifesto)
		SELECT id, $2, $3 FROM portfolios WHERE id = $1 AND election_id = $4
		RETURNING id, full_name, manifesto, created_at
	`, portfolioID, fullName, manifestoVal, electionID).Scan(&c.ID, &c.FullName, &c.Manifesto, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("creating candidate: %w", err)
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

// ListPortfolios returns an election's portfolios, each with its candidates
// attached - the whole ballot shape in one call, which is what every caller
// actually wants it for.
func (s *Service) ListPortfolios(ctx context.Context, electionID string) ([]Portfolio, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, election_id, title, description, created_at
		FROM portfolios
		WHERE election_id = $1
		ORDER BY created_at ASC
	`, electionID)
	if err != nil {
		return nil, fmt.Errorf("listing portfolios: %w", err)
	}

	portfolios := []Portfolio{}
	byID := map[string]*Portfolio{}
	for rows.Next() {
		var p Portfolio
		p.Candidates = []Candidate{}
		if err := rows.Scan(&p.ID, &p.ElectionID, &p.Title, &p.Description, &p.CreatedAt); err != nil {
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
		SELECT c.id, c.portfolio_id, c.full_name, c.manifesto, c.created_at
		FROM candidates c
		JOIN portfolios p ON p.id = c.portfolio_id
		WHERE p.election_id = $1
		ORDER BY c.created_at ASC
	`, electionID)
	if err != nil {
		return nil, fmt.Errorf("listing candidates: %w", err)
	}
	defer candidateRows.Close()

	for candidateRows.Next() {
		var c Candidate
		if err := candidateRows.Scan(&c.ID, &c.PortfolioID, &c.FullName, &c.Manifesto, &c.CreatedAt); err != nil {
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
