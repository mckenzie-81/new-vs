"use client";

import { useState } from "react";
import { ApiError, createCandidate, updateCandidate, Candidate } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Field } from "@/components/ui/field";
import { ImageUpload } from "@/components/ui/image-upload";

export function CandidateForm({
  electionId,
  portfolioId,
  runningMateRequired,
  candidate,
  onSaved,
}: {
  electionId: string;
  portfolioId: string;
  // Comes from the portfolio, not chosen per-candidate - every candidate
  // under a portfolio has the same ticket shape.
  runningMateRequired: boolean;
  candidate?: Candidate;
  onSaved: () => void;
}) {
  const [fullName, setFullName] = useState(candidate?.fullName ?? "");
  const [photoUrl, setPhotoUrl] = useState(candidate?.photoUrl ?? "");
  const [runningMate, setRunningMate] = useState(candidate?.runningMates[0] ?? "");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const runningMates = runningMateRequired && runningMate.trim() ? [runningMate] : [];

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      if (candidate) {
        await updateCandidate(
          electionId,
          portfolioId,
          candidate.id,
          fullName,
          photoUrl,
          runningMates
        );
      } else {
        await createCandidate(electionId, portfolioId, fullName, photoUrl, runningMates);
      }
      onSaved();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Something went wrong. Please try again.");
      setSubmitting(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      <Field label="Name" htmlFor="candidate-name">
        <Input
          id="candidate-name"
          required
          value={fullName}
          onChange={(e) => setFullName(e.target.value)}
        />
      </Field>

      {runningMateRequired && (
        <Field label="Running mate" htmlFor="candidate-running-mate">
          <Input
            id="candidate-running-mate"
            required
            value={runningMate}
            onChange={(e) => setRunningMate(e.target.value)}
          />
        </Field>
      )}

      <Field label="Photo" htmlFor="candidate-photo" hint="Optional.">
        <div className="mt-1">
          <ImageUpload
            id="candidate-photo"
            kind="candidate-photos"
            value={photoUrl}
            onChange={setPhotoUrl}
          />
        </div>
      </Field>

      {runningMateRequired && (
        <p className="text-sm text-ink-muted">
          This portfolio runs as two-person tickets - the running mate above shares this one
          photo and appears as one choice on the ballot.
        </p>
      )}

      {error && <p className="text-sm text-danger">{error}</p>}

      <Button type="submit" disabled={submitting} className="mt-1">
        {submitting ? "Saving..." : candidate ? "Save changes" : "Add candidate"}
      </Button>
    </form>
  );
}
