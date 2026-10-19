"use client";

import { useEffect, useState } from "react";
import {
  getTOTPStatus,
  beginTOTPSetup,
  confirmTOTPSetup,
  disableTOTP,
  ApiError,
  TOTPSetup,
} from "@/lib/api";
import { useDashboardUser } from "@/components/dashboard/user-context";
import { AdminHeader } from "@/components/dashboard/admin-header";
import { Breadcrumbs } from "@/components/dashboard/breadcrumbs";
import { Button, SecondaryButton } from "@/components/ui/button";
import { Modal } from "@/components/ui/modal";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { PasswordInput } from "@/components/ui/password-input";

export default function AccountPage() {
  const user = useDashboardUser();
  const [enabled, setEnabled] = useState<boolean | null>(null);
  const [statusError, setStatusError] = useState<string | null>(null);

  const [setup, setSetup] = useState<TOTPSetup | null>(null);
  const [startingSetup, setStartingSetup] = useState(false);
  const [setupError, setSetupError] = useState<string | null>(null);

  const [confirmCode, setConfirmCode] = useState("");
  const [confirming, setConfirming] = useState(false);
  const [confirmError, setConfirmError] = useState<string | null>(null);
  const [recoveryCodes, setRecoveryCodes] = useState<string[] | null>(null);

  const [disableOpen, setDisableOpen] = useState(false);
  const [disablePassword, setDisablePassword] = useState("");
  const [disabling, setDisabling] = useState(false);
  const [disableError, setDisableError] = useState<string | null>(null);

  function refreshStatus() {
    getTOTPStatus()
      .then((s) => setEnabled(s.enabled))
      .catch((err) => setStatusError(err instanceof ApiError ? err.message : "Failed to load."));
  }

  useEffect(refreshStatus, []);

  async function handleStartSetup() {
    setStartingSetup(true);
    setSetupError(null);
    try {
      setSetup(await beginTOTPSetup());
    } catch (err) {
      setSetupError(err instanceof ApiError ? err.message : "Failed to start setup.");
    } finally {
      setStartingSetup(false);
    }
  }

  async function handleConfirm(e: React.FormEvent) {
    e.preventDefault();
    setConfirming(true);
    setConfirmError(null);
    try {
      const result = await confirmTOTPSetup(confirmCode);
      setRecoveryCodes(result.recoveryCodes);
      setSetup(null);
      setEnabled(true);
    } catch (err) {
      setConfirmError(err instanceof ApiError ? err.message : "Failed to confirm.");
    } finally {
      setConfirming(false);
    }
  }

  async function handleDisable(e: React.FormEvent) {
    e.preventDefault();
    setDisabling(true);
    setDisableError(null);
    try {
      await disableTOTP(disablePassword);
      setDisableOpen(false);
      setDisablePassword("");
      setEnabled(false);
    } catch (err) {
      setDisableError(err instanceof ApiError ? err.message : "Failed to disable 2FA.");
    } finally {
      setDisabling(false);
    }
  }

  return (
    <div>
      <Breadcrumbs items={[{ label: "Dashboard", href: "/dashboard" }, { label: "My account" }]} />
      <AdminHeader title="My account" intro={`${user.name} · ${user.email}`} />

      <div className="border-t border-line pt-6">
        <p className="text-xs font-semibold uppercase tracking-wide text-ink-faint">
          Two-factor authentication
        </p>

        {statusError && <p className="mt-2 text-sm text-danger">{statusError}</p>}

        {enabled === null && !statusError && (
          <p className="mt-2 text-sm text-ink-faint">Loading...</p>
        )}

        {/* Recovery codes, shown exactly once right after confirming setup */}
        {recoveryCodes && (
          <div className="mt-3 border-l-2 border-success bg-success-soft py-3 pl-4">
            <p className="text-sm font-semibold text-success">
              Two-factor authentication is on. Save these recovery codes now.
            </p>
            <p className="mt-1 text-sm text-ink-muted">
              Each one works once, if you ever lose access to your authenticator app. They
              won&apos;t be shown again.
            </p>
            <ul className="mt-3 grid grid-cols-2 gap-2 font-mono text-sm text-ink">
              {recoveryCodes.map((c) => (
                <li key={c}>{c}</li>
              ))}
            </ul>
            <Button
              fullWidth={false}
              className="mt-4"
              onClick={() => {
                navigator.clipboard?.writeText(recoveryCodes.join("\n")).catch(() => {});
              }}
            >
              Copy all
            </Button>
            <SecondaryButton className="mt-4 ml-3" onClick={() => setRecoveryCodes(null)}>
              Done
            </SecondaryButton>
          </div>
        )}

        {/* Enabled, steady state */}
        {enabled === true && !recoveryCodes && (
          <div className="mt-3">
            <p className="text-sm text-ink-muted">
              Signing in now requires a code from your authenticator app, in addition to your
              password.
            </p>
            <SecondaryButton className="mt-3" onClick={() => setDisableOpen(true)}>
              Turn off
            </SecondaryButton>
          </div>
        )}

        {/* Disabled, not yet started setup */}
        {enabled === false && !setup && (
          <div className="mt-3">
            <p className="text-sm text-ink-muted">
              Not set up. Add an authenticator app (like Google Authenticator or 1Password) as a
              second step when you sign in.
            </p>
            {setupError && <p className="mt-2 text-sm text-danger">{setupError}</p>}
            <Button fullWidth={false} className="mt-3" onClick={handleStartSetup} disabled={startingSetup}>
              {startingSetup ? "Starting..." : "Set up two-factor authentication"}
            </Button>
          </div>
        )}

        {/* Setup in progress - scan or enter manually, then confirm with a live code */}
        {setup && (
          <div className="mt-3 flex flex-col gap-4 border-l-2 border-line pl-4">
            <p className="text-sm text-ink-muted">
              Scan this with your authenticator app, or enter the code manually if it can&apos;t
              scan.
            </p>
            {setup.qrCodePng && (
              // eslint-disable-next-line @next/next/no-img-element
              <img
                src={`data:image/png;base64,${setup.qrCodePng}`}
                alt="Scan with your authenticator app"
                width={180}
                height={180}
                className="border border-line"
              />
            )}
            <p className="font-mono text-sm text-ink">{setup.secret}</p>

            <form onSubmit={handleConfirm} className="flex flex-col gap-4">
              <Field label="Enter the 6-digit code it shows" htmlFor="confirm-code">
                <Input
                  id="confirm-code"
                  required
                  autoFocus
                  inputMode="numeric"
                  value={confirmCode}
                  onChange={(e) => setConfirmCode(e.target.value)}
                />
              </Field>
              {confirmError && <p className="text-sm text-danger">{confirmError}</p>}
              <div className="flex gap-3">
                <SecondaryButton type="button" onClick={() => setSetup(null)} disabled={confirming}>
                  Cancel
                </SecondaryButton>
                <Button type="submit" fullWidth={false} disabled={confirming}>
                  {confirming ? "Confirming..." : "Confirm and enable"}
                </Button>
              </div>
            </form>
          </div>
        )}
      </div>

      {disableOpen && (
        <Modal title="Turn off two-factor authentication" onClose={() => setDisableOpen(false)}>
          <form onSubmit={handleDisable} className="flex flex-col gap-4">
            <Field label="Confirm your password" htmlFor="disable-password">
              <PasswordInput
                id="disable-password"
                required
                autoFocus
                value={disablePassword}
                onChange={(e) => setDisablePassword(e.target.value)}
              />
            </Field>
            {disableError && <p className="text-sm text-danger">{disableError}</p>}
            <Button type="submit" disabled={disabling}>
              {disabling ? "Turning off..." : "Turn off two-factor authentication"}
            </Button>
          </form>
        </Modal>
      )}
    </div>
  );
}
