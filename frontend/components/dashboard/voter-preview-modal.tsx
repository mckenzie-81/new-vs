"use client";

import { useEffect, useState } from "react";
import { listPortfolios, candidateTicketName, Election, Portfolio } from "@/lib/api";
import { Modal } from "@/components/ui/modal";
import { Button, SecondaryButton } from "@/components/ui/button";
import { CandidateOption, candidateListClassName } from "@/components/ballot/candidate-option";
import { PortfolioBanner } from "@/components/ballot/portfolio-banner";

type Selections = Record<string, string>;

export function VoterPreviewModal({
  election,
  onClose,
}: {
  election: Election;
  onClose: () => void;
}) {
  const [portfolios, setPortfolios] = useState<Portfolio[] | null>(null);
  const [selections, setSelections] = useState<Selections>({});
  const [screen, setScreen] = useState<"select" | "review" | "done">("select");

  useEffect(() => {
    listPortfolios(election.id)
      .then(setPortfolios)
      .catch(() => setPortfolios([]));
  }, [election.id]);

  // Selecting the already-selected candidate again clears it - abstain
  // isn't a choice you make, it's just what no selection means.
  function choose(portfolioId: string, candidateId: string) {
    setSelections((prev) => {
      const next = { ...prev };
      if (next[portfolioId] === candidateId) {
        delete next[portfolioId];
      } else {
        next[portfolioId] = candidateId;
      }
      return next;
    });
  }

  function resetPortfolio(portfolioId: string) {
    setSelections((prev) => {
      const next = { ...prev };
      delete next[portfolioId];
      return next;
    });
  }

  const disclaimer =
    "Preview only - this simulates the voting experience design. Nothing here is submitted or recorded.";

  if (screen === "done") {
    return (
      <Modal title="Preview complete" onClose={onClose}>
        <p className="text-sm text-ink-muted">{disclaimer}</p>
        <Button className="mt-5" onClick={onClose}>
          Close
        </Button>
      </Modal>
    );
  }

  if (screen === "review") {
    return (
      <Modal title="Review your selections" size="lg" onClose={onClose}>
        <p className="mb-5 text-sm text-ink-muted">{disclaimer}</p>
        <ul className="divide-y divide-line border-t border-line">
          {portfolios?.map((p) => {
            const candidate = p.candidates.find((c) => c.id === selections[p.id]);
            return (
              <li key={p.id} className="py-3">
                <div className="text-sm font-semibold text-ink">{p.title}</div>
                <div className="text-sm text-ink-muted">
                  {candidate ? candidateTicketName(candidate) : "Abstained"}
                </div>
              </li>
            );
          })}
        </ul>
        <div className="mt-5 flex gap-3">
          <SecondaryButton onClick={() => setScreen("select")}>Back</SecondaryButton>
          <Button variant="accent" fullWidth={false} onClick={() => setScreen("done")}>
            Submit (preview)
          </Button>
        </div>
      </Modal>
    );
  }

  return (
    <Modal title="Voter preview" size="lg" onClose={onClose}>
      <p className="mb-5 text-sm text-ink-muted">{disclaimer}</p>

      {portfolios === null && <p className="py-10 text-sm text-ink-faint">Loading...</p>}

      <div className="max-h-[55vh] overflow-y-auto">
        {portfolios?.map((p) => (
          <div key={p.id} className="border-t border-line py-5 first:border-t-0">
            <PortfolioBanner
              title={p.title}
              onReset={selections[p.id] ? () => resetPortfolio(p.id) : undefined}
            />
            <div className={`mt-4 ${candidateListClassName}`}>
              {p.candidates.map((c) => (
                <CandidateOption
                  key={c.id}
                  candidate={c}
                  selected={selections[p.id] === c.id}
                  onSelect={() => choose(p.id, c.id)}
                  style={election.ballotStyle}
                />
              ))}
            </div>
          </div>
        ))}
      </div>

      <div className="mt-5 flex gap-3">
        <SecondaryButton
          onClick={() => setSelections({})}
          disabled={Object.keys(selections).length === 0}
        >
          Reset selections
        </SecondaryButton>
        <Button variant="accent" fullWidth={false} onClick={() => setScreen("review")}>
          Review selections
        </Button>
      </div>
    </Modal>
  );
}
