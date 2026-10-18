"use client";

import { useEffect, useState } from "react";
import { getElectionResults, candidateTicketName, ApiError, ElectionResults } from "@/lib/api";
import { useElection } from "@/components/dashboard/election-context";
import { AdminHeader } from "@/components/dashboard/admin-header";
import { CandidatePhoto } from "@/components/ballot/candidate-option";

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
        <div className="flex flex-col gap-10 border-t border-line pt-6">
          {results.portfolios.map((p) => {
            const maxVotes = Math.max(1, ...p.candidates.map((c) => c.voteCount));
            const sorted = [...p.candidates].sort((a, b) => b.voteCount - a.voteCount);
            // Every candidate at the max is a winner - a tie doesn't pick
            // one to highlight, it highlights all of them.
            const topVotes = sorted[0]?.voteCount ?? 0;
            const hasWinner = topVotes > 0;
            return (
              <div key={p.portfolioId}>
                <div className="flex items-baseline justify-between gap-3 border-l-2 border-accent bg-accent-soft py-2.5 pl-3 pr-3">
                  <h3 className="text-base font-semibold tracking-wide text-ink">{p.title}</h3>
                  <p className="text-sm text-ink-faint">{p.totalBallots} ballots cast</p>
                </div>
                <div className="mt-4 flex flex-col gap-3">
                  {sorted.map((c) => {
                    const winner = hasWinner && c.voteCount === topVotes;
                    const share = p.totalBallots > 0 ? (c.voteCount / p.totalBallots) * 100 : 0;
                    return (
                      <div
                        key={c.candidateId}
                        className={`flex items-center gap-3 border-2 p-2.5 ${
                          winner ? "border-success" : "border-line"
                        }`}
                      >
                        <CandidatePhoto
                          candidate={c}
                          className="h-14 w-12 shrink-0 text-lg"
                        />
                        <div className="min-w-0 flex-1">
                          <div className="flex items-baseline justify-between gap-2">
                            <span
                              className={`truncate text-sm font-semibold ${
                                winner ? "text-success" : "text-ink"
                              }`}
                            >
                              {candidateTicketName(c)}
                              {winner && " ✓"}
                            </span>
                            <span
                              className={`shrink-0 tabular-nums text-sm ${
                                winner ? "font-semibold text-success" : "text-ink-muted"
                              }`}
                            >
                              {c.voteCount}{" "}
                              <span className="text-xs text-ink-faint">
                                ({share.toFixed(0)}%)
                              </span>
                            </span>
                          </div>
                          <div className="mt-1.5 h-2 w-full bg-page">
                            <div
                              className={`h-2 transition-[width] duration-300 ${
                                winner ? "bg-success" : "bg-accent"
                              }`}
                              style={{ width: `${(c.voteCount / maxVotes) * 100}%` }}
                            />
                          </div>
                        </div>
                      </div>
                    );
                  })}
                  <div className="flex items-baseline justify-between px-2.5 text-sm text-ink-faint">
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
