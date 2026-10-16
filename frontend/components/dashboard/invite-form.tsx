"use client";

import { useState } from "react";
import { ApiError, createInvite, InvitableRole } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Field } from "@/components/ui/field";

export function InviteForm({ onSent }: { onSent: () => void }) {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<InvitableRole>("ADMIN");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);

    try {
      await createInvite(name, email, role);
      onSent();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Something went wrong. Please try again.");
      setSubmitting(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-3">
      <p className="text-xs leading-relaxed text-ink-muted">
        They&apos;ll get a link to set their own password. It expires in 3 days.
      </p>

      <Field label="Name" htmlFor="invite-name">
        <Input
          id="invite-name"
          required
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
      </Field>

      <Field label="Email" htmlFor="invite-email">
        <Input
          id="invite-email"
          required
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />
      </Field>

      <Field label="Role" htmlFor="invite-role">
        <Select
          id="invite-role"
          value={role}
          onChange={(e) => setRole(e.target.value as InvitableRole)}
        >
          <option value="ADMIN">Admin</option>
          <option value="APPROVER">Approver</option>
          <option value="SYSTEMADMIN">Systemadmin</option>
        </Select>
      </Field>

      {error && <p className="text-sm text-danger">{error}</p>}

      <Button type="submit" disabled={submitting} className="mt-1">
        {submitting ? "Sending..." : "Send invite"}
      </Button>
    </form>
  );
}
