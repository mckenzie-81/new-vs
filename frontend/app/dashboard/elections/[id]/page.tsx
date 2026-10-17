"use client";

import { useEffect, useState, useCallback } from "react";
import { useParams } from "next/navigation";
import { getElection, listPortfolios, Election, Portfolio } from "@/lib/api";
import { useDashboardUser } from "@/components/dashboard/user-context";
import { AdminHeader } from "@/components/dashboard/admin-header";
import { Breadcrumbs } from "@/components/dashboard/breadcrumbs";
import { Badge } from "@/components/dashboard/badge";
import { PortfolioCard } from "@/components/dashboard/portfolio-card";
import { PortfolioForm } from "@/components/dashboard/portfolio-form";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";

function formatRange(startTime: string, endTime: string) {
  const opts: Intl.DateTimeFormatOptions = {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  };
  return `${new Date(startTime).toLocaleString(undefined, opts)} – ${new Date(endTime).toLocaleString(undefined, opts)}`;
}

export default function ElectionDetailPage() {
  const { id } = useParams<{ id: string }>();
  const user = useDashboardUser();
  const [election, setElection] = useState<Election | null>(null);
  const [portfolios, setPortfolios] = useState<Portfolio[] | null>(null);
  const [addPortfolioOpen, setAddPortfolioOpen] = useState(false);
  const [notFound, setNotFound] = useState(false);

  const refresh = useCallback(() => {
    getElection(id)
      .then(setElection)
      .catch(() => setNotFound(true));
    listPortfolios(id)
      .then(setPortfolios)
      .catch(() => setPortfolios([]));
  }, [id]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  if (notFound) {
    return <p className="py-10 text-sm text-ink-faint">No such election.</p>;
  }

  if (!election) {
    return <p className="py-10 text-sm text-ink-faint">Loading...</p>;
  }

  const canEdit = user.role === "SYSTEMADMIN" && election.status === "DRAFT";

  return (
    <div>
      <Breadcrumbs
        items={[
          { label: "Dashboard", href: "/dashboard" },
          { label: "Elections", href: "/dashboard/elections" },
          { label: election.title },
        ]}
      />

      <div className="mb-8 flex flex-wrap items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-2xl font-bold tracking-tight sm:text-3xl">{election.title}</h1>
            <Badge status={election.status} />
          </div>
          {election.description && (
            <p className="mt-2 max-w-2xl text-sm text-ink-muted">{election.description}</p>
          )}
          <p className="mt-1 text-sm text-ink-faint">
            {formatRange(election.startTime, election.endTime)}
          </p>
        </div>
      </div>

      <AdminHeader
        title="Ballot"
        intro={
          canEdit
            ? "Portfolios and candidates. Locked once this election leaves draft."
            : "Portfolios and candidates."
        }
        actions={
          canEdit ? (
            <Button className="w-auto px-5" onClick={() => setAddPortfolioOpen(true)}>
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
              canEdit={canEdit}
              onChange={refresh}
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
              refresh();
            }}
          />
        </Modal>
      )}
    </div>
  );
}
