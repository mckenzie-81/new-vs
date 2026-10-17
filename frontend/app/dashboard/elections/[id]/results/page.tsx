"use client";

import { useEffect, useState } from "react";
import { getElectionResults, ApiError, ElectionResults } from "@/lib/api";
import { useElection } from "@/components/dashboard/election-context";
import { AdminHeader } from "@/components/dashboard/admin-header";

export default function ResultsPage() {
  const { election } = useElection();
  const [results, setResults] = useState<ElectionResults | null>(null);
  const [error, setError] = useState<string | null>(null);

  const isClosed = election.status === "CLOSED" || election.status === "ARCHIVED";

  useEffect(() => {
    if (!isClosed) return;
    getElectionResults(election.id)
      .then(setResults)
      .catch((err) => setError(err instanceof ApiError ? err.message : "Failed to load results."));
  }, [election.id, isClosed]);

  if (!isClosed) {
    return (
      <div>
        <AdminHeader title="Results" intro="Not available until this election closes." />
        <p className="py-10 text-sm text-ink-faint">
          Results will appear here once voting has closed.
        </p>
      </div>
    );
  }

  return (
    <div>
      <AdminHeader title="Results" intro="Final tally for this election." />

      {error && <p className="text-sm text-danger">{error}</p>}
      {!results && !error && <p className="py-10 text-sm text-ink-faint">Loading...</p>}

      {results && (
        <div className="flex flex-col gap-8 border-t border-line pt-6">
          {results.portfolios.map((p) => {
            const maxVotes = Math.max(1, ...p.candidates.map((c) => c.voteCount));
            return (
              <div key={p.portfolioId}>
                <h3 className="text-lg font-semibold text-ink">{p.title}</h3>
                <p className="mt-0.5 text-sm text-ink-faint">{p.totalBallots} ballots cast</p>
                <div className="mt-3 flex flex-col gap-2">
                  {p.candidates.map((c) => (
                    <div key={c.candidateId}>
                      <div className="flex items-baseline justify-between text-sm">
                        <span className="font-medium text-ink">{c.fullName}</span>
                        <span className="tabular-nums text-ink-muted">{c.voteCount}</span>
                      </div>
                      <div className="mt-1 h-2 w-full bg-page">
                        <div
                          className="h-2 bg-accent"
                          style={{ width: `${(c.voteCount / maxVotes) * 100}%` }}
                        />
                      </div>
                    </div>
                  ))}
                  <div className="flex items-baseline justify-between text-sm text-ink-faint">
                    <span>Abstained</span>
                    <span className="tabular-nums">{p.abstainCount}</span>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
