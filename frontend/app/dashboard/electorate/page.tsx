"use client";

import { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { listElectorateDatabases, ElectorateDatabase } from "@/lib/api";
import { AdminHeader } from "@/components/dashboard/admin-header";
import { Breadcrumbs } from "@/components/dashboard/breadcrumbs";
import { ElectorateDatabaseForm } from "@/components/dashboard/electorate-database-form";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";

export default function ElectoratePage() {
  const [databases, setDatabases] = useState<ElectorateDatabase[] | null>(null);
  const [createOpen, setCreateOpen] = useState(false);

  const refresh = useCallback(() => {
    listElectorateDatabases()
      .then(setDatabases)
      .catch(() => setDatabases([]));
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return (
    <div>
      <Breadcrumbs items={[{ label: "Dashboard", href: "/dashboard" }, { label: "Electorate" }]} />
      <AdminHeader
        title="Electorate"
        intro="Voter databases, linked to elections at creation. The same database can back more than one election."
        actions={
          <Button fullWidth={false} onClick={() => setCreateOpen(true)}>
            Create database
          </Button>
        }
      />

      {databases === null && <p className="py-10 text-sm text-ink-faint">Loading...</p>}

      {databases !== null && databases.length === 0 && (
        <p className="py-10 text-sm text-ink-faint">No electorate databases yet.</p>
      )}

      {databases !== null && databases.length > 0 && (
        <ul className="divide-y divide-line border-t border-line">
          {databases.map((db) => (
            <li key={db.id} className="py-3">
              <Link href={`/dashboard/electorate/${db.id}`} className="group block">
                <span className="text-sm font-semibold text-ink group-hover:underline">
                  {db.name}
                </span>
                {db.description && (
                  <p className="mt-0.5 text-sm text-ink-muted">{db.description}</p>
                )}
                <p className="mt-0.5 text-sm text-ink-faint">
                  {db.voterCount} {db.voterCount === 1 ? "voter" : "voters"}
                </p>
              </Link>
            </li>
          ))}
        </ul>
      )}

      {createOpen && (
        <Modal
          title="Create an electorate database"
          onClose={() => setCreateOpen(false)}
          closeOnOverlayClick={false}
        >
          {/* onCreated only refreshes the list in the background - the
              form itself shows a "what next" confirmation after a
              successful create, same pattern as creating an election. */}
          <ElectorateDatabaseForm onCreated={refresh} />
        </Modal>
      )}
    </div>
  );
}
