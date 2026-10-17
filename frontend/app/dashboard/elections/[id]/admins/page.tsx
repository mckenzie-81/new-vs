"use client";

import { useState } from "react";
import { unassignAdmin, ApiError } from "@/lib/api";
import { useDashboardUser } from "@/components/dashboard/user-context";
import { useElection } from "@/components/dashboard/election-context";
import { AdminHeader } from "@/components/dashboard/admin-header";
import { AssignAdminForm } from "@/components/dashboard/assign-admin-form";
import { Modal } from "@/components/ui/modal";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Button } from "@/components/ui/button";

export default function AssignedAdminsPage() {
  const user = useDashboardUser();
  const { election, assignedAdmins, refresh } = useElection();
  const [assignOpen, setAssignOpen] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [confirmUnassign, setConfirmUnassign] = useState<{ userId: string; name: string } | null>(
    null
  );

  const canManage = user.role === "SYSTEMADMIN";

  async function handleUnassign() {
    if (!confirmUnassign) return;
    setError(null);
    setBusy(true);
    try {
      await unassignAdmin(election.id, confirmUnassign.userId);
      refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to unassign.");
      setBusy(false);
      setConfirmUnassign(null);
    }
  }

  return (
    <div>
      <AdminHeader
        title="Assigned admins"
        intro="Admins with day-to-day access to this election's ballot."
        actions={
          canManage ? (
            <Button fullWidth={false} onClick={() => setAssignOpen(true)}>
              Assign admin
            </Button>
          ) : undefined
        }
      />

      {assignedAdmins.length === 0 ? (
        <p className="py-10 text-sm text-ink-faint">No admins assigned yet.</p>
      ) : (
        <ul className="divide-y divide-line border-t border-line">
          {assignedAdmins.map((a) => (
            <li key={a.userId} className="flex items-center justify-between gap-4 py-3">
              <div>
                <div className="text-sm font-semibold text-ink">{a.name}</div>
                <div className="text-sm text-ink-muted">{a.email}</div>
              </div>
              {canManage && (
                <button
                  onClick={() => setConfirmUnassign({ userId: a.userId, name: a.name })}
                  className="text-sm font-semibold text-danger hover:underline"
                >
                  Unassign
                </button>
              )}
            </li>
          ))}
        </ul>
      )}
      {error && <p className="mt-2 text-sm text-danger">{error}</p>}

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

      <ConfirmDialog
        open={confirmUnassign !== null}
        title="Unassign this admin?"
        description={`${confirmUnassign?.name} will lose access to this election's ballot.`}
        confirmLabel="Unassign"
        danger
        busy={busy}
        onConfirm={handleUnassign}
        onCancel={() => setConfirmUnassign(null)}
      />
    </div>
  );
}
