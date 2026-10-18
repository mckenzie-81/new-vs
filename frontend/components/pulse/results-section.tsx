import { PortfolioResult } from "@/lib/api";
import { PortfolioResults } from "@/components/ballot/portfolio-results";

function VerifyingBanner() {
  return (
    <div className="verify-aurora p-1">
      <div className="verify-aurora-surface relative z-10 flex items-center gap-3 bg-surface px-4 py-5">
        <span className="verify-orb h-5 w-5 shrink-0" aria-hidden />
        <div>
          <p className="text-sm font-semibold text-ink">Results are being verified</p>
          <p className="mt-1.5 text-sm text-ink-muted">
            Final standings are being cross-checked before they&apos;re published here. Check
            back soon.
          </p>
        </div>
      </div>
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
