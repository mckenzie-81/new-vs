"use client";

import { useState } from "react";
import {
  ApiError,
  createInvite,
  revokeInvite,
  disableAccount,
  reactivateAccount,
  resetUserTOTP,
  InvitableRole,
  Person,
} from "@/lib/api";
import { Badge } from "./badge";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";

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
  const [confirmAction, setConfirmAction] = useState<"revoke" | "disable" | "reset-totp" | null>(null);

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
    setError(null);
    setBusy(true);
    try {
      await revokeInvite(person.id);
      onChange();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to revoke.");
      setBusy(false);
      setConfirmAction(null);
    }
  }

  async function handleDisable() {
    setError(null);
    setBusy(true);
    try {
      await disableAccount(person.id);
      onChange();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to disable.");
      setBusy(false);
      setConfirmAction(null);
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

  async function handleResetTOTP() {
    setError(null);
    setBusy(true);
    try {
      await resetUserTOTP(person.id);
      onChange();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to reset 2FA.");
      setBusy(false);
      setConfirmAction(null);
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
          <button onClick={() => setConfirmAction("revoke")} disabled={busy} className="text-danger hover:underline disabled:opacity-60">
            Revoke
          </button>
        </div>
      )}

      {isActive && (
        <div className="flex items-center gap-4 text-sm font-semibold">
          {person.totpEnabled && !isSelf && (
            <button
              onClick={() => setConfirmAction("reset-totp")}
              disabled={busy}
              className="text-ink-muted hover:text-ink disabled:opacity-60"
            >
              Reset 2FA
            </button>
          )}
          {!isSelf && (
            <button onClick={() => setConfirmAction("disable")} disabled={busy} className="text-danger hover:underline disabled:opacity-60">
              Disable
            </button>
          )}
        </div>
      )}

      {isDisabled && (
        <button onClick={handleReactivate} disabled={busy} className="text-sm font-semibold text-accent hover:underline disabled:opacity-60">
          Reactivate
        </button>
      )}

      <ConfirmDialog
        open={confirmAction === "revoke"}
        title="Revoke this invite?"
        description={`The invite for ${person.name} will be cancelled and the link will stop working.`}
        confirmLabel="Revoke"
        danger
        busy={busy}
        onConfirm={handleRevoke}
        onCancel={() => setConfirmAction(null)}
      />

      <ConfirmDialog
        open={confirmAction === "disable"}
        title="Disable this account?"
        description={`${person.name} will be signed out immediately and won't be able to log in until reactivated.`}
        confirmLabel="Disable"
        danger
        busy={busy}
        onConfirm={handleDisable}
        onCancel={() => setConfirmAction(null)}
      />

      <ConfirmDialog
        open={confirmAction === "reset-totp"}
        title="Reset two-factor authentication?"
        description={`${person.name} will need to set up 2FA again from scratch on their next login - for when they've lost both their authenticator device and their recovery codes.`}
        confirmLabel="Reset 2FA"
        danger
        busy={busy}
        onConfirm={handleResetTOTP}
        onCancel={() => setConfirmAction(null)}
      />
    </li>
  );
}
