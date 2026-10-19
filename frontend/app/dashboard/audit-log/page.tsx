"use client";

import { useCallback, useEffect, useState } from "react";
import {
  listAuditLog,
  verifyAuditLog,
  AuditLogEntry,
  AuditChainVerification,
  ApiError,
} from "@/lib/api";
import { AdminHeader } from "@/components/dashboard/admin-header";
import { Breadcrumbs } from "@/components/dashboard/breadcrumbs";
import { Button } from "@/components/ui/button";

const PAGE_SIZE = 50;

const actionLabel: Record<string, string> = {
  LOGIN_SUCCEEDED: "Logged in",
  LOGIN_FAILED: "Login failed",
  LOGIN_BLOCKED_DISABLED: "Login blocked (account disabled)",
  LOGIN_BLOCKED_LOCKED: "Login blocked (account locked)",
  ACCOUNT_DISABLED: "Account disabled",
  ACCOUNT_REACTIVATED: "Account reactivated",
  INVITE_CREATED: "Invite sent",
  INVITE_ACCEPTED: "Invite accepted",
  INVITE_ACCEPT_REPEATED: "Invite link reused",
  INVITE_REVOKED: "Invite revoked",
  PASSWORD_RESET_REQUESTED: "Password reset requested",
  PASSWORD_RESET_COMPLETED: "Password reset completed",
  BOOTSTRAP_INITIATED: "System setup initiated",
  BOOTSTRAP_INITIATE_REJECTED: "System setup rejected",
  BOOTSTRAP_CONFIRM_REPEATED: "System setup link reused",
  SYSTEMADMIN_BOOTSTRAPPED: "First systemadmin created",
  ELECTION_CREATED: "Election created",
  ELECTION_UPDATED: "Election updated",
  ELECTION_DELETED: "Election deleted",
  ELECTION_SUBMITTED: "Election submitted for approval",
  ELECTION_APPROVED: "Election approved",
  ELECTION_REJECTED: "Election rejected",
  ELECTION_ARCHIVED: "Election archived",
  ELECTION_SCHEDULE_CHANGED: "Election schedule changed",
  ELECTION_ADMIN_ASSIGNED: "Admin assigned to election",
  ELECTION_ADMIN_UNASSIGNED: "Admin unassigned from election",
  PORTFOLIO_CREATED: "Portfolio created",
  PORTFOLIO_DELETED: "Portfolio deleted",
  PORTFOLIOS_REORDERED: "Portfolios reordered",
  CANDIDATE_CREATED: "Candidate added",
  CANDIDATE_UPDATED: "Candidate updated",
  CANDIDATE_DELETED: "Candidate deleted",
  CANDIDATES_REORDERED: "Candidates reordered",
  ELECTORATE_DATABASE_CREATED: "Electorate database created",
  ELECTORATE_DATABASE_DELETED: "Electorate database deleted",
  ELECTORATE_VOTERS_UPLOADED: "Voters uploaded",
  VOTER_CREDENTIALS_ISSUED: "Voter credentials issued",
  VOTER_CREDENTIAL_LOCKED: "Voter credential locked",
  VOTER_VERIFICATION_FAILED: "Voter verification failed",
  VOTER_AUTHENTICATED: "Voter authenticated",
  VOTE_CAST: "Vote cast",
};

function describe(entry: AuditLogEntry) {
  return actionLabel[entry.action] ?? entry.action.toLowerCase().replaceAll("_", " ");
}

function actorLabel(entry: AuditLogEntry) {
  if (entry.actorName) return entry.actorName;
  if (entry.actorEmail) return entry.actorEmail;
  return "System";
}

