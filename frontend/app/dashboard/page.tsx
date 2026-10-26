"use client";

import { useEffect, useState } from "react";
import {
  ClipboardDocumentCheckIcon,
  BoltIcon,
  UsersIcon,
  ClockIcon,
} from "@heroicons/react/24/outline";
import { listPeople, listElections, Person, Election } from "@/lib/api";
import { useDashboardUser } from "@/components/dashboard/user-context";
import { AdminHeader } from "@/components/dashboard/admin-header";
import { StatTiles } from "@/components/dashboard/stat-tiles";
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

      <StatTiles
        stats={[
          {
            label: "Elections",
            value: elections ? elections.length : "–",
            icon: ClipboardDocumentCheckIcon,
            href: "/dashboard/elections",
          },
          {
            label: "Live now",
            value: elections ? elections.filter((e) => e.status === "LIVE").length : "–",
            icon: BoltIcon,
            href: "/dashboard/elections",
            tone: elections && elections.some((e) => e.status === "LIVE") ? "success" : undefined,
          },
          ...(user.role === "SYSTEMADMIN"
            ? [
                {
                  label: "Administrators",
                  value: people ? people.length : ("–" as const),
                  icon: UsersIcon,
                  href: "/dashboard/administrators",
                },
                {
                  label: "Pending invites",
                  value: people
                    ? people.filter((p) => p.status === "PENDING_VERIFICATION").length
                    : ("–" as const),
                  icon: ClockIcon,
                  href: "/dashboard/administrators",
                  tone:
                    people && people.some((p) => p.status === "PENDING_VERIFICATION")
                      ? ("warning" as const)
                      : undefined,
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
