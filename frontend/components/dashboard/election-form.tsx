"use client";

import { useState } from "react";
import { ApiError, createElection, updateElection, electionToCreateInput, Election } from "@/lib/api";
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

// Deliberately just name and timeline - everything else an election can
// have (electorate, branding, ballot style, description) is configured
// separately, on its own tab, once the election exists. Keeps creating an
// election fast, and keeps a long list of optional settings from being
// the first thing in the way of the one thing every election needs.
export function ElectionForm({
  election,
  onSaved,
}: {
  election?: Election;
  onSaved: () => void;
}) {
  const [title, setTitle] = useState(election?.title ?? "");
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

    const payload = {
      ...(election ? electionToCreateInput(election) : emptyElectionInput()),
      title,
      startTime: startISO,
      endTime: endISO,
    };

    setSubmitting(true);
    try {
      if (election) {
        await updateElection(election.id, payload);
      } else {
        await createElection(payload);
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

function emptyElectionInput() {
  return {
    title: "",
    description: "",
    startTime: "",
    endTime: "",
    electorateDatabaseId: "",
    electorateFilter: {},
    voterVerificationColumns: [],
    accentColor: "",
    logoUrl: "",
    ballotStyle: "",
  };
}
