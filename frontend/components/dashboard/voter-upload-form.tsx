"use client";

import { useState } from "react";
import { ApiError, uploadVoters } from "@/lib/api";
import { Button } from "@/components/ui/button";

export function VoterUploadForm({
  databaseId,
  onUploaded,
}: {
  databaseId: string;
  onUploaded: () => void;
}) {
  const [file, setFile] = useState<File | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<{ imported: number; skipped: number } | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!file) return;
    setError(null);
    setResult(null);
    setSubmitting(true);
    try {
      const res = await uploadVoters(databaseId, file);
      setResult(res);
      setFile(null);
      onUploaded();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Something went wrong. Please try again.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      <div>
        <label className="text-sm font-semibold text-ink" htmlFor="voter-csv">
          CSV file
        </label>
        <p className="mt-1 text-sm text-ink-muted">
          Must include a <strong>name</strong> and an <strong>email</strong> column. Any other
          columns (department, year, faculty...) come along as-is and can be used to filter
          who&apos;s eligible when linking this database to an election.
        </p>
        <input
          id="voter-csv"
          type="file"
          accept=".csv,text/csv"
          required
          onChange={(e) => setFile(e.target.files?.[0] ?? null)}
          className="mt-3 block w-full text-sm text-ink file:mr-3 file:border file:border-line file:bg-transparent file:px-3 file:py-1.5 file:text-sm file:font-semibold file:text-ink hover:file:border-ink"
        />
      </div>

      {error && <p className="text-sm text-danger">{error}</p>}

      {result && (
        <p className="text-sm text-ink-muted">
          Imported <strong className="text-ink">{result.imported}</strong>
          {result.skipped > 0 && (
            <>
              , skipped <strong className="text-ink">{result.skipped}</strong> (missing
              name/email, invalid email, or already in this database)
            </>
          )}
          .
        </p>
      )}

      <Button type="submit" disabled={submitting || !file} className="mt-1">
        {submitting ? "Uploading..." : "Upload"}
      </Button>
    </form>
  );
}
