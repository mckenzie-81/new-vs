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
import { listPortfolios, reorderPortfolios, Portfolio } from "@/lib/api";
import { useDashboardUser } from "@/components/dashboard/user-context";
import { useElection } from "@/components/dashboard/election-context";
import { AdminHeader } from "@/components/dashboard/admin-header";
import { PortfolioCard } from "@/components/dashboard/portfolio-card";
import { PortfolioForm } from "@/components/dashboard/portfolio-form";
import { SortableItem } from "@/components/dashboard/sortable-item";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";

export default function BallotPage() {
  const user = useDashboardUser();
  const { election, assignedAdmins } = useElection();
  const [portfolios, setPortfolios] = useState<Portfolio[] | null>(null);
  const [addPortfolioOpen, setAddPortfolioOpen] = useState(false);
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

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event;
    if (!over || active.id === over.id || !portfolios) return;

    const oldIndex = portfolios.findIndex((p) => p.id === active.id);
    const newIndex = portfolios.findIndex((p) => p.id === over.id);
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
          canEditBallot ? (
            <Button fullWidth={false} onClick={() => setAddPortfolioOpen(true)}>
              Add portfolio
            </Button>
          ) : undefined
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
                <SortableItem key={portfolio.id} id={portfolio.id} disabled={!canEditBallot}>
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
    </div>
  );
}
