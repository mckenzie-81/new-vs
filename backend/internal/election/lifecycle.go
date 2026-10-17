package election

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const maxRejectionReasonLength = 1000

var (
	ErrEmptyBallot          = errors.New("add at least one portfolio with a candidate before submitting for approval")
	ErrPortfolioEmpty       = errors.New("every portfolio needs at least one candidate before submitting for approval")
	ErrNoElectorateDatabase = errors.New("link an electorate database before submitting for approval")
	ErrNoEligibleVoters     = errors.New("the linked electorate has no eligible voters - check the database and filter before submitting")
	ErrElectionNotClosed    = errors.New("only a closed election can be archived")
	ErrNotPendingApproval   = errors.New("only an election awaiting approval can be approved or rejected")
)

// fixupLifecycle self-corrects the status of any election that has crossed
// a time boundary since it was last read - there's no scheduler to flip
// APPROVED->LIVE->CLOSED as wall-clock time passes, so every read does it
// lazily instead. Both updates are no-ops once an election has already
// crossed the relevant boundary, so steady-state cost is negligible; this
// runs before every Get/List so every other read in the system (voting
// eligibility, results gating) sees a status that actually reflects time.
func (s *Service) fixupLifecycle(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, `
		UPDATE elections SET status = 'LIVE', updated_at = now()
		WHERE status = 'APPROVED' AND start_time <= now()
	`); err != nil {
		return fmt.Errorf("advancing elections to live: %w", err)
	}
	if _, err := s.pool.Exec(ctx, `
		UPDATE elections SET status = 'CLOSED', updated_at = now()
		WHERE status IN ('APPROVED', 'LIVE') AND end_time <= now()
	`); err != nil {
		return fmt.Errorf("advancing elections to closed: %w", err)
	}
	return nil
}

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

	if e.ElectorateDatabaseID == nil {
		return nil, ErrNoElectorateDatabase
	}

	eligibleCount, err := s.electorate.CountEligible(ctx, *e.ElectorateDatabaseID, e.ElectorateFilter)
	if err != nil {
		return nil, fmt.Errorf("checking electorate: %w", err)
	}
	if eligibleCount == 0 {
		return nil, ErrNoEligibleVoters
	}

	if _, err := s.pool.Exec(ctx, `
		UPDATE elections SET status = 'PENDING_APPROVAL', rejection_reason = NULL, updated_at = now() WHERE id = $1
	`, electionID); err != nil {
		return nil, fmt.Errorf("submitting election: %w", err)
	}

	return s.Get(ctx, electionID)
}

// Approve signs off on an election awaiting review. Any active approver can
// call this for any pending election - there's no per-election approver
// assignment, unlike admin assignment.
func (s *Service) Approve(ctx context.Context, electionID, approverID string) (*Election, error) {
	e, err := s.Get(ctx, electionID)
	if err != nil {
		return nil, err
	}
	if e.Status != "PENDING_APPROVAL" {
		return nil, ErrNotPendingApproval
	}

	if _, err := s.pool.Exec(ctx, `
		UPDATE elections SET status = 'APPROVED', approved_by = $2, rejection_reason = NULL, updated_at = now()
		WHERE id = $1
	`, electionID, approverID); err != nil {
		return nil, fmt.Errorf("approving election: %w", err)
	}

	return s.Get(ctx, electionID)
}

// Reject sends an election back to DRAFT with a required reason, stored on
// the row itself (not just the audit log) so the creator actually sees why
// without needing email, which this system doesn't send yet anyway.
func (s *Service) Reject(ctx context.Context, electionID, reason string) (*Election, error) {
	e, err := s.Get(ctx, electionID)
	if err != nil {
		return nil, err
	}
	if e.Status != "PENDING_APPROVAL" {
		return nil, ErrNotPendingApproval
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, fmt.Errorf("%w: a reason is required", ErrInvalidInput)
	}
	if len(reason) > maxRejectionReasonLength {
		return nil, fmt.Errorf("%w: reason must be %d characters or fewer", ErrInvalidInput, maxRejectionReasonLength)
	}

	if _, err := s.pool.Exec(ctx, `
		UPDATE elections SET status = 'DRAFT', approved_by = NULL, rejection_reason = $2, updated_at = now()
		WHERE id = $1
	`, electionID, reason); err != nil {
		return nil, fmt.Errorf("rejecting election: %w", err)
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
