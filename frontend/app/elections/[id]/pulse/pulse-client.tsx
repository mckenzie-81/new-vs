"use client";

import { useEffect, useState } from "react";
import { getElectionPulse, getBranding, ApiError, PulseSnapshot } from "@/lib/api";
import { CenteredMessage } from "@/components/layout/centered-card";
import { Badge } from "@/components/dashboard/badge";
import { StatRow, Stat } from "@/components/dashboard/stat-row";
import { TurnoutChart } from "@/components/pulse/turnout-chart";
import { AbstentionList } from "@/components/pulse/abstention-list";
import { ResultsSection } from "@/components/pulse/results-section";
import { ThemeScope } from "@/components/branding/theme-scope";

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
  const [systemAccentColor, setSystemAccentColor] = useState<string | null>(null);
  const [systemLogoUrl, setSystemLogoUrl] = useState<string | null>(null);

  useEffect(() => {
    getBranding()
      .then((settings) => {
        setSystemAccentColor(settings.accentColor);
        setSystemLogoUrl(settings.logoUrl);
      })
      .catch(() => {});
  }, []);

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
  const closingSoon =
    !notStarted &&
    election.status === "LIVE" &&
    new Date(election.endTime).getTime() - now < 60 * 60_000;

  const stats: Stat[] = [
    { label: "Eligible voters", value: snapshot.eligibleCount.toLocaleString() },
    { label: "Voted so far", value: snapshot.votedCount.toLocaleString() },
    { label: "Turnout", value: `${turnoutPct}%`, tone: "accent" },
    {
      label: notStarted ? "Starts" : "Time left",
      value: timeLeftLabel,
      tone: closingSoon ? "danger" : undefined,
    },
  ];

  const effectiveAccentColor = election.accentColor ?? systemAccentColor;
  const effectiveLogoUrl = election.logoUrl ?? systemLogoUrl;

  return (
    <ThemeScope accentColor={effectiveAccentColor}>
    <div className="mx-auto max-w-3xl px-6 py-12">
      {effectiveLogoUrl && (
        // eslint-disable-next-line @next/next/no-img-element
        <img src={effectiveLogoUrl} alt="" className="mb-4 h-10 object-contain" />
      )}
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

      {(election.status === "CLOSED" || election.status === "ARCHIVED") && (
        <div className="mt-10 border-t border-line pt-6">
          <ResultsSection published={snapshot.resultsPublished} portfolios={snapshot.results} />
        </div>
      )}

      {snapshot.scheduleChanges.length > 0 && (
        <div className="mt-10 border-t border-line pt-6">
          <h3 className="text-sm font-semibold uppercase tracking-wide text-ink-faint">
            Schedule changes
          </h3>
          <ul className="mt-3 flex flex-col gap-3">
            {snapshot.scheduleChanges.map((c) => (
              <li key={c.createdAt} className="text-sm">
                <p className="text-ink-muted">
                  <span className="font-medium text-ink">
                    {new Date(c.createdAt).toLocaleString()}
                  </span>{" "}
                  — end time changed from {new Date(c.previousEndTime).toLocaleString()} to{" "}
                  {new Date(c.newEndTime).toLocaleString()}.
                </p>
                <p className="mt-0.5 text-ink-faint">Reason: {c.reason}</p>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
    </ThemeScope>
  );
}
