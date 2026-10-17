"use client";

import { useState } from "react";
import { ApiError, createCandidate, updateCandidate, Candidate } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Field } from "@/components/ui/field";

export function CandidateForm({
  electionId,
  portfolioId,
  candidate,
  onSaved,
}: {
  electionId: string;
  portfolioId: string;
  candidate?: Candidate;
  onSaved: () => void;
}) {
  const [fullName, setFullName] = useState(candidate?.fullName ?? "");
  const [photoUrl, setPhotoUrl] = useState(candidate?.photoUrl ?? "");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      if (candidate) {
        await updateCandidate(electionId, portfolioId, candidate.id, fullName, photoUrl);
      } else {
        await createCandidate(electionId, portfolioId, fullName, photoUrl);
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

      {/*
        TEMPORARY: this should be a device file-picker upload, not a pasted
        URL - confirmed explicitly, not just left this way by default. It's
        a URL field only because the storage mechanism (Cloudinary vs. our
        own backend) isn't decided yet. The `photoUrl` string this produces
        is the real data model either way, so swapping this input for a
        real upload widget later won't need backend changes.
      */}
      <Field
        label="Photo URL"
        htmlFor="candidate-photo"
        hint="Paste a link to an image. Optional. (Placeholder - will become a file upload.)"
      >
        <Input
          id="candidate-photo"
          type="url"
          value={photoUrl}
          onChange={(e) => setPhotoUrl(e.target.value)}
        />
      </Field>

      {error && <p className="text-sm text-danger">{error}</p>}

      <Button type="submit" disabled={submitting} className="mt-1">
        {submitting ? "Saving..." : candidate ? "Save changes" : "Add candidate"}
      </Button>
    </form>
  );
}
