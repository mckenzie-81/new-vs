"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import {
  getElectionElectorate,
  issueVoterCredentials,
  ApiError,
  ElectorateSummary,
  IssueVoterCredentialsResult,
} from "@/lib/api";
import { useDashboardUser } from "@/components/dashboard/user-context";
import { useElection } from "@/components/dashboard/election-context";
import { AdminHeader } from "@/components/dashboard/admin-header";
import { Button } from "@/components/ui/button";

export default function ElectionElectoratePage() {
  const user = useDashboardUser();
  const { election } = useElection();
  const [summary, setSummary] = useState<ElectorateSummary | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [issuing, setIssuing] = useState(false);
  const [issueError, setIssueError] = useState<string | null>(null);
  const [issueResult, setIssueResult] = useState<IssueVoterCredentialsResult | null>(null);

  useEffect(() => {
    getElectionElectorate(election.id)
      .then(setSummary)
      .catch((err) => setError(err instanceof ApiError ? err.message : "Failed to load."));
  }, [election.id]);

  async function handleIssue() {
    setIssuing(true);
    setIssueError(null);
    try {
      setIssueResult(await issueVoterCredentials(election.id));
    } catch (err) {
      setIssueError(err instanceof ApiError ? err.message : "Failed to issue voter credentials.");
    } finally {
      setIssuing(false);
    }
  }

  const canIssue =
    (election.status === "APPROVED" || election.status === "LIVE") &&
    (user.role === "SYSTEMADMIN" || user.role === "ADMIN");

  const filterEntries = summary ? Object.entries(summary.filter) : [];

  return (
    <div>
      <AdminHeader title="Electorate" intro="Who's eligible to vote in this election." />

      {error && <p className="text-sm text-danger">{error}</p>}

      {!summary && !error && <p className="py-10 text-sm text-ink-faint">Loading...</p>}

      {summary && !summary.linked && (
        <p className="py-10 text-sm text-ink-faint">
          No electorate database linked yet.
          {election.status === "DRAFT" &&
            " Use the edit button on this election to add one before submitting for approval."}
        </p>
      )}

      {summary && summary.linked && (
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

          {canIssue && (
            <div className="mt-8 border-t border-line pt-6">
              <p className="text-xs font-semibold uppercase tracking-wide text-ink-faint">
                Voter credentials
              </p>
              <p className="mt-1 text-sm text-ink-muted">
                Generates one voting link per eligible voter who doesn&apos;t already have one,
                and emails it to them. Safe to click again later if new voters are added - they
                won&apos;t get a second link.
              </p>
              <Button
                fullWidth={false}
                className="mt-3"
                onClick={handleIssue}
                disabled={issuing}
              >
                {issuing ? "Issuing..." : "Issue & send voter credentials"}
              </Button>

              {issueError && <p className="mt-3 text-sm text-danger">{issueError}</p>}

              {issueResult && (
                <div className="mt-3">
                  <p className="text-sm text-ink-muted">
                    {issueResult.eligible} eligible, {issueResult.issued} newly issued,{" "}
                    {issueResult.alreadyIssued} already had a link.
                  </p>
                  {issueResult.invites.length > 0 && (
                    <>
                      <p className="mt-2 text-xs text-ink-faint">
                        No email sending is wired up yet - here are the links to use for testing.
                      </p>
                      <ul className="mt-1 max-h-64 overflow-y-auto text-sm text-ink-muted">
                        {issueResult.invites.map((invite) => (
                          <li key={invite.email} className="py-0.5">
                            <span className="font-medium text-ink">{invite.email}</span>:{" "}
                            <a href={invite.link} className="text-accent hover:underline">
                              {invite.link}
                            </a>
                          </li>
                        ))}
                      </ul>
                    </>
                  )}
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
