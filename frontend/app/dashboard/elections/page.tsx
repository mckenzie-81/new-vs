"use client";

import { useEffect, useState, useCallback } from "react";
import { listElections, Election } from "@/lib/api";
import { useDashboardUser } from "@/components/dashboard/user-context";
import { AdminHeader } from "@/components/dashboard/admin-header";
import { Breadcrumbs } from "@/components/dashboard/breadcrumbs";
import { ElectionRow } from "@/components/dashboard/election-row";
import { ElectionForm } from "@/components/dashboard/election-form";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";

export default function ElectionsPage() {
  const user = useDashboardUser();
  const [elections, setElections] = useState<Election[] | null>(null);
  const [createOpen, setCreateOpen] = useState(false);

  const refresh = useCallback(() => {
    listElections()
      .then(setElections)
      .catch(() => setElections([]));
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return (
    <div>
      <Breadcrumbs items={[{ label: "Dashboard", href: "/dashboard" }, { label: "Elections" }]} />
      <AdminHeader
        title="Elections"
        intro="Create, approve, and run elections."
        actions={
          user.role === "SYSTEMADMIN" ? (
            <Button fullWidth={false} onClick={() => setCreateOpen(true)}>
              Create election
            </Button>
          ) : undefined
        }
      />

      {elections === null && <p className="py-10 text-sm text-ink-faint">Loading...</p>}

      {elections !== null && elections.length === 0 && (
        <p className="py-10 text-sm text-ink-faint">No elections yet.</p>
      )}

      {elections !== null && elections.length > 0 && (
        <ul className="divide-y divide-line border-t border-line">
          {elections.map((election) => (
            <ElectionRow key={election.id} election={election} />
          ))}
        </ul>
      )}

      {createOpen && (
        <Modal title="Create an election" onClose={() => setCreateOpen(false)}>
          <ElectionForm
            onSaved={() => {
              setCreateOpen(false);
              refresh();
            }}
          />
        </Modal>
      )}
    </div>
  );
}
