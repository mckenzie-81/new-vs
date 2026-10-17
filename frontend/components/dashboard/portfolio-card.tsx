"use client";

import { useState } from "react";
import {
  DndContext,
  closestCenter,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
} from "@dnd-kit/core";
import { SortableContext, verticalListSortingStrategy, arrayMove } from "@dnd-kit/sortable";
import { ApiError, deleteCandidate, deletePortfolio, reorderCandidates, Candidate, Portfolio } from "@/lib/api";
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
  const [candidates, setCandidates] = useState(portfolio.candidates);
  const [prevPortfolioCandidates, setPrevPortfolioCandidates] = useState(portfolio.candidates);
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 8 } }));

  // Local state exists only for optimistic drag reordering; when the parent
  // hands us a genuinely new candidates array (after onChange() refetches),
  // adopt it. Comparing during render rather than in an effect is the
  // documented way to do this - see "Adjusting state when a prop changes"
  // in the React docs.
  if (portfolio.candidates !== prevPortfolioCandidates) {
    setPrevPortfolioCandidates(portfolio.candidates);
    setCandidates(portfolio.candidates);
  }

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

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event;
    if (!over || active.id === over.id) return;

    const oldIndex = candidates.findIndex((c) => c.id === active.id);
    const newIndex = candidates.findIndex((c) => c.id === over.id);
    const reordered = arrayMove(candidates, oldIndex, newIndex);
    setCandidates(reordered);

    reorderCandidates(
      electionId,
      portfolio.id,
      reordered.map((c, i) => ({ id: c.id, order: i }))
    )
      .then(onChange)
      .catch(onChange);
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

      {candidates.length === 0 ? (
        <p className="mt-3 text-sm text-ink-faint">No candidates yet.</p>
      ) : (
        <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
          <SortableContext items={candidates.map((c) => c.id)} strategy={verticalListSortingStrategy}>
            <ul className={size === "comfortable" ? "mt-4 flex flex-col gap-3" : "mt-3 flex flex-col gap-2"}>
              {candidates.map((candidate) => (
                <SortableItem key={candidate.id} id={candidate.id} disabled={!canEdit} as="li">
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
                        {candidate.fullName}
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
        </DndContext>
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
