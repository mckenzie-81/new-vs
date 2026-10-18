import { PortfolioResult, candidateTicketName } from "@/lib/api";
import { CandidatePhoto } from "@/components/ballot/candidate-option";
import { PortfolioBanner } from "@/components/ballot/portfolio-banner";

// Shared between the admin's Results tab and the public pulse page's
// published standings - same tally, same presentation, so a published
// result never looks different from what the election's own admins see.
export function PortfolioResults({ portfolio: p }: { portfolio: PortfolioResult }) {
  const maxVotes = Math.max(1, ...p.candidates.map((c) => c.voteCount));
  const sorted = [...p.candidates].sort((a, b) => b.voteCount - a.voteCount);
  // Every candidate at the max is a winner - a tie doesn't pick one to
  // highlight, it highlights all of them.
  const topVotes = sorted[0]?.voteCount ?? 0;
  const hasWinner = topVotes > 0;

  return (
    <div>
      <PortfolioBanner
        title={p.title}
        trailing={<p className="text-sm text-ink-faint">{p.totalBallots} ballots cast</p>}
      />
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
              <CandidatePhoto candidate={c} className="h-14 w-12 shrink-0 text-lg" />
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
                    {c.voteCount} <span className="text-xs text-ink-faint">({share.toFixed(0)}%)</span>
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
}
