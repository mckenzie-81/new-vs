"use client";

import { useState } from "react";
import { ApiError, updateElectionSchedule, Election } from "@/lib/api";
import { Button, SecondaryButton } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Field } from "@/components/ui/field";

// Mirrors the equivalent helper in election-form.tsx - datetime-local
// inputs need "YYYY-MM-DDTHH:mm" in the browser's local time.
function toLocalInputValue(iso: string): string {
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export function UpdateScheduleForm({
  election,
  onUpdated,
}: {
  election: Election;
  onUpdated: () => void;
}) {
  const [endTime, setEndTime] = useState(toLocalInputValue(election.endTime));
  const [reason, setReason] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const currentEndISO = new Date(election.endTime).toISOString();
  const newEndISO = endTime ? new Date(endTime).toISOString() : null;
  const direction =
    newEndISO && newEndISO !== currentEndISO
      ? newEndISO < currentEndISO
        ? "end this election earlier than scheduled"
        : "extend this election"
      : null;

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);

    if (!newEndISO || newEndISO === currentEndISO) {
      setError("Pick a different end time first.");
      return;
    }

    setSubmitting(true);
    try {
      await updateElectionSchedule(election.id, newEndISO, reason);
      onUpdated();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Something went wrong. Please try again.");
      setSubmitting(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      <Field label="New end time" htmlFor="schedule-end-time">
        <Input
          id="schedule-end-time"
          required
          type="datetime-local"
          value={endTime}
          onChange={(e) => setEndTime(e.target.value)}
        />
      </Field>

      <SecondaryButton
        type="button"
        onClick={() => setEndTime(toLocalInputValue(new Date().toISOString()))}
        className="self-start"
      >
        Use current time (end now)
      </SecondaryButton>

      <Field
        label="Reason"
        htmlFor="schedule-reason"
        hint={
          direction
            ? `This will ${direction}. Shown on this election's public pulse page.`
            : "Shown on this election's public pulse page."
        }
      >
        <Textarea
          id="schedule-reason"
          required
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
      </Field>

      {error && <p className="text-sm text-danger">{error}</p>}

      <Button type="submit" variant="accent" disabled={submitting} className="mt-1">
        {submitting ? "Saving..." : "Update schedule"}
      </Button>
    </form>
  );
}
