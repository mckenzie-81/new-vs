import Link from "next/link";

interface Action {
  title: string;
  description: string;
  href: string;
  roles: string[];
}

const actions: Action[] = [
  {
    title: "Invite someone",
    description: "Add an admin, approver, or systemadmin",
    href: "/dashboard/people",
    roles: ["SYSTEMADMIN"],
  },
  {
    title: "View elections",
    description: "Create, approve, and run elections",
    href: "/dashboard/elections",
    roles: ["SYSTEMADMIN", "ADMIN", "APPROVER"],
  },
  {
    title: "Audit log",
    description: "Review recent sensitive activity",
    href: "/dashboard/audit-log",
    roles: ["SYSTEMADMIN"],
  },
  {
    title: "Settings",
    description: "System-wide configuration",
    href: "/dashboard/settings",
    roles: ["SYSTEMADMIN"],
  },
];

export function QuickActions({ role }: { role: string }) {
  const visible = actions.filter((action) => action.roles.includes(role));

  return (
    <div>
      <p className="mb-3 text-xs font-semibold uppercase tracking-wide text-ink-faint">
        Quick actions
      </p>
      <ul className="divide-y divide-line border-t border-line">
        {visible.map((action) => (
          <li key={action.href}>
            <Link href={action.href} className="group flex items-baseline justify-between gap-4 py-3">
              <span className="text-sm font-semibold text-ink group-hover:underline">
                {action.title}
              </span>
              <span className="text-sm text-ink-muted">{action.description}</span>
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}
