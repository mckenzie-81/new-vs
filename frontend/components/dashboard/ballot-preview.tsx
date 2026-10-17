"use client";

import { useEffect, useState } from "react";
import { DndContext } from "@dnd-kit/core";
import { listPortfolios, Election, Portfolio } from "@/lib/api";
import { PortfolioCard } from "./portfolio-card";

export function BallotPreview({ election }: { election: Election }) {
  const [portfolios, setPortfolios] = useState<Portfolio[] | null>(null);

  useEffect(() => {
    listPortfolios(election.id)
      .then(setPortfolios)
      .catch(() => setPortfolios([]));
  }, [election.id]);

  return (
    <>
      <div className="border-t border-line pt-4">
        <h3 className="text-xl font-semibold text-ink">{election.title}</h3>
        {election.description && (
          <p className="mt-1 text-base text-ink-muted">{election.description}</p>
        )}
      </div>

      {portfolios === null && <p className="py-10 text-sm text-ink-faint">Loading...</p>}

      {portfolios !== null && portfolios.length === 0 && (
        <p className="py-10 text-sm text-ink-faint">No portfolios yet.</p>
      )}

      {portfolios !== null && portfolios.length > 0 && (
        <div className="max-h-[55vh] overflow-y-auto">
          {/* canEdit is always false here, so nothing is ever actually
              draggable - this DndContext exists only because PortfolioCard's
              candidate list needs an ambient one to render at all. */}
          <DndContext onDragEnd={() => {}}>
            {portfolios.map((portfolio) => (
              <PortfolioCard
                key={portfolio.id}
                electionId={election.id}
                portfolio={portfolio}
                canEdit={false}
                onChange={() => {}}
                size="comfortable"
              />
            ))}
          </DndContext>
        </div>
      )}
    </>
  );
}
