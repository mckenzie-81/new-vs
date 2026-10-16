"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { ApiError, getBootstrapStatus, initiateBootstrap } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Field } from "@/components/ui/field";
import { MailIcon } from "@/components/ui/status-icon";
import { CenteredCard, CenteredMessage } from "@/components/layout/centered-card";

type ViewState = "checking" | "unavailable" | "form" | "pending" | "load-error";

export default function Home() {
  const [view, setView] = useState<ViewState>("checking");
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [submittedEmail, setSubmittedEmail] = useState("");

  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [secret, setSecret] = useState("");

  useEffect(() => {
    getBootstrapStatus()
      .then(({ available }) => setView(available ? "form" : "unavailable"))
      .catch(() => setView("load-error"));
  }, []);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setFormError(null);
    setSubmitting(true);

    try {
      await initiateBootstrap({ name, email, password, secret });
      setSubmittedEmail(email);
      setView("pending");
    } catch (err) {
      if (err instanceof ApiError) {
        setFormError(err.message);
        if (err.status === 409) setView("unavailable");
      } else {
        setFormError("Something went wrong. Please try again.");
      }
    } finally {
      setSubmitting(false);
    }
  }

  if (view === "checking") {
    return <CenteredMessage>Checking setup status...</CenteredMessage>;
  }

  if (view === "load-error") {
    return (
      <CenteredMessage>
        Could not reach the server. Confirm it&apos;s running and try reloading
        this page.
      </CenteredMessage>
    );
  }

  if (view === "unavailable") {
    return (
      <CenteredCard>
        <div className="text-center">
          <p className="text-sm leading-relaxed text-ink-muted">
            Setup has already been completed for this system.
          </p>
          <Link
            href="/login"
            className="mt-4 inline-block text-sm text-accent hover:underline"
          >
            Sign in
          </Link>
        </div>
      </CenteredCard>
    );
  }

  if (view === "pending") {
    return (
      <CenteredCard>
        <div className="text-center">
          <MailIcon />
          <h1 className="mb-2 text-xl font-semibold tracking-tight">
            Check your email
          </h1>
          <p className="text-sm leading-relaxed text-ink-muted">
            We sent a verification link to{" "}
            <strong className="text-ink">{submittedEmail}</strong>. It expires
            in 12 hours.
          </p>
          <p className="mt-6 text-xs leading-relaxed text-ink-faint">
            Didn&apos;t get it? Check spam, or start the setup process again.
          </p>
        </div>
      </CenteredCard>
    );
  }

  return (
    <CenteredCard>
      <div className="mb-2 text-[11px] font-semibold uppercase tracking-[0.09em] text-accent">
        One-time setup
      </div>
      <h1 className="mb-2 text-[23px] font-semibold leading-tight tracking-tight">
        Create the admin account
      </h1>
      <p className="mb-8 text-sm leading-relaxed text-ink-muted">
        This system has no administrator yet. You&apos;ll need the setup key
        that was issued when it was deployed to continue.
      </p>

      <form onSubmit={handleSubmit} className="flex flex-col gap-[18px]">
        <Field label="Full name" htmlFor="name">
          <Input
            id="name"
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </Field>

        <Field label="Email" htmlFor="email">
          <Input
            id="email"
            required
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
        </Field>

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

        <Field label="Setup key" htmlFor="secret">
          <Input
            id="secret"
            required
            type="password"
            value={secret}
            onChange={(e) => setSecret(e.target.value)}
          />
        </Field>

        {formError && <p className="text-sm text-danger">{formError}</p>}

        <Button type="submit" disabled={submitting} className="mt-2">
          {submitting ? "Creating account..." : "Create account"}
        </Button>
      </form>
    </CenteredCard>
  );
}
