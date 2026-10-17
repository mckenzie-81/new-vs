"use client";

import { useEffect, useState } from "react";
import { listPortfolios, Election, Portfolio, Candidate } from "@/lib/api";
import { Modal } from "@/components/ui/modal";
import { Button, SecondaryButton } from "@/components/ui/button";

type Selections = Record<string, string>;

function CandidateOption({
  number,
  candidate,
  selected,
  onSelect,
}: {
  number: number;
  candidate: Candidate;
  selected: boolean;
  onSelect: () => void;
}) {
  return (
    <button
      onClick={onSelect}
      className={`flex w-full items-center gap-4 border p-3 text-left ${
        selected ? "border-accent bg-accent-soft" : "border-line hover:border-ink"
      }`}
    >
      <span className="w-6 shrink-0 text-center text-base font-semibold text-ink-faint">
        {number}
      </span>
      {candidate.photoUrl ? (
        // next/image needs a fixed set of allowed hosts; the host here
        // isn't decided yet (Cloudinary vs. our own backend), so a plain
        // img tag avoids hardcoding one prematurely.
        // eslint-disable-next-line @next/next/no-img-element
        <img
          src={candidate.photoUrl}
          alt=""
          className="h-20 w-20 shrink-0 bg-page object-cover"
        />
      ) : (
        <div className="flex h-20 w-20 shrink-0 items-center justify-center bg-page text-2xl font-semibold text-ink-faint">
          {candidate.fullName.trim().charAt(0).toUpperCase() || "?"}
        </div>
      )}
      <span className="text-lg font-medium text-ink">{candidate.fullName}</span>
    </button>
  );
}

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
                  {candidate ? candidate.fullName : "Abstained"}
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
            <h3 className="text-xl font-semibold text-ink">{p.title}</h3>
            <p className="mt-0.5 text-sm text-ink-faint">
              Tap a candidate to select them. Tap again, or leave blank, to abstain.
            </p>
            <div className="mt-4 flex flex-col gap-2">
              {p.candidates.map((c, index) => (
                <CandidateOption
                  key={c.id}
                  number={index + 1}
                  candidate={c}
                  selected={selections[p.id] === c.id}
                  onSelect={() => choose(p.id, c.id)}
                />
              ))}
            </div>
          </div>
        ))}
      </div>

      <Button variant="accent" className="mt-5" onClick={() => setScreen("review")}>
        Review selections
      </Button>
    </Modal>
  );
}
