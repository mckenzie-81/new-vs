"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import {
  getElectionElectorate,
  issueVoterCredentials,
  updateElection,
  electionToCreateInput,
  getDeviceSignals,
  unlockVoterCredential,
  ApiError,
  ElectorateSummary,
  IssueVoterCredentialsResult,
  DeviceSignal,
} from "@/lib/api";
import { useDashboardUser } from "@/components/dashboard/user-context";
import { useElection } from "@/components/dashboard/election-context";
import { AdminHeader } from "@/components/dashboard/admin-header";
import { Button, SecondaryButton } from "@/components/ui/button";
import { ElectorateSelector } from "@/components/dashboard/electorate-selector";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";

export default function ElectionElectoratePage() {
  const user = useDashboardUser();
  const { election, assignedAdmins, refresh } = useElection();
  const [summary, setSummary] = useState<ElectorateSummary | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [issuing, setIssuing] = useState(false);
  const [issueError, setIssueError] = useState<string | null>(null);
  const [issueResult, setIssueResult] = useState<IssueVoterCredentialsResult | null>(null);
  const [deviceSignals, setDeviceSignals] = useState<DeviceSignal[] | null>(null);
  const [unlockEmail, setUnlockEmail] = useState("");
  const [unlocking, setUnlocking] = useState(false);
  const [unlockError, setUnlockError] = useState<string | null>(null);
  const [unlockSuccess, setUnlockSuccess] = useState<string | null>(null);

  const [editing, setEditing] = useState(false);
  const [electorateDatabaseId, setElectorateDatabaseId] = useState(
    election.electorateDatabaseId ?? ""
  );
  const [electorateFilter, setElectorateFilter] = useState<Record<string, string>>(
    election.electorateFilter
  );
  const [voterVerificationColumns, setVoterVerificationColumns] = useState<string[]>(
    election.voterVerificationColumns
  );
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  function refreshSummary() {
    getElectionElectorate(election.id)
      .then(setSummary)
      .catch((err) => setError(err instanceof ApiError ? err.message : "Failed to load."));
  }

  useEffect(refreshSummary, [election.id]);

  const showDeviceSignals = election.status === "LIVE" || election.status === "CLOSED" || election.status === "ARCHIVED";

  useEffect(() => {
    if (!showDeviceSignals) return;
    getDeviceSignals(election.id)
      .then(setDeviceSignals)
      .catch(() => setDeviceSignals([]));
  }, [election.id, showDeviceSignals]);

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

  async function handleUnlock(e: React.FormEvent) {
    e.preventDefault();
    setUnlocking(true);
    setUnlockError(null);
    setUnlockSuccess(null);
    try {
      await unlockVoterCredential(election.id, unlockEmail);
      setUnlockSuccess(
        `${unlockEmail} can now verify again, including from a new device, if they haven't already voted.`
      );
      setUnlockEmail("");
    } catch (err) {
      setUnlockError(err instanceof ApiError ? err.message : "Failed to unlock this voter.");
    } finally {
      setUnlocking(false);
    }
  }

  function startEditing() {
    setElectorateDatabaseId(election.electorateDatabaseId ?? "");
    setElectorateFilter(election.electorateFilter);
    setVoterVerificationColumns(election.voterVerificationColumns);
    setSaveError(null);
    setEditing(true);
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    setSaveError(null);
    setSaving(true);
    try {
      await updateElection(election.id, {
        ...electionToCreateInput(election),
        electorateDatabaseId,
        electorateFilter,
        voterVerificationColumns,
      });
      setEditing(false);
      refresh();
      refreshSummary();
    } catch (err) {
      setSaveError(err instanceof ApiError ? err.message : "Failed to save.");
    } finally {
      setSaving(false);
    }
  }

  const canEdit =
    election.status === "DRAFT" &&
    (user.role === "SYSTEMADMIN" ||
      (user.role === "ADMIN" && assignedAdmins.some((a) => a.userId === user.id)));

  const canIssue =
    (election.status === "APPROVED" || election.status === "LIVE") &&
    (user.role === "SYSTEMADMIN" || user.role === "ADMIN");

  const filterEntries = summary ? Object.entries(summary.filter) : [];

  if (editing) {
    return (
      <div>
        <AdminHeader title="Electorate" intro="Who's eligible to vote in this election." />
        <form onSubmit={handleSave} className="flex flex-col gap-4 border-t border-line pt-6">
          <ElectorateSelector
            databaseId={electorateDatabaseId}
            filter={electorateFilter}
            verificationColumns={voterVerificationColumns}
            onDatabaseChange={setElectorateDatabaseId}
            onFilterChange={setElectorateFilter}
            onVerificationColumnsChange={setVoterVerificationColumns}
          />

          {saveError && <p className="text-sm text-danger">{saveError}</p>}

          <div className="flex gap-3">
            <SecondaryButton type="button" onClick={() => setEditing(false)} disabled={saving}>
              Cancel
            </SecondaryButton>
            <Button type="submit" fullWidth={false} disabled={saving}>
              {saving ? "Saving..." : "Save"}
            </Button>
          </div>
        </form>
      </div>
    );
  }

  return (
    <div>
      <AdminHeader
        title="Electorate"
        intro="Who's eligible to vote in this election."
        actions={
          canEdit ? <SecondaryButton onClick={startEditing}>Edit</SecondaryButton> : undefined
        }
      />

      {error && <p className="text-sm text-danger">{error}</p>}

      {!summary && !error && <p className="py-10 text-sm text-ink-faint">Loading...</p>}

      {summary && !summary.linked && (
        <p className="py-10 text-sm text-ink-faint">
          No electorate database linked yet.
          {canEdit && " Use the edit button above to add one before submitting for approval."}
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

          {election.voterVerificationColumns.length > 0 && (
            <div className="mt-6">
              <p className="text-xs font-semibold uppercase tracking-wide text-ink-faint">
                Voter verification
              </p>
              <p className="mt-1 text-sm text-ink-muted">
                {election.voterVerificationColumns.join(", ")}
              </p>
            </div>
          )}

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

          {canIssue && (
            <div className="mt-8 border-t border-line pt-6">
              <p className="text-xs font-semibold uppercase tracking-wide text-ink-faint">
                Unlock a voter
              </p>
              <p className="mt-1 text-sm text-ink-muted">
                A voting link can only be completed on the first device that successfully
                verifies it - a second device is turned away so the same link can&apos;t be
                opened twice. Use this if a voter got stuck (switched devices, cleared cookies,
                failed verification too many times) and hasn&apos;t voted yet.
              </p>
              <form onSubmit={handleUnlock} className="mt-3 flex items-end gap-3">
                <div className="flex-1">
                  <Field label="Voter's email" htmlFor="unlock-email">
                    <Input
                      id="unlock-email"
                      type="email"
                      required
                      value={unlockEmail}
                      onChange={(e) => setUnlockEmail(e.target.value)}
                    />
                  </Field>
                </div>
                <Button type="submit" fullWidth={false} disabled={unlocking}>
                  {unlocking ? "Unlocking..." : "Unlock"}
                </Button>
              </form>
              {unlockError && <p className="mt-3 text-sm text-danger">{unlockError}</p>}
              {unlockSuccess && <p className="mt-3 text-sm text-success">{unlockSuccess}</p>}
            </div>
          )}

          {showDeviceSignals && deviceSignals && deviceSignals.length > 0 && (
            <div className="mt-8 border-t border-line pt-6">
              <p className="text-xs font-semibold uppercase tracking-wide text-ink-faint">
                Device activity
              </p>
              <p className="mt-1 text-sm text-ink-muted">
                Browsers that have authenticated as more than one voter. This is informational,
                not proof of anything on its own - a shared lab or polling-station computer will
                show up here too.
              </p>
              <ul className="mt-3 divide-y divide-line border-t border-line">
                {deviceSignals.map((d) => (
                  <li key={d.deviceId} className="flex items-baseline justify-between gap-4 py-2">
                    <span className="font-mono text-xs text-ink-faint">
                      {d.deviceId.slice(0, 12)}...
                    </span>
                    <span className="text-sm text-ink-muted">
                      {d.distinctVoters} distinct voters · {d.credentialCount} attempts
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
