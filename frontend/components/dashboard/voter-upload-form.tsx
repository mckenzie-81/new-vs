"use client";

import { useState } from "react";
import { ApiError, uploadVoters, ElectorateDatabase } from "@/lib/api";
import { Button } from "@/components/ui/button";

// Reads just enough of the file to split its header row into column
// names - good enough for a "which columns are sensitive" picker, not a
// real CSV parse (quoted commas in a header would confuse it, but that's
// an edge case for a header name, and the server does the real parsing).
async function readHeaderColumns(file: File): Promise<string[]> {
  const sample = await file.slice(0, 8192).text();
  const firstLine = sample.split(/\r?\n/)[0] ?? "";
  return firstLine
    .split(",")
    .map((c) => c.trim().replace(/^﻿/, ""))
    .filter((c) => c !== "");
}

export function VoterUploadForm({
  database,
  onUploaded,
}: {
  database: ElectorateDatabase;
  onUploaded: () => void;
}) {
  const isFirstUpload = database.columns.length === 0;

  const [file, setFile] = useState<File | null>(null);
  const [headerColumns, setHeaderColumns] = useState<string[]>([]);
  const [sensitiveColumns, setSensitiveColumns] = useState<string[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<{ imported: number; skipped: number } | null>(null);

  const candidateColumns = headerColumns.filter(
    (c) => c.toLowerCase() !== "name" && c.toLowerCase() !== "email"
  );

  async function handleFileChange(selected: File | null) {
    setFile(selected);
    setSensitiveColumns([]);
    if (selected && isFirstUpload) {
      try {
        setHeaderColumns(await readHeaderColumns(selected));
      } catch {
        setHeaderColumns([]);
      }
    } else {
      setHeaderColumns([]);
    }
  }

  function toggleSensitive(column: string) {
    setSensitiveColumns((prev) =>
      prev.includes(column) ? prev.filter((c) => c !== column) : [...prev, column]
    );
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!file) return;
    setError(null);
    setResult(null);
    setSubmitting(true);
    try {
      const columnsToHash = isFirstUpload ? sensitiveColumns : database.sensitiveColumns;
      const res = await uploadVoters(database.id, file, columnsToHash);
      setResult(res);
      setFile(null);
      setHeaderColumns([]);
      setSensitiveColumns([]);
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
          onChange={(e) => handleFileChange(e.target.files?.[0] ?? null)}
          className="mt-3 block w-full text-sm text-ink file:mr-3 file:border file:border-line file:bg-transparent file:px-3 file:py-1.5 file:text-sm file:font-semibold file:text-ink hover:file:border-ink"
        />
      </div>

      {isFirstUpload && candidateColumns.length > 0 && (
        <div>
          <p className="text-sm font-semibold text-ink">Mark sensitive columns</p>
          <p className="mt-1 text-sm text-ink-muted">
            Optional. A column marked sensitive (a PIN or secondary ID, say) gets hashed before
            it&apos;s ever stored and can&apos;t be read back by anyone, including an admin - only
            used later to verify a voter alongside their emailed link. This can&apos;t be changed
            after this first upload. Uploads with sensitive columns take longer to process.
          </p>
          <div className="mt-3 flex flex-col gap-2">
            {candidateColumns.map((column) => (
              <label key={column} className="flex items-center gap-2 text-sm text-ink">
                <input
                  type="checkbox"
                  checked={sensitiveColumns.includes(column)}
                  onChange={() => toggleSensitive(column)}
                />
                {column}
              </label>
            ))}
          </div>
        </div>
      )}

      {!isFirstUpload && database.sensitiveColumns.length > 0 && (
        <p className="text-sm text-ink-muted">
          Sensitive columns are fixed from this database&apos;s first upload:{" "}
          <strong className="text-ink">{database.sensitiveColumns.join(", ")}</strong>.
        </p>
      )}

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
