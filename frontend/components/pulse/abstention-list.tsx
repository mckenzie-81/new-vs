import { PortfolioAbstention } from "@/lib/api";

// Ranked by abstention rate, not raw count - a derived metric computed
// here, not a field the backend hands over directly. An uncontested
// portfolio's ballot/abstain counts would be that one candidate's live
// vote count, so the backend never sends real numbers for one; this just
// renders what it's given, which is "nothing" for those.
export function AbstentionList({ portfolios }: { portfolios: PortfolioAbstention[] }) {
  const ranked = portfolios
    .filter((p) => !p.uncontested)
    .map((p) => ({
      ...p,
      rate: p.ballots && p.ballots > 0 ? (p.abstentions ?? 0) / p.ballots : 0,
    }))
    .sort((a, b) => b.rate - a.rate);
  const uncontested = portfolios.filter((p) => p.uncontested);

  if (ranked.length === 0 && uncontested.length === 0) {
    return <p className="py-4 text-sm text-ink-faint">No portfolios on this ballot.</p>;
  }

  return (
    <section>
      <h3 className="text-sm font-semibold uppercase tracking-wide text-ink-faint">
        Abstention by portfolio
      </h3>
      <ol className="mt-1">
        {ranked.map((p) => (
          <li key={p.portfolioId} className="py-2">
            <div className="flex items-baseline justify-between gap-4 text-sm">
              <span className="min-w-0 truncate text-ink">{p.title}</span>
              <span className="shrink-0 tabular-nums text-ink">
                {Math.round(p.rate * 100)}%
                <span className="text-ink-faint"> abstained · {p.ballots} ballots</span>
              </span>
            </div>
            <span aria-hidden className="mt-1.5 block h-1 bg-line">
              <span className="block h-full bg-accent" style={{ width: `${p.rate * 100}%` }} />
            </span>
          </li>
        ))}
        {uncontested.map((p) => (
          <li key={p.portfolioId} className="flex items-baseline justify-between gap-4 py-2 text-sm">
            <span className="min-w-0 truncate text-ink">{p.title}</span>
            <span className="shrink-0 text-ink-faint">Uncontested</span>
          </li>
        ))}
      </ol>
    </section>
  );
}
