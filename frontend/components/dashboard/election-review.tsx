"use client";

import { useEffect, useState } from "react";
import { getElectionElectorate, Election, ElectorateSummary, AdminAssignment } from "@/lib/api";
import { BallotPreview } from "./ballot-preview";

function formatDateTime(iso: string) {
  return new Date(iso).toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

// A placeholder shaped roughly like the content it stands in for, so the
// layout doesn't visibly jump once the real summary arrives - better
// anticipation than a bare "Loading...".
function SkeletonLine({ width = "w-40" }: { width?: string }) {
  return <div className={`h-4 ${width} animate-pulse rounded-sm bg-line`} />;
}

// Everything a human - admin before submitting, approver before approving -
// should actually look at, bundled in one place rather than scattered
// across tabs nobody's prompted to visit.
export function ElectionReview({
  election,
  assignedAdmins,
}: {
  election: Election;
  assignedAdmins: AdminAssignment[];
}) {
  const [summary, setSummary] = useState<ElectorateSummary | null>(null);
  // A one-time snapshot taken when this review opens, not a live clock -
  // "has this already passed" shouldn't flicker while someone's reading it.
  const [now] = useState(() => Date.now());

  useEffect(() => {
    getElectionElectorate(election.id)
      .then(setSummary)
      .catch(() =>
        setSummary({ linked: false, databaseId: null, databaseName: null, filter: {}, eligibleCount: 0 })
      );
  }, [election.id]);

  const endTime = new Date(election.endTime).getTime();
  const startTime = new Date(election.startTime).getTime();
  const alreadyEnded = endTime <= now;
  const alreadyStarted = !alreadyEnded && startTime <= now;

  const filterEntries = summary && summary.linked ? Object.entries(summary.filter) : [];

  return (
    <div className="flex flex-col gap-6">
      <div>
        <p className="text-xs font-semibold uppercase tracking-wide text-ink-faint">Schedule</p>
        <p className="mt-1 text-sm font-medium text-ink">
          {formatDateTime(election.startTime)} – {formatDateTime(election.endTime)}
        </p>
        {alreadyEnded && (
          <p className="mt-2 border-l-2 border-danger bg-danger-soft py-2 pl-3 text-sm text-danger">
            This election&apos;s end time has already passed. Approving it now would move it
            straight to closed without anyone able to vote - fix the schedule first.
          </p>
        )}
        {alreadyStarted && (
          <p className="mt-2 border-l-2 border-warning bg-warning-soft py-2 pl-3 text-sm text-warning">
            This election&apos;s start time has already passed - voting will open immediately
            once approved.
          </p>
        )}
      </div>

      <div>
        <p className="text-xs font-semibold uppercase tracking-wide text-ink-faint">Electorate</p>
        {summary === null ? (
          <div className="mt-2 flex flex-col gap-2">
            <SkeletonLine width="w-56" />
            <SkeletonLine width="w-40" />
          </div>
        ) : !summary.linked ? (
          <p className="mt-1 text-sm text-danger">No electorate database linked.</p>
        ) : (
          <div className="mt-1 text-sm text-ink-muted">
            <p>
              <span className="font-medium text-ink">{summary.databaseName}</span> -{" "}
              {summary.eligibleCount} eligible {summary.eligibleCount === 1 ? "voter" : "voters"}
            </p>
            <p className="mt-1">
              {filterEntries.length === 0
                ? "Everyone in the database is eligible."
                : filterEntries.map(([field, value]) => `${field} = ${value}`).join(", ")}
            </p>
            {election.voterVerificationColumns.length > 0 && (
              <p className="mt-1">
                Voters must also confirm: {election.voterVerificationColumns.join(", ")}
              </p>
            )}
          </div>
        )}
      </div>

      <div>
        <p className="text-xs font-semibold uppercase tracking-wide text-ink-faint">
          Assigned admins
        </p>
        <p className="mt-1 text-sm text-ink-muted">
          {assignedAdmins.length === 0
            ? "None assigned."
            : assignedAdmins.map((a) => a.name).join(", ")}
        </p>
      </div>

      <div>
        <p className="mb-1 text-xs font-semibold uppercase tracking-wide text-ink-faint">
          Ballot
        </p>
        <BallotPreview election={election} />
      </div>
    </div>
  );
}
