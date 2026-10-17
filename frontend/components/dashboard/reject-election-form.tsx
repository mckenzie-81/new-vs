"use client";

import { useState } from "react";
import { ApiError, rejectElection } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Field } from "@/components/ui/field";

export function RejectElectionForm({
  electionId,
  onRejected,
}: {
  electionId: string;
  onRejected: () => void;
}) {
  const [reason, setReason] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      await rejectElection(electionId, reason);
      onRejected();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Something went wrong. Please try again.");
      setSubmitting(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      <Field
        label="Reason"
        htmlFor="reject-reason"
        hint="Sent back to the creator as a draft, with this reason visible."
      >
        <Textarea
          id="reject-reason"
          required
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
      </Field>

      {error && <p className="text-sm text-danger">{error}</p>}

      <Button type="submit" variant="danger" disabled={submitting} className="mt-1">
        {submitting ? "Rejecting..." : "Reject"}
      </Button>
    </form>
  );
}
