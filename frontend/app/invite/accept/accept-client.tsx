"use client";

import { useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { ApiError, acceptInvite } from "@/lib/api";
import { CenteredCard, CenteredMessage } from "@/components/layout/centered-card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Field } from "@/components/ui/field";

export default function AcceptInviteClient() {
  const searchParams = useSearchParams();
  const token = searchParams.get("token");

  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [status, setStatus] = useState<"form" | "active" | "invalid" | "expired">(
    token ? "form" : "invalid"
  );

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);

    if (password !== confirmPassword) {
      setError("Passwords don't match.");
      return;
    }

    setSubmitting(true);
    try {
      await acceptInvite(token!, password);
      setStatus("active");
    } catch (err) {
      if (err instanceof ApiError && err.status === 410) {
        setStatus("expired");
      } else if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("Something went wrong. Please try again.");
      }
    } finally {
      setSubmitting(false);
    }
  }

  if (status === "invalid") {
    return (
      <CenteredMessage>
        This invite link isn&apos;t valid. Double-check the link in your email.
      </CenteredMessage>
    );
  }

  if (status === "expired") {
    return (
      <CenteredMessage>
        This invite has expired. Ask whoever invited you to send a new one.
      </CenteredMessage>
    );
  }

  if (status === "active") {
    return (
      <CenteredCard>
        <div className="text-center">
          <h1 className="mb-2 text-xl font-semibold tracking-tight">
            Account activated
          </h1>
          <p className="mb-6 text-sm leading-relaxed text-ink-muted">
            Your account is ready.
          </p>
          <Link href="/login" className="text-sm text-accent hover:underline">
            Sign in
          </Link>
        </div>
      </CenteredCard>
    );
  }

  return (
    <CenteredCard>
      <h1 className="mb-2 text-[23px] font-semibold leading-tight tracking-tight">
        Set your password
      </h1>
      <p className="mb-8 text-sm leading-relaxed text-ink-muted">
        Choose a password to activate your account.
      </p>

      <form onSubmit={handleSubmit} className="flex flex-col gap-5">
        <Field label="Password" htmlFor="password" hint="At least 12 characters">
          <Input
            id="password"
            required
            type="password"
            minLength={12}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </Field>

        <Field label="Confirm password" htmlFor="confirm-password">
          <Input
            id="confirm-password"
            required
            type="password"
            minLength={12}
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
          />
        </Field>

        {error && <p className="text-sm text-danger">{error}</p>}

        <Button type="submit" disabled={submitting} className="mt-2">
          {submitting ? "Activating..." : "Activate account"}
        </Button>
      </form>
    </CenteredCard>
  );
}
