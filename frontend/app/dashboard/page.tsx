"use client";

import { useEffect, useState } from "react";
import { listPeople, Person } from "@/lib/api";
import { useDashboardUser } from "@/components/dashboard/user-context";
import { AdminHeader } from "@/components/dashboard/admin-header";
import { StatRow } from "@/components/dashboard/stat-row";
import { QuickActions } from "@/components/dashboard/quick-actions";

export default function OverviewPage() {
  const user = useDashboardUser();
  const [people, setPeople] = useState<Person[] | null>(null);

  useEffect(() => {
    if (user.role === "SYSTEMADMIN") {
      listPeople()
        .then(setPeople)
        .catch(() => setPeople([]));
    }
  }, [user.role]);

  return (
    <div>
      <AdminHeader
        title={`Welcome, ${user.name.split(" ")[0]}`}
        intro={`Signed in as ${user.role.toLowerCase()}.`}
      />

      {user.role === "SYSTEMADMIN" && (
        <StatRow
          stats={[
            { label: "People", value: people ? people.length : "–", href: "/dashboard/people" },
            {
              label: "Admins",
              value: people ? people.filter((p) => p.role === "ADMIN").length : "–",
              href: "/dashboard/people",
            },
            {
              label: "Approvers",
              value: people ? people.filter((p) => p.role === "APPROVER").length : "–",
              href: "/dashboard/people",
            },
            {
              label: "Pending invites",
              value: people
                ? people.filter((p) => p.status === "PENDING_VERIFICATION").length
                : "–",
              href: "/dashboard/people",
            },
          ]}
        />
      )}

      <div className="mt-12">
        <QuickActions role={user.role} />
      </div>
    </div>
  );
}
