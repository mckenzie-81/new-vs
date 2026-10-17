"use client";

import { useEffect, useState } from "react";
import { listPortfolios, Election, Portfolio } from "@/lib/api";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { PortfolioCard } from "./portfolio-card";

export function SubmitPreviewModal({
  election,
  busy,
  error,
  onClose,
  onConfirm,
}: {
  election: Election;
  busy: boolean;
  error?: string | null;
  onClose: () => void;
  onConfirm: () => void;
}) {
  const [portfolios, setPortfolios] = useState<Portfolio[] | null>(null);

  useEffect(() => {
    listPortfolios(election.id)
      .then(setPortfolios)
      .catch(() => setPortfolios([]));
  }, [election.id]);

  return (
    <Modal title="Review before submitting" size="lg" onClose={onClose}>
      <p className="mb-5 text-sm text-ink-muted">
        This is exactly what the approver will see. Once submitted, the ballot is locked until
        it&apos;s approved or rejected.
      </p>

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
        </div>
      )}

      {error && <p className="mt-4 text-sm text-danger">{error}</p>}

      <Button
        variant="accent"
        onClick={onConfirm}
        disabled={busy || portfolios === null}
        className="mt-5"
      >
        {busy ? "Submitting..." : "Submit for approval"}
      </Button>
    </Modal>
  );
}
