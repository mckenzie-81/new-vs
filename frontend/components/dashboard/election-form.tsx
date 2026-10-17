"use client";

import { useState } from "react";
import { ApiError, createElection, updateElection, Election } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Field } from "@/components/ui/field";

// datetime-local inputs need "YYYY-MM-DDTHH:mm" in the browser's local
// time - this reverses the ISO string we get back from the server into
// that shape for pre-filling an edit form.
function toLocalInputValue(iso: string): string {
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export function ElectionForm({
  election,
  onSaved,
}: {
  election?: Election;
  onSaved: () => void;
}) {
  const [title, setTitle] = useState(election?.title ?? "");
  const [description, setDescription] = useState(election?.description ?? "");
  const [startTime, setStartTime] = useState(
    election ? toLocalInputValue(election.startTime) : ""
  );
  const [endTime, setEndTime] = useState(election ? toLocalInputValue(election.endTime) : "");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);

    // datetime-local gives a value with no timezone, interpreted as the
    // browser's local time - Date() reads it that way too, so converting
    // to an ISO string here gives the server an unambiguous instant.
    const startISO = new Date(startTime).toISOString();
    const endISO = new Date(endTime).toISOString();

    if (new Date(endISO) <= new Date(startISO)) {
      setError("End time must be after start time.");
      return;
    }

    setSubmitting(true);
    try {
      if (election) {
        await updateElection(election.id, { title, description, startTime: startISO, endTime: endISO });
      } else {
        await createElection({ title, description, startTime: startISO, endTime: endISO });
      }
      onSaved();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Something went wrong. Please try again.");
      setSubmitting(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      <Field label="Title" htmlFor="election-title">
        <Input
          id="election-title"
          required
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />
      </Field>

      <Field label="Description" htmlFor="election-description" hint="Optional">
        <Input
          id="election-description"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
      </Field>

      <div className="grid grid-cols-2 gap-4">
        <Field label="Starts" htmlFor="election-start">
          <Input
            id="election-start"
            required
            type="datetime-local"
            value={startTime}
            onChange={(e) => setStartTime(e.target.value)}
          />
        </Field>

        <Field label="Ends" htmlFor="election-end">
          <Input
            id="election-end"
            required
            type="datetime-local"
            value={endTime}
            onChange={(e) => setEndTime(e.target.value)}
          />
        </Field>
      </div>

      {error && <p className="text-sm text-danger">{error}</p>}

      <Button type="submit" disabled={submitting} className="mt-1">
        {submitting ? "Saving..." : election ? "Save changes" : "Create election"}
      </Button>
    </form>
  );
}
