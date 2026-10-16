"use client";

import { useState } from "react";
import { ApiError, logout as apiLogout, login, SessionUser } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Field } from "@/components/ui/field";
import { CenteredCard } from "@/components/layout/centered-card";

export default function LoginPage() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [user, setUser] = useState<SessionUser | null>(null);
  const [loggingOut, setLoggingOut] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);

    try {
      const sessionUser = await login(email, password);
      setUser(sessionUser);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Something went wrong. Please try again.");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleLogout() {
    setLoggingOut(true);
    try {
      await apiLogout();
    } finally {
      setUser(null);
      setPassword("");
      setLoggingOut(false);
    }
  }

  if (user) {
    return (
      <CenteredCard>
        <div className="text-center">
          <h1 className="mb-2 text-xl font-semibold tracking-tight">
            Signed in
          </h1>
          <p className="mb-6 text-sm leading-relaxed text-ink-muted">
            You&apos;re signed in as <strong className="text-ink">{user.name}</strong>{" "}
            ({user.role.toLowerCase()}). There&apos;s nothing to do here yet -
            the rest of the system hasn&apos;t been built.
          </p>
          <Button onClick={handleLogout} disabled={loggingOut} className="w-auto px-6">
            {loggingOut ? "Signing out..." : "Sign out"}
          </Button>
        </div>
      </CenteredCard>
    );
  }

  return (
    <CenteredCard>
      <h1 className="mb-2 text-[23px] font-semibold leading-tight tracking-tight">
        Sign in
      </h1>
      <p className="mb-8 text-sm leading-relaxed text-ink-muted">
        Enter your email and password to continue.
      </p>

      <form onSubmit={handleSubmit} className="flex flex-col gap-[18px]">
        <Field label="Email" htmlFor="email">
          <Input
            id="email"
            required
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
        </Field>

        <Field label="Password" htmlFor="password">
          <Input
            id="password"
            required
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </Field>

        {error && <p className="text-sm text-danger">{error}</p>}

        <Button type="submit" disabled={submitting} className="mt-2">
          {submitting ? "Signing in..." : "Sign in"}
        </Button>
      </form>
    </CenteredCard>
  );
}
