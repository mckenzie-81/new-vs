"use client";

import { useState } from "react";
import { SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable";
import {
  ApiError,
  deleteCandidate,
  deletePortfolio,
  candidateTicketName,
  Candidate,
  Portfolio,
} from "@/lib/api";
import { Modal } from "@/components/ui/modal";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { SortableItem } from "./sortable-item";
import { CandidateForm } from "./candidate-form";

type CardSize = "compact" | "comfortable";

const thumbnailClasses: Record<CardSize, string> = {
  compact: "h-8 w-8 text-xs",
  comfortable: "h-12 w-12 text-base",
};

function CandidateThumbnail({ candidate, size }: { candidate: Candidate; size: CardSize }) {
  if (candidate.photoUrl) {
    return (
      // next/image needs a fixed set of allowed hosts; the host here isn't
      // decided yet (Cloudinary vs. our own backend), so a plain img tag
      // avoids hardcoding one prematurely.
      // eslint-disable-next-line @next/next/no-img-element
      <img
        src={candidate.photoUrl}
        alt=""
        className={`shrink-0 bg-page object-cover ${thumbnailClasses[size]}`}
      />
    );
  }
  return (
    <div
      className={`flex shrink-0 items-center justify-center bg-page font-semibold text-ink-faint ${thumbnailClasses[size]}`}
    >
      {candidate.fullName.trim().charAt(0).toUpperCase() || "?"}
    </div>
  );
}

// Candidate drag-and-drop lives here, but the DndContext itself does not -
// this card is always rendered inside a portfolio-level DndContext owned
// by the page, and nesting a second independent DndContext here previously
// broke both dragging and ordinary clicks on Edit/Remove, since two sets
// of PointerSensor listeners ended up competing over the same DOM subtree.
// A SortableContext here is fine - dnd-kit supports nesting those under one
// shared DndContext, which is exactly the "sortable groups" pattern this is.
export function PortfolioCard({
  electionId,
  portfolio,
  canEdit,
  onChange,
  size = "compact",
}: {
  electionId: string;
  portfolio: Portfolio;
  canEdit: boolean;
  onChange: () => void;
  size?: CardSize;
}) {
  const [addCandidateOpen, setAddCandidateOpen] = useState(false);
  const [editCandidate, setEditCandidate] = useState<Candidate | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [confirmDeletePortfolio, setConfirmDeletePortfolio] = useState(false);
  const [confirmDeleteCandidate, setConfirmDeleteCandidate] = useState<{
    id: string;
    name: string;
  } | null>(null);

  async function handleDeletePortfolio() {
    setError(null);
    setBusy(true);
    try {
      await deletePortfolio(electionId, portfolio.id);
      onChange();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to delete portfolio.");
      setBusy(false);
      setConfirmDeletePortfolio(false);
    }
  }

  async function handleDeleteCandidate() {
    if (!confirmDeleteCandidate) return;
    setError(null);
    setBusy(true);
    try {
      await deleteCandidate(electionId, portfolio.id, confirmDeleteCandidate.id);
      onChange();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to remove candidate.");
      setBusy(false);
      setConfirmDeleteCandidate(null);
    }
  }

  return (
    <div className="border-t border-line py-5 first:border-t-0">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h3 className={size === "comfortable" ? "text-xl font-semibold text-ink" : "text-sm font-semibold text-ink"}>
            {portfolio.title}
          </h3>
          {portfolio.description && (
            <p
              className={
                size === "comfortable"
                  ? "mt-1 text-base text-ink-muted"
                  : "mt-0.5 text-sm text-ink-muted"
              }
            >
              {portfolio.description}
            </p>
          )}
        </div>
        {canEdit && (
          <button
            onClick={() => setConfirmDeletePortfolio(true)}
            className="text-sm font-semibold text-danger hover:underline"
          >
            Delete
          </button>
        )}
      </div>

      {error && <p className="mt-2 text-sm text-danger">{error}</p>}

      {portfolio.candidates.length === 0 ? (
        <p className="mt-3 text-sm text-ink-faint">No candidates yet.</p>
      ) : (
        <SortableContext
          items={portfolio.candidates.map((c) => c.id)}
          strategy={verticalListSortingStrategy}
        >
          <ul className={size === "comfortable" ? "mt-4 flex flex-col gap-3" : "mt-3 flex flex-col gap-2"}>
            {portfolio.candidates.map((candidate) => (
              <SortableItem
                key={candidate.id}
                id={candidate.id}
                disabled={!canEdit}
                as="li"
                data={{ type: "candidate", portfolioId: portfolio.id }}
              >
                <div className="flex items-center justify-between gap-4">
                  <div className={size === "comfortable" ? "flex items-center gap-3.5" : "flex items-center gap-2.5"}>
                    <CandidateThumbnail candidate={candidate} size={size} />
                    <span
                      className={
                        size === "comfortable"
                          ? "text-lg font-medium text-ink"
                          : "text-sm font-medium text-ink"
                      }
                    >
                      {candidateTicketName(candidate)}
                    </span>
                  </div>
                  {canEdit && (
                    <div className="flex shrink-0 items-center gap-3 text-sm">
                      <button
                        onClick={() => setEditCandidate(candidate)}
                        className="text-ink-faint hover:text-ink"
                      >
                        Edit
                      </button>
                      <button
                        onClick={() =>
                          setConfirmDeleteCandidate({ id: candidate.id, name: candidate.fullName })
                        }
                        className="text-ink-faint hover:text-danger"
                      >
                        Remove
                      </button>
                    </div>
                  )}
                </div>
              </SortableItem>
            ))}
          </ul>
        </SortableContext>
      )}

      {canEdit && (
        <button
          onClick={() => setAddCandidateOpen(true)}
          className="mt-3 text-sm font-semibold text-accent hover:underline"
        >
          + Add candidate
        </button>
      )}

      {addCandidateOpen && (
        <Modal title={`Add a candidate for ${portfolio.title}`} onClose={() => setAddCandidateOpen(false)}>
          <CandidateForm
            electionId={electionId}
            portfolioId={portfolio.id}
            runningMateRequired={portfolio.runningMateRequired}
            onSaved={() => {
              setAddCandidateOpen(false);
              onChange();
            }}
          />
        </Modal>
      )}

      {editCandidate && (
        <Modal title={`Edit ${editCandidate.fullName}`} onClose={() => setEditCandidate(null)}>
          <CandidateForm
            electionId={electionId}
            portfolioId={portfolio.id}
            runningMateRequired={portfolio.runningMateRequired}
            candidate={editCandidate}
            onSaved={() => {
              setEditCandidate(null);
              onChange();
            }}
          />
        </Modal>
      )}

      <ConfirmDialog
        open={confirmDeletePortfolio}
        title="Delete this portfolio?"
        description={`"${portfolio.title}" and all its candidates will be removed.`}
        confirmLabel="Delete"
        danger
        busy={busy}
        onConfirm={handleDeletePortfolio}
        onCancel={() => setConfirmDeletePortfolio(false)}
      />

      <ConfirmDialog
        open={confirmDeleteCandidate !== null}
        title="Remove this candidate?"
        description={`${confirmDeleteCandidate?.name} will be removed from "${portfolio.title}".`}
        confirmLabel="Remove"
        danger
        busy={busy}
        onConfirm={handleDeleteCandidate}
        onCancel={() => setConfirmDeleteCandidate(null)}
      />
    </div>
  );
}
