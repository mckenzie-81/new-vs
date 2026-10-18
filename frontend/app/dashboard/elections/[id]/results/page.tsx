"use client";

import { useEffect, useState } from "react";
import {
  getElectionResults,
  getElectionPulse,
  publishResults,
  ApiError,
  ElectionResults,
} from "@/lib/api";
import { useDashboardUser } from "@/components/dashboard/user-context";
import { useElection } from "@/components/dashboard/election-context";
import { AdminHeader } from "@/components/dashboard/admin-header";
import { PortfolioResults } from "@/components/ballot/portfolio-results";
import { Button } from "@/components/ui/button";

export default function ResultsPage() {
  const user = useDashboardUser();
  const { election, assignedAdmins } = useElection();
  const [results, setResults] = useState<ElectionResults | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [publishedAt, setPublishedAt] = useState<string | null | undefined>(undefined);
  const [publishing, setPublishing] = useState(false);
  const [publishError, setPublishError] = useState<string | null>(null);

  const isClosed = election.status === "CLOSED" || election.status === "ARCHIVED";
  const canPublish =
    user.role === "SYSTEMADMIN" ||
    (user.role === "ADMIN" && assignedAdmins.some((a) => a.userId === user.id));

  function refreshPublishStatus() {
    getElectionPulse(election.id)
      .then((snapshot) => setPublishedAt(snapshot.resultsPublishedAt ?? null))
      .catch(() => setPublishedAt(null));
  }

  useEffect(() => {
    if (!isClosed) return;
    getElectionResults(election.id)
      .then(setResults)
      .catch((err) => setError(err instanceof ApiError ? err.message : "Failed to load results."));
    refreshPublishStatus();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [election.id, isClosed]);

  async function handlePublish() {
    setPublishing(true);
    setPublishError(null);
    try {
      await publishResults(election.id);
      refreshPublishStatus();
    } catch (err) {
      setPublishError(err instanceof ApiError ? err.message : "Failed to publish results.");
    } finally {
      setPublishing(false);
    }
  }

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
      <AdminHeader
        title="Results"
        intro="Final tally for this election."
        actions={
          canPublish && publishedAt === null ? (
            <Button variant="accent" fullWidth={false} onClick={handlePublish} disabled={publishing}>
              {publishing ? "Publishing..." : "Publish to public pulse"}
            </Button>
          ) : undefined
        }
      />

      {publishedAt && (
        <p className="mb-6 text-sm text-ink-muted">
          Published to the public pulse page on {new Date(publishedAt).toLocaleString()}.
        </p>
      )}
      {publishError && <p className="mb-6 text-sm text-danger">{publishError}</p>}

      {error && <p className="text-sm text-danger">{error}</p>}
      {!results && !error && <p className="py-10 text-sm text-ink-faint">Loading...</p>}

      {results && (
        <div className="flex flex-col gap-10 border-t border-line pt-6">
          {results.portfolios.map((p) => (
            <PortfolioResults key={p.portfolioId} portfolio={p} />
          ))}
        </div>
      )}
    </div>
  );
}
