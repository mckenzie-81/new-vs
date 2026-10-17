"use client";

import { useState } from "react";
import Link from "next/link";
import { requestPasswordReset } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Field } from "@/components/ui/field";
import { CenteredCard } from "@/components/layout/centered-card";

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [sent, setSent] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    try {
      await requestPasswordReset(email);
    } finally {
      // Same outcome shown whether or not the email is registered - the
      // backend's response can't be used to tell, so neither can this.
      setSent(true);
      setSubmitting(false);
    }
  }

  if (sent) {
    return (
      <CenteredCard>
        <div className="text-center">
          <h1 className="mb-2 text-xl font-semibold tracking-tight">Check your email</h1>
          <p className="text-sm leading-relaxed text-ink-muted">
            If <strong className="text-ink">{email}</strong> has an account, we sent a link to
            reset the password. It expires in an hour.
          </p>
          <Link href="/login" className="mt-6 inline-block text-sm text-accent hover:underline">
            Back to sign in
          </Link>
        </div>
      </CenteredCard>
    );
  }

  return (
    <CenteredCard>
      <h1 className="mb-2 text-[23px] font-semibold leading-tight tracking-tight">
        Reset your password
      </h1>
      <p className="mb-8 text-sm leading-relaxed text-ink-muted">
        Enter your email and we&apos;ll send a link to reset your password.
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

        <Button type="submit" disabled={submitting} className="mt-2">
          {submitting ? "Sending..." : "Send reset link"}
        </Button>

        <Link href="/login" className="self-center text-sm text-ink-muted hover:text-ink">
          Back to sign in
        </Link>
      </form>
    </CenteredCard>
  );
}
