"use client";

import { Election, AdminAssignment } from "@/lib/api";
import { Modal } from "@/components/ui/modal";
import { Button, SecondaryButton } from "@/components/ui/button";
import { ElectionReview } from "./election-review";

export function ApprovalReviewModal({
  election,
  assignedAdmins,
  busy,
  error,
  onClose,
  onApprove,
  onReject,
}: {
  election: Election;
  assignedAdmins: AdminAssignment[];
  busy: boolean;
  error?: string | null;
  onClose: () => void;
  onApprove: () => void;
  onReject: () => void;
}) {
  return (
    <Modal title="Review before approving" size="lg" onClose={onClose}>
      <p className="mb-5 text-sm text-ink-muted">
        Check the schedule, electorate, and ballot below - approving locks all of this in for
        when voting opens.
      </p>

      <ElectionReview election={election} assignedAdmins={assignedAdmins} />

      {error && <p className="mt-4 text-sm text-danger">{error}</p>}

      <div className="mt-5 flex gap-3">
        <SecondaryButton onClick={onReject} disabled={busy}>
          Reject instead
        </SecondaryButton>
        <Button variant="accent" fullWidth={false} onClick={onApprove} disabled={busy}>
          {busy ? "Approving..." : "Approve"}
        </Button>
      </div>
    </Modal>
  );
}
