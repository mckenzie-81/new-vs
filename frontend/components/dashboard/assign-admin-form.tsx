"use client";

import { useEffect, useState } from "react";
import { ApiError, assignAdmin, listPeople, Person } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Select } from "@/components/ui/select";
import { Field } from "@/components/ui/field";

export function AssignAdminForm({
  electionId,
  excludeUserIds,
  onAssigned,
}: {
  electionId: string;
  excludeUserIds: string[];
  onAssigned: () => void;
}) {
  const [people, setPeople] = useState<Person[] | null>(null);
  const [selected, setSelected] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listPeople()
      .then(setPeople)
      .catch(() => setPeople([]));
  }, []);

  const eligible = (people ?? []).filter(
    (p) => p.role === "ADMIN" && p.status === "ACTIVE" && !excludeUserIds.includes(p.id)
  );

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!selected) return;
    setError(null);
    setSubmitting(true);
    try {
      await assignAdmin(electionId, selected);
      onAssigned();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to assign.");
      setSubmitting(false);
    }
  }

  if (people === null) {
    return <p className="text-sm text-ink-faint">Loading...</p>;
  }

  if (eligible.length === 0) {
    return (
      <p className="text-sm text-ink-faint">
        No active admins available to assign. Invite one from Administrators first.
      </p>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      <Field label="Admin" htmlFor="admin-select">
        <Select
          id="admin-select"
          required
          value={selected}
          onChange={(e) => setSelected(e.target.value)}
        >
          <option value="" disabled>
            Choose an admin
          </option>
          {eligible.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name} ({p.email})
            </option>
          ))}
        </Select>
      </Field>

      {error && <p className="text-sm text-danger">{error}</p>}

      <Button type="submit" disabled={submitting || !selected} className="mt-1">
        {submitting ? "Assigning..." : "Assign"}
      </Button>
    </form>
  );
}
