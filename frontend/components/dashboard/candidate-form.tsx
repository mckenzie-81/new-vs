"use client";

import { useState } from "react";
import { ApiError, createCandidate } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Field } from "@/components/ui/field";

export function CandidateForm({
  electionId,
  portfolioId,
  onCreated,
}: {
  electionId: string;
  portfolioId: string;
  onCreated: () => void;
}) {
  const [fullName, setFullName] = useState("");
  const [manifesto, setManifesto] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      await createCandidate(electionId, portfolioId, fullName, manifesto);
      onCreated();
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

      <Field label="Manifesto" htmlFor="candidate-manifesto" hint="Optional">
        <Input
          id="candidate-manifesto"
          value={manifesto}
          onChange={(e) => setManifesto(e.target.value)}
        />
      </Field>

      {error && <p className="text-sm text-danger">{error}</p>}

      <Button type="submit" disabled={submitting} className="mt-1">
        {submitting ? "Adding..." : "Add candidate"}
      </Button>
    </form>
  );
}
