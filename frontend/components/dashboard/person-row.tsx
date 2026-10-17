"use client";

import { useState } from "react";
import {
  ApiError,
  createInvite,
  revokeInvite,
  disableAccount,
  reactivateAccount,
  InvitableRole,
  Person,
} from "@/lib/api";
import { Badge } from "./badge";

export function PersonRow({
  person,
  currentUserId,
  onChange,
}: {
  person: Person;
  currentUserId: string;
  onChange: () => void;
}) {
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

  async function handleDisable() {
    if (!window.confirm(`Disable ${person.name}'s account? They'll be signed out immediately.`)) return;
    setError(null);
    setBusy(true);
    try {
      await disableAccount(person.id);
      onChange();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to disable.");
      setBusy(false);
    }
  }

  async function handleReactivate() {
    setError(null);
    setBusy(true);
    try {
      await reactivateAccount(person.id);
      onChange();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to reactivate.");
      setBusy(false);
    }
  }

  const isPending = person.status === "PENDING_VERIFICATION";
  const isActive = person.status === "ACTIVE";
  const isDisabled = person.status === "DISABLED";
  const isSelf = person.id === currentUserId;

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

      {isActive && !isSelf && (
        <button onClick={handleDisable} disabled={busy} className="text-sm font-semibold text-danger hover:underline disabled:opacity-60">
          Disable
        </button>
      )}

      {isDisabled && (
        <button onClick={handleReactivate} disabled={busy} className="text-sm font-semibold text-accent hover:underline disabled:opacity-60">
          Reactivate
        </button>
      )}
    </li>
  );
}
