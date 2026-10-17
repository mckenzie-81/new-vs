"use client";

import { useEffect, useState } from "react";
import { listPeople, listElections, Person, Election } from "@/lib/api";
import { useDashboardUser } from "@/components/dashboard/user-context";
import { AdminHeader } from "@/components/dashboard/admin-header";
import { StatRow } from "@/components/dashboard/stat-row";
import { QuickActions } from "@/components/dashboard/quick-actions";

export default function OverviewPage() {
  const user = useDashboardUser();
  const [people, setPeople] = useState<Person[] | null>(null);
  const [elections, setElections] = useState<Election[] | null>(null);

  useEffect(() => {
    listElections().then(setElections).catch(() => setElections([]));
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

      <StatRow
        stats={[
          {
            label: "Elections",
            value: elections ? elections.length : "–",
            href: "/dashboard/elections",
          },
          {
            label: "Live now",
            value: elections ? elections.filter((e) => e.status === "LIVE").length : "–",
            href: "/dashboard/elections",
          },
          ...(user.role === "SYSTEMADMIN"
            ? [
                {
                  label: "Administrators",
                  value: people ? people.length : ("–" as const),
                  href: "/dashboard/administrators",
                },
                {
                  label: "Pending invites",
                  value: people
                    ? people.filter((p) => p.status === "PENDING_VERIFICATION").length
                    : ("–" as const),
                  href: "/dashboard/administrators",
                },
              ]
            : []),
        ]}
      />

      <div className="mt-12">
        <QuickActions role={user.role} />
      </div>
    </div>
  );
}
