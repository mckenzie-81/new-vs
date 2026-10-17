"use client";

import { useElection } from "@/components/dashboard/election-context";
import { AdminHeader } from "@/components/dashboard/admin-header";

export default function ResultsPage() {
  const { election } = useElection();

  return (
    <div>
      <AdminHeader title="Results" intro="Not available until this election closes." />
      <p className="py-10 text-sm text-ink-faint">
        {election.status === "CLOSED" || election.status === "ARCHIVED"
          ? "Not built yet."
          : "Results will appear here once voting has closed."}
      </p>
    </div>
  );
}
