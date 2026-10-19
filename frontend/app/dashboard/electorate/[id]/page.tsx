"use client";

import { useEffect, useState, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import {
  getElectorateDatabase,
  listElectorateVoters,
  deleteElectorateDatabase,
  getDistinctValues,
  getDuplicateVoters,
  removeVoters,
  ApiError,
  ElectorateDatabase,
  Voter,
  DuplicatePair,
} from "@/lib/api";
import { Breadcrumbs } from "@/components/dashboard/breadcrumbs";
import { VoterUploadForm } from "@/components/dashboard/voter-upload-form";
import { Modal } from "@/components/ui/modal";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Button, SecondaryButton } from "@/components/ui/button";
import { Select } from "@/components/ui/select";

// The backend's own ceiling for one page of voters (see
// internal/electorate/handlers.go) - requesting this much up front means
// "select all" on a database under this size really does select every
// voter, not just a smaller default page.
const FETCH_LIMIT = 1000;

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

  const [filter, setFilter] = useState<Record<string, string>>({});
  const [distinctValues, setDistinctValues] = useState<Record<string, string[]>>({});
  const [search, setSearch] = useState("");

  const [selectMode, setSelectMode] = useState(false);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [confirmRemoveOpen, setConfirmRemoveOpen] = useState(false);
  const [removeError, setRemoveError] = useState<string | null>(null);

  const [duplicates, setDuplicates] = useState<DuplicatePair[] | null>(null);
  const [duplicatesOpen, setDuplicatesOpen] = useState(false);
  const [duplicatesError, setDuplicatesError] = useState<string | null>(null);

  const refresh = useCallback(() => {
    getElectorateDatabase(id)
      .then(setDatabase)
      .catch(() => setNotFound(true));
  }, [id]);

  const refreshVoters = useCallback(() => {
    listElectorateVoters(id, { ...filter, limit: String(FETCH_LIMIT) })
      .then((v) => {
        setVoters(v);
        setSelected(new Set());
      })
      .catch(() => setVoters([]));
  }, [id, filter]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  useEffect(() => {
    refreshVoters();
  }, [refreshVoters]);

  const filterableColumns = (database?.columns ?? []).filter(
    (c) =>
      c.toLowerCase() !== "name" &&
      c.toLowerCase() !== "email" &&
      !database?.sensitiveColumns.some((s) => s.toLowerCase() === c.toLowerCase())
  );

  // Every filterable column's distinct values are loaded up front, not
  // lazily on interaction - the filter control lives in that column's
  // own header now, always visible, Excel-style, so there's no "pick a
  // column first" step left to hang the fetch off of.
  useEffect(() => {
    filterableColumns.forEach((column) => {
      if (distinctValues[column]) return;
      getDistinctValues(id, column)
        .then((values) => setDistinctValues((prev) => ({ ...prev, [column]: values })))
        .catch(() => setDistinctValues((prev) => ({ ...prev, [column]: [] })));
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, database?.columns]);

  function setColumnFilter(column: string, value: string) {
    const next = { ...filter };
    if (value === "") delete next[column];
    else next[column] = value;
    setFilter(next);
  }

  // A plain substring search across every loaded voter's data, client
  // side - separate from the per-column exact-value filters above,
  // which only ever narrow by a fixed value. This is for finding one
  // person by typing part of their name, email, or anything else in
  // their row.
  const displayedVoters = (voters ?? []).filter((voter) => {
    if (!search.trim()) return true;
    const needle = search.trim().toLowerCase();
    return Object.values(voter.data).some((value) => value.toLowerCase().includes(needle));
  });

  function toggleSelected(voterId: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(voterId)) next.delete(voterId);
      else next.add(voterId);
      return next;
    });
  }

  function toggleSelectAll() {
    setSelected((prev) =>
      prev.size === displayedVoters.length ? new Set() : new Set(displayedVoters.map((v) => v.id))
    );
  }

  function toggleSelectMode() {
    setSelectMode((prev) => !prev);
    setSelected(new Set());
  }

  async function handleRemoveSelected() {
    setRemoveError(null);
    setBusy(true);
    try {
      await removeVoters(id, Array.from(selected));
      setConfirmRemoveOpen(false);
      refresh();
      refreshVoters();
    } catch (err) {
      setRemoveError(err instanceof ApiError ? err.message : "Failed to remove voters.");
    } finally {
      setBusy(false);
    }
  }

  function handleCheckDuplicates() {
    setDuplicatesOpen(true);
    setDuplicatesError(null);
    if (duplicates !== null) return;
    getDuplicateVoters(id)
      .then(setDuplicates)
      .catch((err) =>
        setDuplicatesError(err instanceof ApiError ? err.message : "Failed to check for duplicates.")
      );
  }

  async function handleRemoveOne(voterId: string) {
    setBusy(true);
    try {
      await removeVoters(id, [voterId]);
      setDuplicates((prev) => prev?.filter((p) => p.voterAId !== voterId && p.voterBId !== voterId) ?? null);
      refresh();
      refreshVoters();
    } catch (err) {
      setDuplicatesError(err instanceof ApiError ? err.message : "Failed to remove voter.");
    } finally {
      setBusy(false);
    }
  }

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

  const allSelected = displayedVoters.length > 0 && selected.size === displayedVoters.length;

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
          <SecondaryButton onClick={handleCheckDuplicates}>Check for duplicates</SecondaryButton>
          <SecondaryButton onClick={toggleSelectMode}>
            {selectMode ? "Cancel" : "Select"}
          </SecondaryButton>
          <SecondaryButton onClick={() => setConfirmDeleteOpen(true)} tone="danger">
            Delete
          </SecondaryButton>
          <Button fullWidth={false} onClick={() => setUploadOpen(true)}>
            Upload voters
          </Button>
        </div>
      </div>

      {deleteError && <p className="mb-6 text-sm text-danger">{deleteError}</p>}

      {voters.length > 0 && (
        <div className="mb-4">
          <input
            type="search"
            placeholder="Search by name, email, or anything else in a row..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="block min-h-11 w-full max-w-sm border-0 border-b border-line bg-transparent px-0 py-2 text-base text-ink outline-none placeholder:text-ink-faint focus:border-ink"
          />
        </div>
      )}

      {selected.size > 0 && (
        <div className="mb-4 flex items-center justify-between gap-4 border-l-2 border-danger bg-danger-soft py-2 pl-4">
          <p className="text-sm text-danger">
            {selected.size} {selected.size === 1 ? "voter" : "voters"} selected
          </p>
          <SecondaryButton tone="danger" onClick={() => setConfirmRemoveOpen(true)} disabled={busy}>
            Remove selected
          </SecondaryButton>
        </div>
      )}
      {removeError && <p className="mb-4 text-sm text-danger">{removeError}</p>}

      {voters.length === 0 ? (
        <p className="py-10 text-sm text-ink-faint">
          {Object.keys(filter).length > 0
            ? "No voters match this filter."
            : "No voters yet. Upload a CSV to get started."}
        </p>
      ) : (
        <div className="overflow-x-auto border-t border-line">
          {database.sensitiveColumns.length > 0 && (
            <p className="py-2 text-sm text-ink-faint">
              {database.sensitiveColumns.join(", ")} {database.sensitiveColumns.length === 1 ? "is" : "are"} hashed and shown masked below.
            </p>
          )}
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-line">
                {selectMode && (
                  <th className="w-8 py-2 pr-2 align-bottom">
                    <input type="checkbox" checked={allSelected} onChange={toggleSelectAll} aria-label="Select all" />
                  </th>
                )}
                {database.columns.map((col) => {
                  const isFilterable = filterableColumns.includes(col);
                  return (
                    <th key={col} className="whitespace-nowrap py-2 pr-6 align-bottom font-semibold text-ink">
                      <div>{col}</div>
                      {isFilterable && (
                        <Select
                          value={filter[col] ?? ""}
                          onChange={(e) => setColumnFilter(col, e.target.value)}
                          className="mt-1 min-h-8 py-0 text-xs font-normal"
                        >
                          <option value="">All</option>
                          {(distinctValues[col] ?? []).map((v) => (
                            <option key={v} value={v}>
                              {v}
                            </option>
                          ))}
                        </Select>
                      )}
                    </th>
                  );
                })}
              </tr>
            </thead>
            <tbody>
              {displayedVoters.map((voter) => (
                <tr key={voter.id} className="border-b border-line">
                  {selectMode && (
                    <td className="py-2 pr-2">
                      <input
                        type="checkbox"
                        checked={selected.has(voter.id)}
                        onChange={() => toggleSelected(voter.id)}
                        aria-label={`Select ${voter.data[database.columns[0]] ?? "voter"}`}
                      />
                    </td>
                  )}
                  {database.columns.map((col) => (
                    <td key={col} className="whitespace-nowrap py-2 pr-6 text-ink-muted">
                      {voter.data[col] ?? ""}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
          {displayedVoters.length === 0 && (
            <p className="py-6 text-sm text-ink-faint">No voters match &quot;{search}&quot;.</p>
          )}
          {voters.length >= FETCH_LIMIT && (
            <p className="py-3 text-sm text-ink-faint">
              Showing the first {FETCH_LIMIT} voters. Remove this batch and reload to work through
              the rest.
            </p>
          )}
        </div>
      )}

      {uploadOpen && (
        <Modal title="Upload voters" onClose={() => setUploadOpen(false)}>
          <VoterUploadForm
            database={database}
            onUploaded={() => {
              refresh();
              refreshVoters();
            }}
          />
        </Modal>
      )}

      {duplicatesOpen && (
        <Modal title="Possible duplicate voters" size="lg" onClose={() => setDuplicatesOpen(false)}>
          <p className="text-sm text-ink-muted">
            Pairs with the exact same name or email, case and spacing aside. This is a heuristic
            for you to judge, not proof either row is wrong - two real people can share a name;
            nothing is ever removed automatically.
          </p>

          {duplicatesError && <p className="mt-3 text-sm text-danger">{duplicatesError}</p>}
          {!duplicatesError && duplicates === null && (
            <p className="mt-3 text-sm text-ink-faint">Checking...</p>
          )}
          {duplicates && duplicates.length === 0 && (
            <p className="mt-3 text-sm text-ink-faint">No likely duplicates found.</p>
          )}

          {duplicates && duplicates.length > 0 && (
            <ul className="mt-4 flex flex-col gap-3">
              {duplicates.map((pair) => (
                <li key={`${pair.voterAId}-${pair.voterBId}`} className="border-t border-line pt-3">
                  <div className="flex flex-wrap items-center justify-between gap-3">
                    <div className="text-sm">
                      <p className="text-xs font-semibold uppercase tracking-wide text-ink-faint">
                        Matched on {pair.matchedOn}
                      </p>
                      <p className="mt-1 text-ink">
                        <strong>{pair.voterAName}</strong>{" "}
                        <span className="text-ink-faint">{pair.voterAEmail}</span>
                      </p>
                      <p className="mt-0.5 text-ink">
                        <strong>{pair.voterBName}</strong>{" "}
                        <span className="text-ink-faint">{pair.voterBEmail}</span>
                      </p>
                    </div>
                    <div className="flex shrink-0 gap-2">
                      <SecondaryButton
                        tone="danger"
                        disabled={busy}
                        onClick={() => handleRemoveOne(pair.voterAId)}
                      >
                        Remove first
                      </SecondaryButton>
                      <SecondaryButton
                        tone="danger"
                        disabled={busy}
                        onClick={() => handleRemoveOne(pair.voterBId)}
                      >
                        Remove second
                      </SecondaryButton>
                    </div>
                  </div>
                </li>
              ))}
            </ul>
          )}
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

      <ConfirmDialog
        open={confirmRemoveOpen}
        title={`Remove ${selected.size} ${selected.size === 1 ? "voter" : "voters"}?`}
        description="They'll no longer be eligible for any election linked to this database. This can't be undone, and only works while this database isn't linked to an election past draft."
        confirmLabel="Remove"
        danger
        busy={busy}
        onConfirm={handleRemoveSelected}
        onCancel={() => setConfirmRemoveOpen(false)}
      />
    </div>
  );
}
