"use client";

import { Election } from "@/lib/api";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { BallotPreview } from "./ballot-preview";

export function SubmitPreviewModal({
  election,
  busy,
  error,
  onClose,
  onConfirm,
}: {
  election: Election;
  busy: boolean;
  error?: string | null;
  onClose: () => void;
  onConfirm: () => void;
}) {
  return (
    <Modal title="Review before submitting" size="lg" onClose={onClose}>
      <p className="mb-5 text-sm text-ink-muted">
        This is exactly what the approver will see. Once submitted, the ballot is locked until
        it&apos;s approved or rejected.
      </p>

      <BallotPreview election={election} />

      {error && <p className="mt-4 text-sm text-danger">{error}</p>}

      <Button variant="accent" onClick={onConfirm} disabled={busy} className="mt-5">
        {busy ? "Submitting..." : "Submit for approval"}
      </Button>
    </Modal>
  );
}
