"use client";

import { useState } from "react";
import { ApiError, deleteCandidate, deletePortfolio, Portfolio } from "@/lib/api";
import { Modal } from "@/components/ui/modal";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { CandidateForm } from "./candidate-form";

export function PortfolioCard({
  electionId,
  portfolio,
  canEdit,
  onChange,
}: {
  electionId: string;
  portfolio: Portfolio;
  canEdit: boolean;
  onChange: () => void;
}) {
  const [addCandidateOpen, setAddCandidateOpen] = useState(false);
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
          <h3 className="text-sm font-semibold text-ink">{portfolio.title}</h3>
          {portfolio.description && (
            <p className="mt-0.5 text-sm text-ink-muted">{portfolio.description}</p>
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
        <ul className="mt-3 flex flex-col gap-2">
          {portfolio.candidates.map((candidate) => (
            <li key={candidate.id} className="flex items-start justify-between gap-4">
              <div>
                <div className="text-sm font-medium text-ink">{candidate.fullName}</div>
                {candidate.manifesto && (
                  <div className="text-sm text-ink-muted">{candidate.manifesto}</div>
                )}
              </div>
              {canEdit && (
                <button
                  onClick={() =>
                    setConfirmDeleteCandidate({ id: candidate.id, name: candidate.fullName })
                  }
                  className="shrink-0 text-sm text-ink-faint hover:text-danger"
                >
                  Remove
                </button>
              )}
            </li>
          ))}
        </ul>
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
            onCreated={() => {
              setAddCandidateOpen(false);
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
