"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { getIntegrityAlert, acknowledgeIntegrityAlert, IntegrityAlert } from "@/lib/api";

const kindLabel: Record<string, string> = {
  CHAIN_BROKEN: "The audit chain failed an internal consistency check.",
  CHECKPOINT_MISMATCH: "The audit chain no longer matches a checkpoint taken earlier.",
};

// Polled once per dashboard load, not continuously - alerts are rare by
// design (an unresolved one is deliberately never re-raised, see
// internal/audit's recordAlert), so there's nothing to gain from polling
// a SYSTEMADMIN's open tab every few seconds for something this
// infrequent.
export function IntegrityAlertBanner() {
  const [alert, setAlert] = useState<IntegrityAlert | null>(null);
  const [acknowledging, setAcknowledging] = useState(false);

  useEffect(() => {
    getIntegrityAlert()
      .then(setAlert)
      .catch(() => {});
  }, []);

  if (!alert) return null;

  async function handleAcknowledge() {
    setAcknowledging(true);
    try {
      await acknowledgeIntegrityAlert(alert!.id);
      setAlert(null);
    } catch {
      setAcknowledging(false);
    }
  }

  return (
    <div className="border-b border-danger bg-danger-soft px-6 py-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm font-medium text-danger">
          Audit integrity alert: {kindLabel[alert.kind] ?? alert.detail}{" "}
          <Link href="/dashboard/audit-log" className="underline hover:no-underline">
            View details
          </Link>
        </p>
        <button
          type="button"
          onClick={handleAcknowledge}
          disabled={acknowledging}
          className="shrink-0 text-sm font-semibold text-danger underline hover:no-underline disabled:opacity-60"
        >
          {acknowledging ? "Acknowledging..." : "Acknowledge"}
        </button>
      </div>
    </div>
  );
}
