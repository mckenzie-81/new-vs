"use client";

import { useEffect, useState, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import {
  getElectorateDatabase,
  listElectorateVoters,
  deleteElectorateDatabase,
  ApiError,
  ElectorateDatabase,
  Voter,
} from "@/lib/api";
import { Breadcrumbs } from "@/components/dashboard/breadcrumbs";
import { VoterUploadForm } from "@/components/dashboard/voter-upload-form";
import { Modal } from "@/components/ui/modal";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Button, SecondaryButton } from "@/components/ui/button";

export default function ElectorateDatabasePage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const [database, setDatabase] = useState<ElectorateDatabase | null>(null);
  const [voters, setVoters] = useState<Voter[] | null>(null);
  const [notFound, setNotFound] = useState(false);
  const [uploadOpen, setUploadOpen] = useState(false);
  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const refresh = useCallback(() => {
    getElectorateDatabase(id)
      .then(setDatabase)
      .catch(() => setNotFound(true));
    listElectorateVoters(id)
      .then(setVoters)
      .catch(() => setVoters([]));
  }, [id]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  async function handleDelete() {
    setDeleteError(null);
    setBusy(true);
    try {
      await deleteElectorateDatabase(id);
      router.push("/dashboard/electorate");
    } catch (err) {
      setDeleteError(err instanceof ApiError ? err.message : "Failed to delete.");
      setBusy(false);
      setConfirmDeleteOpen(false);
    }
  }

  if (notFound) {
    return <p className="py-10 text-sm text-ink-faint">No such electorate database.</p>;
  }

  if (!database || voters === null) {
    return <p className="py-10 text-sm text-ink-faint">Loading...</p>;
  }

  return (
    <div>
      <Breadcrumbs
        items={[
          { label: "Dashboard", href: "/dashboard" },
          { label: "Electorate", href: "/dashboard/electorate" },
          { label: database.name },
        ]}
      />

      <div className="mb-8 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight sm:text-3xl">{database.name}</h1>
          {database.description && (
            <p className="mt-2 max-w-2xl text-base text-ink-muted">{database.description}</p>
          )}
          <p className="mt-1 text-sm font-medium text-ink-muted">
            {database.voterCount} {database.voterCount === 1 ? "voter" : "voters"}
          </p>
        </div>
        <div className="flex gap-3">
          <SecondaryButton onClick={() => setConfirmDeleteOpen(true)} tone="danger">
            Delete
          </SecondaryButton>
          <Button fullWidth={false} onClick={() => setUploadOpen(true)}>
            Upload voters
          </Button>
        </div>
      </div>

      {deleteError && <p className="mb-6 text-sm text-danger">{deleteError}</p>}

      {voters.length === 0 ? (
        <p className="py-10 text-sm text-ink-faint">
          No voters yet. Upload a CSV to get started.
        </p>
      ) : (
        <div className="overflow-x-auto border-t border-line">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-line">
                {database.columns.map((col) => (
                  <th key={col} className="whitespace-nowrap py-2 pr-6 font-semibold text-ink">
                    {col}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {voters.map((voter) => (
                <tr key={voter.id} className="border-b border-line">
                  {database.columns.map((col) => (
                    <td key={col} className="whitespace-nowrap py-2 pr-6 text-ink-muted">
                      {voter.data[col] ?? ""}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
          {voters.length >= 200 && (
            <p className="py-3 text-sm text-ink-faint">Showing the first 200 voters.</p>
          )}
        </div>
      )}

      {uploadOpen && (
        <Modal title="Upload voters" onClose={() => setUploadOpen(false)}>
          <VoterUploadForm
            databaseId={id}
            onUploaded={() => {
              refresh();
            }}
          />
        </Modal>
      )}

      <ConfirmDialog
        open={confirmDeleteOpen}
        title="Delete this electorate database?"
        description={`"${database.name}" and all ${database.voterCount} voters in it will be permanently removed. This can't be undone, and only works if no election is currently linked to it.`}
        confirmLabel="Delete"
        danger
        busy={busy}
        onConfirm={handleDelete}
        onCancel={() => setConfirmDeleteOpen(false)}
      />
    </div>
  );
}
