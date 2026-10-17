"use client";

import { useEffect, useState, useCallback } from "react";
import { useParams } from "next/navigation";
import {
  getElection,
  listPortfolios,
  listAssignedAdmins,
  unassignAdmin,
  ApiError,
  Election,
  Portfolio,
  AdminAssignment,
} from "@/lib/api";
import { useDashboardUser } from "@/components/dashboard/user-context";
import { AdminHeader } from "@/components/dashboard/admin-header";
import { Breadcrumbs } from "@/components/dashboard/breadcrumbs";
import { Badge } from "@/components/dashboard/badge";
import { PortfolioCard } from "@/components/dashboard/portfolio-card";
import { PortfolioForm } from "@/components/dashboard/portfolio-form";
import { AssignAdminForm } from "@/components/dashboard/assign-admin-form";
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
  const [assignedAdmins, setAssignedAdmins] = useState<AdminAssignment[] | null>(null);
  const [addPortfolioOpen, setAddPortfolioOpen] = useState(false);
  const [assignOpen, setAssignOpen] = useState(false);
  const [assignError, setAssignError] = useState<string | null>(null);
  const [notFound, setNotFound] = useState(false);

  const refresh = useCallback(() => {
    getElection(id)
      .then(setElection)
      .catch(() => setNotFound(true));
    listPortfolios(id)
      .then(setPortfolios)
      .catch(() => setPortfolios([]));
    listAssignedAdmins(id)
      .then(setAssignedAdmins)
      .catch(() => setAssignedAdmins([]));
  }, [id]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  async function handleUnassign(userId: string, name: string) {
    if (!window.confirm(`Unassign ${name} from this election?`)) return;
    setAssignError(null);
    try {
      await unassignAdmin(id, userId);
      refresh();
    } catch (err) {
      setAssignError(err instanceof ApiError ? err.message : "Failed to unassign.");
    }
  }

  if (notFound) {
    return <p className="py-10 text-sm text-ink-faint">No such election.</p>;
  }

  if (!election || assignedAdmins === null) {
    return <p className="py-10 text-sm text-ink-faint">Loading...</p>;
  }

  const isAssignedAdmin =
    user.role === "ADMIN" && assignedAdmins.some((a) => a.userId === user.id);
  const canEditBallot =
    election.status === "DRAFT" && (user.role === "SYSTEMADMIN" || isAssignedAdmin);
  const canManageAssignments = user.role === "SYSTEMADMIN";

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
        title="Assigned admins"
        intro="Admins with day-to-day access to this election's ballot."
        actions={
          canManageAssignments ? (
            <Button className="w-auto px-5" onClick={() => setAssignOpen(true)}>
              Assign admin
            </Button>
          ) : undefined
        }
      />

      {assignedAdmins.length === 0 ? (
        <p className="py-6 text-sm text-ink-faint">No admins assigned yet.</p>
      ) : (
        <ul className="divide-y divide-line border-t border-line">
          {assignedAdmins.map((a) => (
            <li key={a.userId} className="flex items-center justify-between gap-4 py-3">
              <div>
                <div className="text-sm font-semibold text-ink">{a.name}</div>
                <div className="text-sm text-ink-muted">{a.email}</div>
              </div>
              {canManageAssignments && (
                <button
                  onClick={() => handleUnassign(a.userId, a.name)}
                  className="text-sm font-semibold text-danger hover:underline"
                >
                  Unassign
                </button>
              )}
            </li>
          ))}
        </ul>
      )}
      {assignError && <p className="mt-2 text-sm text-danger">{assignError}</p>}

      <div className="mt-12">
        <AdminHeader
          title="Ballot"
          intro={
            canEditBallot
              ? "Portfolios and candidates. Locked once this election leaves draft."
              : "Portfolios and candidates."
          }
          actions={
            canEditBallot ? (
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
                canEdit={canEditBallot}
                onChange={refresh}
              />
            ))}
          </div>
        )}
      </div>

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

      {assignOpen && (
        <Modal title="Assign an admin" onClose={() => setAssignOpen(false)}>
          <AssignAdminForm
            electionId={election.id}
            excludeUserIds={assignedAdmins.map((a) => a.userId)}
            onAssigned={() => {
              setAssignOpen(false);
              refresh();
            }}
          />
        </Modal>
      )}
    </div>
  );
}
