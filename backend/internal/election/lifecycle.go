package election

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrEmptyBallot       = errors.New("add at least one portfolio with a candidate before submitting for approval")
	ErrPortfolioEmpty    = errors.New("every portfolio needs at least one candidate before submitting for approval")
	ErrElectionNotClosed = errors.New("only a closed election can be archived")
)

// SubmitForApproval moves a draft to PENDING_APPROVAL. It refuses an empty
// or half-built ballot - an approver reviewing a portfolio with no
// candidates isn't reviewing anything real, so that gets caught here
// rather than becoming their problem.
func (s *Service) SubmitForApproval(ctx context.Context, electionID string) (*Election, error) {
	e, err := s.Get(ctx, electionID)
	if err != nil {
		return nil, err
	}
	if e.Status != "DRAFT" {
		return nil, ErrElectionNotDraft
	}

	portfolios, err := s.ListPortfolios(ctx, electionID)
	if err != nil {
		return nil, fmt.Errorf("checking ballot: %w", err)
	}
	if len(portfolios) == 0 {
		return nil, ErrEmptyBallot
	}
	for _, p := range portfolios {
		if len(p.Candidates) == 0 {
			return nil, ErrPortfolioEmpty
		}
	}

	if _, err := s.pool.Exec(ctx, `
		UPDATE elections SET status = 'PENDING_APPROVAL', updated_at = now() WHERE id = $1
	`, electionID); err != nil {
		return nil, fmt.Errorf("submitting election: %w", err)
	}

	return s.Get(ctx, electionID)
}

// Delete removes a draft outright. Nothing of consequence can have
// happened to a draft yet - no approval, no votes - so unlike every other
// exit from the lifecycle, this one doesn't need to be reversible.
func (s *Service) Delete(ctx context.Context, electionID string) error {
	e, err := s.Get(ctx, electionID)
	if err != nil {
		return err
	}
	if e.Status != "DRAFT" {
		return ErrElectionNotDraft
	}

	if _, err := s.pool.Exec(ctx, `DELETE FROM elections WHERE id = $1`, electionID); err != nil {
		return fmt.Errorf("deleting election: %w", err)
	}
	return nil
}

// Archive is the terminal step after CLOSED. There's currently no path
// that reaches CLOSED yet - that's the automatic live/close checker, not
// built - so this has nothing to act on until that piece lands, but it's
// cheap and correct to have ready now.
func (s *Service) Archive(ctx context.Context, electionID string) (*Election, error) {
	e, err := s.Get(ctx, electionID)
	if err != nil {
		return nil, err
	}
	if e.Status != "CLOSED" {
		return nil, ErrElectionNotClosed
	}

	if _, err := s.pool.Exec(ctx, `
		UPDATE elections SET status = 'ARCHIVED', updated_at = now() WHERE id = $1
	`, electionID); err != nil {
		return nil, fmt.Errorf("archiving election: %w", err)
	}

	return s.Get(ctx, electionID)
}
