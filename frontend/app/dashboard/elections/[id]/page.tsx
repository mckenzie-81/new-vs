"use client";

import { useState, useEffect } from "react";
import {
  DndContext,
  closestCenter,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
} from "@dnd-kit/core";
import { SortableContext, verticalListSortingStrategy, arrayMove } from "@dnd-kit/sortable";
import { listPortfolios, reorderPortfolios, reorderCandidates, Portfolio } from "@/lib/api";
import { useDashboardUser } from "@/components/dashboard/user-context";
import { useElection } from "@/components/dashboard/election-context";
import { AdminHeader } from "@/components/dashboard/admin-header";
import { PortfolioCard } from "@/components/dashboard/portfolio-card";
import { PortfolioForm } from "@/components/dashboard/portfolio-form";
import { SortableItem } from "@/components/dashboard/sortable-item";
import { VoterPreviewModal } from "@/components/dashboard/voter-preview-modal";
import { EyeIcon } from "@heroicons/react/24/outline";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { Tooltip } from "@/components/ui/tooltip";

export default function BallotPage() {
  const user = useDashboardUser();
  const { election, assignedAdmins } = useElection();
  const [portfolios, setPortfolios] = useState<Portfolio[] | null>(null);
  const [addPortfolioOpen, setAddPortfolioOpen] = useState(false);
  const [previewOpen, setPreviewOpen] = useState(false);
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 8 } }));

  const refreshPortfolios = () => {
    listPortfolios(election.id)
      .then(setPortfolios)
      .catch(() => setPortfolios([]));
  };

  useEffect(() => {
    refreshPortfolios();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [election.id]);

  const isAssignedAdmin =
    user.role === "ADMIN" && assignedAdmins.some((a) => a.userId === user.id);
  const canEditBallot =
    election.status === "DRAFT" && (user.role === "SYSTEMADMIN" || isAssignedAdmin);

  // One DndContext handles both portfolio reordering and candidate
  // reordering within a portfolio - nesting a second, independent
  // DndContext inside PortfolioCard used to conflict with this one and
  // broke both dragging and ordinary clicks nearby. active.data.current
  // (set via SortableItem's data prop) says which kind of drag this is.
  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event;
    if (!over || active.id === over.id || !portfolios) return;

    const type = active.data.current?.type;

    if (type === "candidate") {
      const portfolioId = active.data.current?.portfolioId as string;
      const portfolio = portfolios.find((p) => p.id === portfolioId);
      if (!portfolio) return;

      const oldIndex = portfolio.candidates.findIndex((c) => c.id === active.id);
      const newIndex = portfolio.candidates.findIndex((c) => c.id === over.id);
      if (oldIndex === -1 || newIndex === -1) return;
      const reorderedCandidates = arrayMove(portfolio.candidates, oldIndex, newIndex);

      setPortfolios(
        portfolios.map((p) => (p.id === portfolioId ? { ...p, candidates: reorderedCandidates } : p))
      );

      reorderCandidates(
        election.id,
        portfolioId,
        reorderedCandidates.map((c, i) => ({ id: c.id, order: i }))
      ).catch(() => refreshPortfolios());
      return;
    }

    const oldIndex = portfolios.findIndex((p) => p.id === active.id);
    const newIndex = portfolios.findIndex((p) => p.id === over.id);
    if (oldIndex === -1 || newIndex === -1) return;
    const reordered = arrayMove(portfolios, oldIndex, newIndex);
    setPortfolios(reordered);

    reorderPortfolios(
      election.id,
      reordered.map((p, i) => ({ id: p.id, order: i }))
    ).catch(() => refreshPortfolios());
  }

  return (
    <div>
      <AdminHeader
        title="Ballot"
        intro={
          canEditBallot
            ? "Portfolios and candidates. Drag to reorder. Locked once this election leaves draft."
            : "Portfolios and candidates."
        }
        actions={
          <>
            <Tooltip label="Preview ballot">
              <button
                aria-label="Preview ballot"
                onClick={() => setPreviewOpen(true)}
                className="p-2.5 text-ink-faint hover:text-ink"
              >
                <EyeIcon className="h-5 w-5" />
              </button>
            </Tooltip>
            {canEditBallot && (
              <Button fullWidth={false} onClick={() => setAddPortfolioOpen(true)}>
                Add portfolio
              </Button>
            )}
          </>
        }
      />

      {portfolios === null && <p className="py-10 text-sm text-ink-faint">Loading...</p>}

      {portfolios !== null && portfolios.length === 0 && (
        <p className="py-10 text-sm text-ink-faint">No portfolios yet.</p>
      )}

      {portfolios !== null && portfolios.length > 0 && (
        <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
          <SortableContext
            items={portfolios.map((p) => p.id)}
            strategy={verticalListSortingStrategy}
          >
            <div>
              {portfolios.map((portfolio) => (
                <SortableItem
                  key={portfolio.id}
                  id={portfolio.id}
                  disabled={!canEditBallot}
                  data={{ type: "portfolio" }}
                >
                  <PortfolioCard
                    electionId={election.id}
                    portfolio={portfolio}
                    canEdit={canEditBallot}
                    onChange={refreshPortfolios}
                  />
                </SortableItem>
              ))}
            </div>
          </SortableContext>
        </DndContext>
      )}

      {addPortfolioOpen && (
        <Modal title="Add a portfolio" onClose={() => setAddPortfolioOpen(false)}>
          <PortfolioForm
            electionId={election.id}
            onCreated={() => {
              setAddPortfolioOpen(false);
              refreshPortfolios();
            }}
          />
        </Modal>
      )}

      {previewOpen && (
        <VoterPreviewModal election={election} onClose={() => setPreviewOpen(false)} />
      )}
    </div>
  );
}
