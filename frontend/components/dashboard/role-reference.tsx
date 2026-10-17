const roles = [
  {
    name: "Systemadmin",
    description:
      "Full access. Creates elections, invites everyone else, and is the only role that can manage administrators.",
  },
  {
    name: "Admin",
    description: "Runs the elections they're assigned to - candidates, portfolios, day-to-day setup.",
  },
  {
    name: "Approver",
    description: "Reviews an election and signs off before it can go live - a separate check from whoever configured it.",
  },
];

export function RoleReference() {
  return (
    <div>
      <p className="mb-3 text-xs font-semibold uppercase tracking-wide text-ink-faint">
        Roles
      </p>
      <dl className="divide-y divide-line border-t border-line">
        {roles.map((role) => (
          <div key={role.name} className="grid grid-cols-1 gap-1 py-3 sm:grid-cols-[140px_1fr] sm:gap-4">
            <dt className="text-sm font-semibold text-ink">{role.name}</dt>
            <dd className="text-sm text-ink-muted">{role.description}</dd>
          </div>
        ))}
      </dl>
    </div>
  );
}
