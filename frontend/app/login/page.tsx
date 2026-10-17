"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ApiError, login } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { PasswordInput } from "@/components/ui/password-input";
import { Field } from "@/components/ui/field";
import { CenteredCard } from "@/components/layout/centered-card";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);

    try {
      await login(email, password);
      router.push("/dashboard");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Something went wrong. Please try again.");
      setSubmitting(false);
    }
  }

  return (
    <CenteredCard>
      <h1 className="mb-2 text-[23px] font-semibold leading-tight tracking-tight">
        Sign in
      </h1>
      <p className="mb-8 text-sm leading-relaxed text-ink-muted">
        Enter your email and password to continue.
      </p>

      <form onSubmit={handleSubmit} className="flex flex-col gap-5">
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
          <PasswordInput
            id="password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </Field>

        <Link href="/forgot-password" className="-mt-2 self-start text-sm text-accent hover:underline">
          Forgot password?
        </Link>

        {error && <p className="text-sm text-danger">{error}</p>}

        <Button type="submit" disabled={submitting} className="mt-2">
          {submitting ? "Signing in..." : "Sign in"}
        </Button>
      </form>
    </CenteredCard>
  );
}
