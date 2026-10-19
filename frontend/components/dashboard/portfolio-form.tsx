"use client";

import { useState } from "react";
import { ApiError, createPortfolio } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Field } from "@/components/ui/field";

export function PortfolioForm({
  electionId,
  onCreated,
}: {
  electionId: string;
  onCreated: () => void;
}) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [runningMateRequired, setRunningMateRequired] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      await createPortfolio(electionId, title, description, runningMateRequired);
      onCreated();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Something went wrong. Please try again.");
      setSubmitting(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      <Field label="Title" htmlFor="portfolio-title" hint="e.g. President, Treasurer">
        <Input
          id="portfolio-title"
          required
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />
      </Field>

      <Field label="Description" htmlFor="portfolio-description" hint="Optional">
        <Input
          id="portfolio-description"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
      </Field>

      <label className="flex items-start gap-2.5 text-sm text-ink">
        <input
          type="checkbox"
          checked={runningMateRequired}
          onChange={(e) => setRunningMateRequired(e.target.checked)}
          className="mt-0.5"
        />
        <span>
          <span className="font-semibold">Running mate required</span>
          <span className="block text-ink-muted">
            Every candidate added under this portfolio will be a two-person ticket (e.g.
            President &amp; Vice President), sharing one photo. Can&apos;t be changed after the
            portfolio is created.
          </span>
        </span>
      </label>

      {error && <p className="text-sm text-danger">{error}</p>}

      <Button type="submit" disabled={submitting} className="mt-1">
        {submitting ? "Adding..." : "Add portfolio"}
      </Button>
    </form>
  );
}
