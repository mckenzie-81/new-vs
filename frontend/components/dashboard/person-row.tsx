import { Person } from "@/lib/api";
import { Badge } from "./badge";

export function PersonRow({ person }: { person: Person }) {
  const joined = new Date(person.createdAt).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  });

  return (
    <li className="flex flex-wrap items-center justify-between gap-2 py-3">
      <div>
        <div className="flex items-center gap-2">
          <span className="text-sm font-semibold text-ink">{person.name}</span>
          <Badge status={person.status} />
        </div>
        <div className="mt-0.5 text-sm text-ink-muted">
          {person.email} · {person.role.toLowerCase()} · invited {joined}
        </div>
      </div>
    </li>
  );
}