export default function AuditLogPage() {
  const [entries, setEntries] = useState<AuditLogEntry[] | null>(null);
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<Set<number>>(new Set());
  const [verification, setVerification] = useState<AuditChainVerification | null>(null);
  const [verifying, setVerifying] = useState(false);
  const [verifyError, setVerifyError] = useState<string | null>(null);

  const load = useCallback((before?: number) => {
    const setBusy = before === undefined ? () => {} : setLoadingMore;
    setBusy(true);
    listAuditLog({ before, limit: PAGE_SIZE })
      .then((page) => {
        setEntries((prev) => (before === undefined ? page : [...(prev ?? []), ...page]));
        setHasMore(page.length === PAGE_SIZE);
        setError(null);
      })
      .catch((err) => setError(err instanceof ApiError ? err.message : "Failed to load audit log."))
      .finally(() => setBusy(false));
  }, []);

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function toggle(seq: number) {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(seq)) next.delete(seq);
      else next.add(seq);
      return next;
    });
  }

  async function handleVerify() {
    setVerifying(true);
    setVerifyError(null);
    try {
      setVerification(await verifyAuditLog());
    } catch (err) {
      setVerifyError(err instanceof ApiError ? err.message : "Failed to verify audit log.");
    } finally {
      setVerifying(false);
    }
  }

  return (
    <div>
      <Breadcrumbs items={[{ label: "Dashboard", href: "/dashboard" }, { label: "Audit log" }]} />
      <AdminHeader
        title="Audit log"
        intro="Every sensitive action in the system, in order."
        actions={
          <Button fullWidth={false} onClick={handleVerify} disabled={verifying}>
            {verifying ? "Verifying..." : "Verify integrity"}
          </Button>
        }
      />

      {verifyError && <p className="mb-6 text-sm text-danger">{verifyError}</p>}
      {verification && (
        <p className={`mb-6 text-sm ${verification.ok ? "text-success" : "text-danger"}`}>
          {verification.ok
            ? `Chain intact across all ${verification.entries.toLocaleString()} entries. Nothing has been altered since it was written.`
            : `Chain is broken at entry #${verification.brokenAtSeq} - that entry (or one before it) doesn't match what was recorded at write time.`}
        </p>
      )}

      {error && <p className="text-sm text-danger">{error}</p>}
      {entries === null && !error && <p className="py-10 text-sm text-ink-faint">Loading...</p>}
      {entries !== null && entries.length === 0 && (
        <p className="py-10 text-sm text-ink-faint">No activity recorded yet.</p>
      )}

      {entries !== null && entries.length > 0 && (
        <ul className="divide-y divide-line border-t border-line">
          {entries.map((entry) => {
            const hasMetadata = Object.keys(entry.metadata).length > 0;
            const isOpen = expanded.has(entry.seq);
            return (
              <li key={entry.seq} className="py-3">
                <button
                  type="button"
                  onClick={() => hasMetadata && toggle(entry.seq)}
                  className={`flex w-full items-baseline justify-between gap-4 text-left ${
                    hasMetadata ? "cursor-pointer" : "cursor-default"
                  }`}
                >
                  <span className="text-sm font-semibold text-ink">{describe(entry)}</span>
                  <span className="shrink-0 text-xs text-ink-faint">
                    {new Date(entry.createdAt).toLocaleString()}
                  </span>
                </button>
                <p className="mt-0.5 text-sm text-ink-muted">
                  {actorLabel(entry)}
                  {entry.ipAddress ? ` · ${entry.ipAddress}` : ""}
                </p>
                {hasMetadata && isOpen && (
                  <pre className="mt-2 overflow-x-auto bg-page p-3 text-xs text-ink-muted">
                    {JSON.stringify(entry.metadata, null, 2)}
                  </pre>
                )}
              </li>
            );
          })}
        </ul>
      )}

      {hasMore && entries && entries.length > 0 && (
        <div className="mt-6">
          <Button
            fullWidth={false}
            onClick={() => load(entries[entries.length - 1].seq)}
            disabled={loadingMore}
          >
            {loadingMore ? "Loading..." : "Load more"}
          </Button>
        </div>
      )}
    </div>
  );
}
