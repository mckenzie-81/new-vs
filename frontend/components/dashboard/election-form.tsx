"use client";

import { useState } from "react";
import { ApiError, createElection, updateElection, Election } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Field } from "@/components/ui/field";
import { ElectorateSelector } from "./electorate-selector";
import { BrandingFields } from "@/components/branding/branding-fields";

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
  const [electorateDatabaseId, setElectorateDatabaseId] = useState(
    election?.electorateDatabaseId ?? ""
  );
  const [electorateFilter, setElectorateFilter] = useState<Record<string, string>>(
    election?.electorateFilter ?? {}
  );
  const [voterVerificationColumns, setVoterVerificationColumns] = useState<string[]>(
    election?.voterVerificationColumns ?? []
  );
  const [accentColor, setAccentColor] = useState(election?.accentColor ?? "");
  const [logoUrl, setLogoUrl] = useState(election?.logoUrl ?? "");
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
      title,
      description,
      startTime: startISO,
      endTime: endISO,
      electorateDatabaseId,
      electorateFilter,
      voterVerificationColumns,
      accentColor,
      logoUrl,
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

      <ElectorateSelector
        databaseId={electorateDatabaseId}
        filter={electorateFilter}
        verificationColumns={voterVerificationColumns}
        onDatabaseChange={setElectorateDatabaseId}
        onFilterChange={setElectorateFilter}
        onVerificationColumnsChange={setVoterVerificationColumns}
      />

      <div>
        <p className="text-sm font-semibold text-ink">Branding</p>
        <p className="mt-1 text-sm text-ink-muted">
          Optional - leave blank to use the system default.
        </p>
        <div className="mt-3">
          <BrandingFields
            accentColor={accentColor}
            logoUrl={logoUrl}
            onAccentColorChange={setAccentColor}
            onLogoUrlChange={setLogoUrl}
          />
        </div>
        {(accentColor || logoUrl) && (
          <button
            type="button"
            onClick={() => {
              setAccentColor("");
              setLogoUrl("");
            }}
            className="mt-2 text-sm font-semibold text-accent hover:underline"
          >
            Use system default
          </button>
        )}
      </div>

      {error && <p className="text-sm text-danger">{error}</p>}

      <Button type="submit" disabled={submitting} className="mt-1">
        {submitting ? "Saving..." : election ? "Save changes" : "Create election"}
      </Button>
    </form>
  );
}
