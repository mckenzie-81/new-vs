"use client";

import { useEffect, useState, useCallback } from "react";
import { useParams, usePathname, useRouter } from "next/navigation";
import Link from "next/link";
import {
  getElection,
  listAssignedAdmins,
  submitForApproval,
  approveElection,
  deleteElection,
  archiveElection,
  ApiError,
  Election,
  AdminAssignment,
} from "@/lib/api";
import { TrashIcon, PencilIcon } from "@heroicons/react/24/outline";
import { useDashboardUser } from "@/components/dashboard/user-context";
import { ElectionProvider } from "@/components/dashboard/election-context";
import { Breadcrumbs } from "@/components/dashboard/breadcrumbs";
import { Badge } from "@/components/dashboard/badge";
import { Button, SecondaryButton } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Modal } from "@/components/ui/modal";
import { RejectElectionForm } from "@/components/dashboard/reject-election-form";
import { SubmitPreviewModal } from "@/components/dashboard/submit-preview-modal";
import { ElectionForm } from "@/components/dashboard/election-form";

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

const tabs = [
  { label: "Ballot", href: "" },
  { label: "Assigned admins", href: "/admins" },
  { label: "Results", href: "/results" },
];

export function ElectionLayoutClient({ children }: { children: React.ReactNode }) {
  const { id } = useParams<{ id: string }>();
  const pathname = usePathname();
  const router = useRouter();
  const user = useDashboardUser();

  const [election, setElection] = useState<Election | null>(null);
  const [assignedAdmins, setAssignedAdmins] = useState<AdminAssignment[] | null>(null);
  const [notFound, setNotFound] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false);
  const [confirmApproveOpen, setConfirmApproveOpen] = useState(false);
  const [rejectOpen, setRejectOpen] = useState(false);
  const [submitPreviewOpen, setSubmitPreviewOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);

  const refresh = useCallback(() => {
    getElection(id)
      .then(setElection)
      .catch(() => setNotFound(true));
    listAssignedAdmins(id)
      .then(setAssignedAdmins)
      .catch(() => setAssignedAdmins([]));
  }, [id]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  async function handleSubmit() {
    setActionError(null);
    setBusy(true);
    try {
      const updated = await submitForApproval(id);
      setElection(updated);
      setSubmitPreviewOpen(false);
    } catch (err) {
      setActionError(err instanceof ApiError ? err.message : "Failed to submit.");
    } finally {
      setBusy(false);
    }
  }

  async function handleDelete() {
    setActionError(null);
    setBusy(true);
    try {
      await deleteElection(id);
      router.push("/dashboard/elections");
    } catch (err) {
      setActionError(err instanceof ApiError ? err.message : "Failed to delete.");
      setBusy(false);
      setConfirmDeleteOpen(false);
    }
  }

  async function handleApprove() {
    setActionError(null);
    setBusy(true);
    try {
      const updated = await approveElection(id);
      setElection(updated);
    } catch (err) {
      setActionError(err instanceof ApiError ? err.message : "Failed to approve.");
    } finally {
      setBusy(false);
      setConfirmApproveOpen(false);
    }
  }

  async function handleArchive() {
    setActionError(null);
    setBusy(true);
    try {
      const updated = await archiveElection(id);
      setElection(updated);
    } catch (err) {
      setActionError(err instanceof ApiError ? err.message : "Failed to archive.");
    } finally {
      setBusy(false);
    }
  }

  if (notFound) {
    return <p className="py-10 text-sm text-ink-faint">No such election.</p>;
  }

  if (!election || assignedAdmins === null) {
    return <p className="py-10 text-sm text-ink-faint">Loading...</p>;
  }

  const base = `/dashboard/elections/${id}`;
  const isSystemadmin = user.role === "SYSTEMADMIN";
  const isApprover = user.role === "APPROVER";
  const isAssignedAdmin =
    user.role === "ADMIN" && assignedAdmins.some((a) => a.userId === user.id);
  const canSubmit = isSystemadmin || isAssignedAdmin;

  return (
    <div>
      <Breadcrumbs
        items={[
          { label: "Dashboard", href: "/dashboard" },
          { label: "Elections", href: "/dashboard/elections" },
          { label: election.title },
        ]}
      />

      {election.status === "DRAFT" && election.rejectionReason && (
        <div className="mb-6 border-l-2 border-danger bg-page py-3 pl-4">
          <p className="text-sm font-semibold text-danger">Rejected</p>
          <p className="mt-1 text-sm text-ink-muted">{election.rejectionReason}</p>
        </div>
      )}

      <div className="mb-6 flex flex-wrap items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-2xl font-bold tracking-tight sm:text-3xl">{election.title}</h1>
            <Badge status={election.status} />
          </div>
          {election.description && (
            <p className="mt-2 max-w-2xl text-base text-ink-muted">{election.description}</p>
          )}
          <p className="mt-1 text-sm font-medium text-ink-muted">
            {formatRange(election.startTime, election.endTime)}
          </p>
        </div>

        {election.status === "DRAFT" && canSubmit && (
          <div className="flex flex-wrap gap-3">
            <button
              aria-label="Edit election"
              onClick={() => setEditOpen(true)}
              disabled={busy}
              className="p-2.5 text-ink-faint hover:text-ink disabled:opacity-60"
            >
              <PencilIcon className="h-5 w-5" />
            </button>
            {isSystemadmin && (
              <button
                aria-label="Delete election"
                onClick={() => setConfirmDeleteOpen(true)}
                disabled={busy}
                className="p-2.5 text-ink-faint hover:text-danger disabled:opacity-60"
              >
                <TrashIcon className="h-5 w-5" />
              </button>
            )}
            {canSubmit && (
              <Button
                variant="accent"
                onClick={() => setSubmitPreviewOpen(true)}
                disabled={busy}
                fullWidth={false}
              >
                Submit for approval
              </Button>
            )}
          </div>
        )}

        {isSystemadmin && election.status === "CLOSED" && (
          <SecondaryButton onClick={handleArchive} disabled={busy}>
            Archive
          </SecondaryButton>
        )}

        {isApprover && election.status === "PENDING_APPROVAL" && (
          <div className="flex flex-wrap gap-3">
            <SecondaryButton onClick={() => setRejectOpen(true)} disabled={busy} tone="danger">
              Reject
            </SecondaryButton>
            <Button variant="accent" onClick={() => setConfirmApproveOpen(true)} disabled={busy} fullWidth={false}>
              Approve
            </Button>
          </div>
        )}
      </div>

      {actionError && <p className="mb-6 text-sm text-danger">{actionError}</p>}

      <div className="mb-8 flex gap-6 border-b border-line">
        {tabs.map((tab) => {
          const href = base + tab.href;
          const active = pathname === href;
          return (
            <Link
              key={tab.href}
              href={href}
              className={`-mb-px border-b-2 pb-3 text-sm font-semibold ${
                active ? "border-accent text-ink" : "border-transparent text-ink-muted hover:text-ink"
              }`}
            >
              {tab.label}
            </Link>
          );
        })}
      </div>

      <ElectionProvider value={{ election, assignedAdmins, refresh }}>{children}</ElectionProvider>

      <ConfirmDialog
        open={confirmDeleteOpen}
        title="Delete this election?"
        description={`"${election.title}" and its entire ballot will be permanently removed. This can't be undone.`}
        confirmLabel="Delete"
        danger
        busy={busy}
        onConfirm={handleDelete}
        onCancel={() => setConfirmDeleteOpen(false)}
      />

      <ConfirmDialog
        open={confirmApproveOpen}
        title="Approve this election?"
        description={`"${election.title}" will move to approved and can go live at its scheduled start time.`}
        confirmLabel="Approve"
        busy={busy}
        onConfirm={handleApprove}
        onCancel={() => setConfirmApproveOpen(false)}
      />

      {rejectOpen && (
        <Modal title="Reject this election" onClose={() => setRejectOpen(false)}>
          <RejectElectionForm
            electionId={id}
            onRejected={() => {
              setRejectOpen(false);
              refresh();
            }}
          />
        </Modal>
      )}

      {submitPreviewOpen && (
        <SubmitPreviewModal
          election={election}
          busy={busy}
          error={actionError}
          onClose={() => setSubmitPreviewOpen(false)}
          onConfirm={handleSubmit}
        />
      )}

      {editOpen && (
        <Modal title="Edit election" onClose={() => setEditOpen(false)}>
          <ElectionForm
            election={election}
            onSaved={() => {
              setEditOpen(false);
              refresh();
            }}
          />
        </Modal>
      )}
    </div>
  );
}
