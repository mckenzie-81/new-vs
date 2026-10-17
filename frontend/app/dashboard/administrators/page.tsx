"use client";

import { useEffect, useState, useCallback } from "react";
import { listPeople, Person } from "@/lib/api";
import { useDashboardUser } from "@/components/dashboard/user-context";
import { AdminHeader } from "@/components/dashboard/admin-header";
import { Breadcrumbs } from "@/components/dashboard/breadcrumbs";
import { PersonRow } from "@/components/dashboard/person-row";
import { InviteForm } from "@/components/dashboard/invite-form";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";

export default function PeoplePage() {
  const user = useDashboardUser();
  const [people, setPeople] = useState<Person[] | null>(null);
  const [inviteOpen, setInviteOpen] = useState(false);

  const refresh = useCallback(() => {
    listPeople()
      .then(setPeople)
      .catch(() => setPeople([]));
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const others = people?.filter((p) => p.id !== user.id) ?? [];

  return (
    <div>
      <Breadcrumbs items={[{ label: "Dashboard", href: "/dashboard" }, { label: "People" }]} />
      <AdminHeader
        title="People"
        intro="Admins, approvers, and systemadmins."
        actions={
          <Button className="w-auto px-5" onClick={() => setInviteOpen(true)}>
            Invite
          </Button>
        }
      />

      {people === null && <p className="py-10 text-sm text-ink-faint">Loading...</p>}

      {people !== null && others.length === 0 && (
        <p className="py-10 text-sm text-ink-faint">
          No one else yet. Invite an admin or approver to help run elections.
        </p>
      )}

      {people !== null && others.length > 0 && (
        <ul className="divide-y divide-line border-t border-line">
          {people.map((person) => (
            <PersonRow key={person.id} person={person} />
          ))}
        </ul>
      )}

      {inviteOpen && (
        <Modal title="Invite someone" onClose={() => setInviteOpen(false)}>
          <InviteForm
            onSent={() => {
              setInviteOpen(false);
              refresh();
            }}
          />
        </Modal>
      )}
    </div>
  );
}
