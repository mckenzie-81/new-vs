"use client";

import { useState, useEffect } from "react";
import { listPortfolios, Portfolio } from "@/lib/api";
import { useDashboardUser } from "@/components/dashboard/user-context";
import { useElection } from "@/components/dashboard/election-context";
import { AdminHeader } from "@/components/dashboard/admin-header";
import { PortfolioCard } from "@/components/dashboard/portfolio-card";
import { PortfolioForm } from "@/components/dashboard/portfolio-form";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";

export default function BallotPage() {
  const user = useDashboardUser();
  const { election, assignedAdmins } = useElection();
  const [portfolios, setPortfolios] = useState<Portfolio[] | null>(null);
  const [addPortfolioOpen, setAddPortfolioOpen] = useState(false);

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

  return (
    <div>
      <AdminHeader
        title="Ballot"
        intro={
          canEditBallot
            ? "Portfolios and candidates. Locked once this election leaves draft."
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
        <div>
          {portfolios.map((portfolio) => (
            <PortfolioCard
              key={portfolio.id}
              electionId={election.id}
              portfolio={portfolio}
              canEdit={canEditBallot}
              onChange={refreshPortfolios}
            />
          ))}
        </div>
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
