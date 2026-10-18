import { PortfolioResult } from "@/lib/api";
import { PortfolioResults } from "@/components/ballot/portfolio-results";

function VerifyingBanner() {
  return (
    <div className="border-l-2 border-accent bg-accent-soft px-4 py-5">
      <p className="text-sm font-semibold text-ink">Results are being verified</p>
      <p className="mt-1.5 text-sm text-ink-muted">
        Final standings are being cross-checked before they&apos;re published here. Check back
        soon.
      </p>
    </div>
  );
}

export function ResultsSection({
  published,
  portfolios,
}: {
  published: boolean;
  portfolios?: PortfolioResult[];
}) {
  if (!published || !portfolios) {
    return (
      <div>
        <h3 className="text-sm font-semibold uppercase tracking-wide text-ink-faint">Results</h3>
        <div className="mt-3">
          <VerifyingBanner />
        </div>
      </div>
    );
  }

  return (
    <div>
      <h3 className="text-sm font-semibold uppercase tracking-wide text-ink-faint">Results</h3>
      <div className="mt-3 flex flex-col gap-8">
        {portfolios.map((p) => (
          <PortfolioResults key={p.portfolioId} portfolio={p} />
        ))}
      </div>
    </div>
  );
}
