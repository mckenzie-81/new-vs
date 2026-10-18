"use client";

import { useState } from "react";
import { TurnoutPoint } from "@/lib/api";

function formatBucket(iso: string) {
  return new Date(iso).toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

// Single-series columns showing ballots cast per time bucket. Hover or
// focus a column for its exact value; a table view covers anyone who
// can't or doesn't want to use either.
export function TurnoutChart({ points }: { points: TurnoutPoint[] }) {
  const [active, setActive] = useState<number | null>(null);
  const [table, setTable] = useState(false);

  if (points.length === 0) {
    return <p className="py-4 text-sm text-ink-faint">No ballots cast yet.</p>;
  }

  const max = Math.max(1, ...points.map((p) => p.count));
  const shown = points[active ?? points.length - 1];

  return (
    <figure>
      <div className="flex flex-wrap items-baseline justify-between gap-4">
        <figcaption className="text-sm font-semibold uppercase tracking-wide text-ink-faint">
          Turnout over time
        </figcaption>
        <button
          type="button"
          onClick={() => setTable(!table)}
          className="min-h-11 text-sm font-semibold text-ink-muted underline-offset-4 hover:underline"
        >
          {table ? "View as chart" : "View as table"}
        </button>
      </div>

      {table ? (
        <table className="mt-3 w-full text-left text-sm text-ink">
          <tbody>
            {points.map((p) => (
              <tr key={p.bucketStart} className="border-t border-line">
                <td className="py-1.5">{formatBucket(p.bucketStart)}</td>
                <td className="py-1.5 text-right tabular-nums">{p.count} ballots</td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : (
        <>
          <p className="mt-1 text-sm text-ink-muted" aria-live="polite">
            {formatBucket(shown.bucketStart)}:{" "}
            <span className="font-semibold text-ink tabular-nums">{shown.count}</span> ballots
          </p>
          <div
            className="relative mt-4 h-40 border-b border-line"
            onMouseLeave={() => setActive(null)}
          >
            <span className="absolute -top-2 right-0 text-xs text-ink-faint tabular-nums">
              {max}
            </span>
            <div className="absolute inset-0 flex items-end gap-[2px]">
              {points.map((p, i) => (
                <button
                  key={p.bucketStart}
                  type="button"
                  aria-label={`${formatBucket(p.bucketStart)}: ${p.count} ballots`}
                  onMouseEnter={() => setActive(i)}
                  onFocus={() => setActive(i)}
                  className="flex h-full flex-1 items-end justify-center"
                >
                  <span
                    className={`block w-full max-w-6 ${
                      p === shown ? "bg-accent" : "bg-accent-soft"
                    }`}
                    style={{ height: `${(p.count / max) * 100}%`, minHeight: p.count > 0 ? 2 : 0 }}
                  />
                </button>
              ))}
            </div>
          </div>
          <div className="mt-1.5 flex justify-between text-xs text-ink-faint">
            <span>{formatBucket(points[0].bucketStart)}</span>
            <span>{formatBucket(points[points.length - 1].bucketStart)}</span>
          </div>
        </>
      )}
    </figure>
  );
}
