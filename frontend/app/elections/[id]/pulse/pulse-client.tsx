"use client";

import { useEffect, useState } from "react";
import { getElectionPulse, ApiError, PulseSnapshot } from "@/lib/api";
import { CenteredMessage } from "@/components/layout/centered-card";
import { Badge } from "@/components/dashboard/badge";
import { StatRow, Stat } from "@/components/dashboard/stat-row";
import { TurnoutChart } from "@/components/pulse/turnout-chart";
import { AbstentionList } from "@/components/pulse/abstention-list";

const POLL_INTERVAL_MS = 15_000;

function formatCountdown(targetIso: string, now: number) {
  const diff = new Date(targetIso).getTime() - now;
  if (diff <= 0) return null;
  const totalMinutes = Math.floor(diff / 60_000);
  const days = Math.floor(totalMinutes / (24 * 60));
  const hours = Math.floor((totalMinutes % (24 * 60)) / 60);
  const minutes = totalMinutes % 60;
  if (days > 0) return `${days}d ${hours}h`;
  if (hours > 0) return `${hours}h ${minutes}m`;
  if (minutes > 0) return `${minutes}m`;
  return "less than a minute";
}

export function PulseClient({ electionId }: { electionId: string }) {
  const [snapshot, setSnapshot] = useState<PulseSnapshot | null>(null);
  const [notFound, setNotFound] = useState(false);
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    let stopped = false;

    function load() {
      if (document.visibilityState !== "visible") return;
      getElectionPulse(electionId)
        .then((data) => {
          if (!stopped) setSnapshot(data);
        })
        .catch((err) => {
          if (!stopped && err instanceof ApiError && err.status === 404) setNotFound(true);
        });
    }

    load();
    const pollId = window.setInterval(load, POLL_INTERVAL_MS);
    return () => {
      stopped = true;
      window.clearInterval(pollId);
    };
  }, [electionId]);

  useEffect(() => {
    const tickId = window.setInterval(() => setNow(Date.now()), 1_000);
    return () => window.clearInterval(tickId);
  }, []);

  if (notFound) {
    return <CenteredMessage>This election isn&apos;t available.</CenteredMessage>;
  }

  if (!snapshot) {
    return <CenteredMessage>Loading...</CenteredMessage>;
  }

  const { election } = snapshot;
  const turnoutPct =
    snapshot.eligibleCount > 0
      ? Math.round((snapshot.votedCount / snapshot.eligibleCount) * 100)
      : 0;

  const notStarted = now < new Date(election.startTime).getTime();
  const countdownTarget = notStarted ? election.startTime : election.endTime;
  const countdown = formatCountdown(countdownTarget, now);
  const timeLeftLabel = countdown
    ? `${notStarted ? "Opens in " : "Closes in "}${countdown}`
    : notStarted
      ? "Opening now"
      : "Closed";

  const stats: Stat[] = [
    { label: "Eligible voters", value: snapshot.eligibleCount.toLocaleString() },
    { label: "Voted so far", value: snapshot.votedCount.toLocaleString() },
    { label: "Turnout", value: `${turnoutPct}%` },
    { label: notStarted ? "Starts" : "Time left", value: timeLeftLabel },
  ];

  return (
    <div className="mx-auto max-w-3xl px-6 py-12">
      <div className="flex items-center gap-2">
        <h1 className="text-2xl font-bold tracking-tight">{election.title}</h1>
        <Badge status={election.status} />
      </div>
      {election.description && (
        <p className="mt-2 max-w-2xl text-base text-ink-muted">{election.description}</p>
      )}
      <p className="mt-4 text-xs text-ink-faint">
        Turnout and participation only - candidate standings stay confidential until voting
        closes. Updated {new Date(snapshot.refreshedAt).toLocaleTimeString()}.
      </p>

      <div className="mt-8 border-t border-line pt-6">
        <StatRow stats={stats} />
      </div>

      <div className="mt-10 border-t border-line pt-6">
        <TurnoutChart points={snapshot.turnoutSeries} />
      </div>

      <div className="mt-10 border-t border-line pt-6">
        <AbstentionList portfolios={snapshot.portfolioAbstention} />
      </div>
    </div>
  );
}
