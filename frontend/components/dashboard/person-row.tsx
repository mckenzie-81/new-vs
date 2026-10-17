"use client";

import { useState } from "react";
import { ApiError, createInvite, revokeInvite, InvitableRole, Person } from "@/lib/api";
import { Badge } from "./badge";

export function PersonRow({ person, onChange }: { person: Person; onChange: () => void }) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [sent, setSent] = useState(false);

  const joined = new Date(person.createdAt).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  });

  async function handleResend() {
    setError(null);
    setBusy(true);
    try {
      await createInvite(person.name, person.email, person.role as InvitableRole);
      setSent(true);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to resend.");
    } finally {
      setBusy(false);
    }
  }

  async function handleRevoke() {
    if (!window.confirm(`Revoke the invite for ${person.name}?`)) return;
    setError(null);
    setBusy(true);
    try {
      await revokeInvite(person.id);
      onChange();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to revoke.");
      setBusy(false);
    }
  }

  const isPending = person.status === "PENDING_VERIFICATION";

  return (
    <li className="flex flex-wrap items-center justify-between gap-3 py-3">
      <div>
        <div className="flex items-center gap-2">
          <span className="text-sm font-semibold text-ink">{person.name}</span>
          <Badge status={person.status} />
        </div>
        <div className="mt-0.5 text-sm text-ink-muted">
          {person.email} · {person.role.toLowerCase()} · invited {joined}
        </div>
        {error && <div className="mt-0.5 text-sm text-danger">{error}</div>}
      </div>

      {isPending && (
        <div className="flex items-center gap-4 text-sm font-semibold">
          {sent ? (
            <span className="text-success">Sent</span>
          ) : (
            <button onClick={handleResend} disabled={busy} className="text-ink-muted hover:text-ink disabled:opacity-60">
              Resend
            </button>
          )}
          <button onClick={handleRevoke} disabled={busy} className="text-danger hover:underline disabled:opacity-60">
            Revoke
          </button>
        </div>
      )}
    </li>
  );
}
