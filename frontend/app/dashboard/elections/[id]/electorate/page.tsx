"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { getElectionElectorate, ApiError, ElectorateSummary } from "@/lib/api";
import { useDashboardUser } from "@/components/dashboard/user-context";
import { useElection } from "@/components/dashboard/election-context";
import { AdminHeader } from "@/components/dashboard/admin-header";

export default function ElectionElectoratePage() {
  const user = useDashboardUser();
  const { election } = useElection();
  const [summary, setSummary] = useState<ElectorateSummary | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getElectionElectorate(election.id)
      .then(setSummary)
      .catch((err) => setError(err instanceof ApiError ? err.message : "Failed to load."));
  }, [election.id]);

  const filterEntries = summary ? Object.entries(summary.filter) : [];

  return (
    <div>
      <AdminHeader title="Electorate" intro="Who's eligible to vote in this election." />

      {error && <p className="text-sm text-danger">{error}</p>}

      {!summary && !error && <p className="py-10 text-sm text-ink-faint">Loading...</p>}

      {summary && (
        <div className="border-t border-line pt-6">
          <p className="text-4xl font-semibold tracking-tight tabular-nums">
            {summary.eligibleCount}
          </p>
          <p className="mt-1 text-sm text-ink-muted">
            {summary.eligibleCount === 1 ? "eligible voter" : "eligible voters"}
          </p>

          <div className="mt-8">
            <p className="text-xs font-semibold uppercase tracking-wide text-ink-faint">
              Database
            </p>
            {user.role === "SYSTEMADMIN" ? (
              <Link
                href={`/dashboard/electorate/${summary.databaseId}`}
                className="mt-1 inline-block text-sm font-semibold text-ink hover:underline"
              >
                {summary.databaseName}
              </Link>
            ) : (
              <p className="mt-1 text-sm font-semibold text-ink">{summary.databaseName}</p>
            )}
          </div>

          <div className="mt-6">
            <p className="text-xs font-semibold uppercase tracking-wide text-ink-faint">
              Eligibility filter
            </p>
            {filterEntries.length === 0 ? (
              <p className="mt-1 text-sm text-ink-muted">
                None - everyone in the database is eligible.
              </p>
            ) : (
              <ul className="mt-1 flex flex-col gap-1">
                {filterEntries.map(([field, value]) => (
                  <li key={field} className="text-sm text-ink-muted">
                    <span className="font-medium text-ink">{field}</span> = {value}
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
