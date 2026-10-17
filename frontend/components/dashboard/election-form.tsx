"use client";

import { useState } from "react";
import { ApiError, createElection } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Field } from "@/components/ui/field";

export function ElectionForm({ onCreated }: { onCreated: () => void }) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [startTime, setStartTime] = useState("");
  const [endTime, setEndTime] = useState("");
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
      await createElection({ title, description, startTime: startISO, endTime: endISO });
      onCreated();
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
        {submitting ? "Creating..." : "Create election"}
      </Button>
    </form>
  );
}
