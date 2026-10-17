"use client";

import { useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { ApiError, confirmPasswordReset } from "@/lib/api";
import { CenteredCard, CenteredMessage } from "@/components/layout/centered-card";
import { Button } from "@/components/ui/button";
import { PasswordInput } from "@/components/ui/password-input";
import { Field } from "@/components/ui/field";

export default function ResetPasswordClient() {
  const searchParams = useSearchParams();
  const token = searchParams.get("token");

  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [status, setStatus] = useState<"form" | "done" | "invalid" | "expired">(
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
      await confirmPasswordReset(token!, password);
      setStatus("done");
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
        This reset link isn&apos;t valid. Double-check the link in your email, or request a new
        one.
      </CenteredMessage>
    );
  }

  if (status === "expired") {
    return (
      <CenteredCard>
        <div className="text-center">
          <h1 className="mb-2 text-xl font-semibold tracking-tight">Link expired</h1>
          <p className="mb-6 text-sm leading-relaxed text-ink-muted">
            This reset link has expired. Request a new one to continue.
          </p>
          <Link href="/forgot-password" className="text-sm text-accent hover:underline">
            Request a new link
          </Link>
        </div>
      </CenteredCard>
    );
  }

  if (status === "done") {
    return (
      <CenteredCard>
        <div className="text-center">
          <h1 className="mb-2 text-xl font-semibold tracking-tight">Password updated</h1>
          <p className="mb-6 text-sm leading-relaxed text-ink-muted">
            You&apos;ve been signed out everywhere else for safety. Sign in again with your new
            password.
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
        Choose a new password
      </h1>
      <p className="mb-8 text-sm leading-relaxed text-ink-muted">
        This will sign you out everywhere else.
      </p>

      <form onSubmit={handleSubmit} className="flex flex-col gap-5">
        <Field label="New password" htmlFor="password" hint="At least 12 characters">
          <PasswordInput
            id="password"
            required
            minLength={12}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </Field>

        <Field label="Confirm password" htmlFor="confirm-password">
          <PasswordInput
            id="confirm-password"
            required
            minLength={12}
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
          />
        </Field>

        {error && <p className="text-sm text-danger">{error}</p>}

        <Button type="submit" disabled={submitting} className="mt-2">
          {submitting ? "Updating..." : "Update password"}
        </Button>
      </form>
    </CenteredCard>
  );
}
