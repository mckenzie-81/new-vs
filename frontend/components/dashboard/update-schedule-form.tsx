"use client";

import { useState } from "react";
import { ApiError, updateElectionSchedule, Election } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { Field } from "@/components/ui/field";

type Mode = "end_now" | "extend";
type Unit = "minutes" | "hours" | "days";

const UNIT_MS: Record<Unit, number> = {
  minutes: 60_000,
  hours: 60 * 60_000,
  days: 24 * 60 * 60_000,
};

export function UpdateScheduleForm({
  election,
  onUpdated,
}: {
  election: Election;
  onUpdated: () => void;
}) {
  const [mode, setMode] = useState<Mode>("extend");
  const [amount, setAmount] = useState(1);
  const [unit, setUnit] = useState<Unit>("hours");
  const [reason, setReason] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const newEndDate =
    mode === "end_now"
      ? new Date()
      : new Date(new Date(election.endTime).getTime() + amount * UNIT_MS[unit]);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      await updateElectionSchedule(election.id, newEndDate.toISOString(), reason);
      onUpdated();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Something went wrong. Please try again.");
      setSubmitting(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      <div className="flex border border-line">
        <button
          type="button"
          onClick={() => setMode("end_now")}
          className={`min-h-11 flex-1 text-sm font-semibold transition-colors duration-150 ${
            mode === "end_now" ? "bg-ink text-white" : "text-ink-muted hover:text-ink"
          }`}
        >
          End now
        </button>
        <button
          type="button"
          onClick={() => setMode("extend")}
          className={`min-h-11 flex-1 border-l border-line text-sm font-semibold transition-colors duration-150 ${
            mode === "extend" ? "bg-ink text-white" : "text-ink-muted hover:text-ink"
          }`}
        >
          Extend
        </button>
      </div>

      {mode === "end_now" ? (
        <p className="text-sm text-ink-muted">
          Voting closes the moment you submit this. Currently scheduled to close{" "}
          <span className="font-semibold text-ink">
            {new Date(election.endTime).toLocaleString()}
          </span>
          .
        </p>
      ) : (
        <div>
          <p className="text-sm font-semibold text-ink">Extend by</p>
          <div className="mt-2 flex items-center gap-2">
            <Input
              type="number"
              min={1}
              required
              value={amount}
              onChange={(e) => setAmount(Math.max(1, Number(e.target.value) || 1))}
              className="w-20"
            />
            <Select
              value={unit}
              onChange={(e) => setUnit(e.target.value as Unit)}
              className="flex-1"
            >
              <option value="minutes">Minutes</option>
              <option value="hours">Hours</option>
              <option value="days">Days</option>
            </Select>
          </div>
          <p className="mt-2 text-sm text-ink-muted">
            New closing time:{" "}
            <span className="font-semibold text-ink">{newEndDate.toLocaleString()}</span>
          </p>
        </div>
      )}

      <Field
        label="Reason"
        htmlFor="schedule-reason"
        hint="Shown on this election's public pulse page."
      >
        <Textarea
          id="schedule-reason"
          required
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
      </Field>

      {error && <p className="text-sm text-danger">{error}</p>}

      <Button type="submit" variant="accent" disabled={submitting} className="mt-1">
        {submitting ? "Saving..." : mode === "end_now" ? "End now" : "Extend"}
      </Button>
    </form>
  );
}
