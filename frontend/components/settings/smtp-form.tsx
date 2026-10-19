"use client";

import { useEffect, useState } from "react";
import {
  getSMTPSettings,
  updateSMTPSettings,
  sendSMTPTestEmail,
  ApiError,
} from "@/lib/api";
import { Button, SecondaryButton } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { PasswordInput } from "@/components/ui/password-input";

export function SMTPForm() {
  const [host, setHost] = useState("");
  const [port, setPort] = useState("587");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [hasPassword, setHasPassword] = useState(false);
  const [fromName, setFromName] = useState("");
  const [fromEmail, setFromEmail] = useState("");
  const [testMode, setTestMode] = useState(true);

  const [loaded, setLoaded] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  const [sendingTest, setSendingTest] = useState(false);
  const [testError, setTestError] = useState<string | null>(null);
  const [testSentTo, setTestSentTo] = useState<string | null>(null);

  useEffect(() => {
    getSMTPSettings()
      .then((settings) => {
        setHost(settings.host ?? "");
        setPort(settings.port ? String(settings.port) : "587");
        setUsername(settings.username ?? "");
        setHasPassword(settings.hasPassword);
        setFromName(settings.fromName ?? "");
        setFromEmail(settings.fromEmail ?? "");
        setTestMode(settings.testMode);
        setLoaded(true);
      })
      .catch(() => setLoaded(true));
  }, []);

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    setSaveError(null);
    setSaved(false);
    setSaving(true);
    try {
      const settings = await updateSMTPSettings({
        host,
        port: Number(port) || 0,
        username,
        password,
        fromName,
        fromEmail,
        testMode,
      });
      setHost(settings.host ?? "");
      setPort(settings.port ? String(settings.port) : "587");
      setUsername(settings.username ?? "");
      setHasPassword(settings.hasPassword);
      setFromName(settings.fromName ?? "");
      setFromEmail(settings.fromEmail ?? "");
      setTestMode(settings.testMode);
      setPassword("");
      setSaved(true);
    } catch (err) {
      setSaveError(err instanceof ApiError ? err.message : "Failed to save.");
    } finally {
      setSaving(false);
    }
  }

  async function handleSendTest() {
    setTestError(null);
    setTestSentTo(null);
    setSendingTest(true);
    try {
      const result = await sendSMTPTestEmail();
      setTestSentTo(result.to);
    } catch (err) {
      setTestError(err instanceof ApiError ? err.message : "Failed to send test email.");
    } finally {
      setSendingTest(false);
    }
  }

  if (!loaded) {
    return <p className="py-10 text-sm text-ink-faint">Loading...</p>;
  }

  return (
    <div>
      <h2 className="text-sm font-semibold uppercase tracking-wide text-ink-faint">Email</h2>
      <p className="mt-1 text-sm text-ink-muted">
        Plug in your own institution&apos;s SMTP account (Google Workspace, Office 365, or
        anything else) so mail genuinely goes out as you - passes your own SPF/DKIM, shows up in
        your own sent mail.
      </p>

      <form onSubmit={handleSave} className="mt-5 flex flex-col gap-4">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div className="sm:col-span-2">
            <Field label="SMTP host" htmlFor="smtp-host">
              <Input
                id="smtp-host"
                placeholder="smtp.gmail.com"
                value={host}
                onChange={(e) => setHost(e.target.value)}
              />
            </Field>
          </div>
          <Field label="Port" htmlFor="smtp-port">
            <Input
              id="smtp-port"
              type="number"
              min={1}
              max={65535}
              value={port}
              onChange={(e) => setPort(e.target.value)}
            />
          </Field>
        </div>

        <Field label="Username" htmlFor="smtp-username">
          <Input
            id="smtp-username"
            placeholder="elections@youruniversity.edu"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
          />
        </Field>

        <Field
          label="Password"
          htmlFor="smtp-password"
          hint={hasPassword ? "A password is already saved - leave blank to keep it." : undefined}
        >
          <PasswordInput
            id="smtp-password"
            placeholder={hasPassword ? "••••••••" : ""}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </Field>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="From name" htmlFor="smtp-from-name" hint="What recipients see as the sender.">
            <Input
              id="smtp-from-name"
              placeholder="University Elections"
              value={fromName}
              onChange={(e) => setFromName(e.target.value)}
            />
          </Field>
          <Field label="From email" htmlFor="smtp-from-email">
            <Input
              id="smtp-from-email"
              type="email"
              placeholder="elections@youruniversity.edu"
              value={fromEmail}
              onChange={(e) => setFromEmail(e.target.value)}
            />
          </Field>
        </div>

        <label className="flex items-start gap-2.5 text-sm text-ink">
          <input
            type="checkbox"
            className="mt-0.5"
            checked={testMode}
            onChange={(e) => setTestMode(e.target.checked)}
          />
          <span>
            <span className="font-semibold">Test mode</span>
            <span className="block text-ink-muted">
              While on, real voter and admin notifications stay exactly as they are today -
              logged, never actually sent - no matter what&apos;s configured above. Turn this off
              only once you&apos;re ready for this institution&apos;s real notifications to go out
              for real.
            </span>
          </span>
        </label>

        {saveError && <p className="text-sm text-danger">{saveError}</p>}
        {saved && !saveError && <p className="text-sm text-success">Saved.</p>}

        <Button type="submit" variant="accent" disabled={saving} className="mt-1" fullWidth={false}>
          {saving ? "Saving..." : "Save"}
        </Button>
      </form>

      <div className="mt-6 border-t border-line pt-4">
        <p className="text-sm text-ink-muted">
          Sends one real email to your own address right now, regardless of test mode above - the
          only way to actually confirm these credentials work.
        </p>
        <SecondaryButton className="mt-3" onClick={handleSendTest} disabled={sendingTest || !host}>
          {sendingTest ? "Sending..." : "Send test email"}
        </SecondaryButton>
        {testError && <p className="mt-2 text-sm text-danger">{testError}</p>}
        {testSentTo && !testError && (
          <p className="mt-2 text-sm text-success">Sent to {testSentTo} - check your inbox.</p>
        )}
      </div>
    </div>
  );
}
