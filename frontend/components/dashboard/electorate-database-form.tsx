"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { ApiError, createElectorateDatabase, ElectorateDatabase } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Field } from "@/components/ui/field";

export function ElectorateDatabaseForm({ onCreated }: { onCreated: () => void }) {
  const router = useRouter();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // Set only after a successful create - switches this form over to the
  // "what next" confirmation below instead of the modal closing out from
  // under the admin before they've had a chance to add any voters.
  const [created, setCreated] = useState<ElectorateDatabase | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      const result = await createElectorateDatabase(name, description);
      setCreated(result);
      onCreated();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Something went wrong. Please try again.");
      setSubmitting(false);
    }
  }

  if (created) {
    return (
      <div className="flex flex-col gap-4">
        <p className="text-sm text-ink">
          <span className="font-semibold">&ldquo;{created.name}&rdquo;</span> was created.
        </p>
        <p className="text-sm text-ink-muted">
          Next, upload a voter list to it - or close this and do it later.
        </p>
        <Button
          onClick={() => router.push(`/dashboard/electorate/${created.id}`)}
          className="mt-1"
        >
          Continue to add electorates
        </Button>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      <Field label="Name" htmlFor="database-name" hint="e.g. 2026 Student Body, Faculty of Law">
        <Input
          id="database-name"
          required
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
      </Field>

      <Field label="Description" htmlFor="database-description" hint="Optional">
        <Input
          id="database-description"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
      </Field>

      {error && <p className="text-sm text-danger">{error}</p>}

      <Button type="submit" disabled={submitting} className="mt-1">
        {submitting ? "Creating..." : "Create database"}
      </Button>
    </form>
  );
}
