import Link from "next/link";
import { Election } from "@/lib/api";
import { Badge } from "./badge";

function formatRange(startTime: string, endTime: string) {
  const opts: Intl.DateTimeFormatOptions = { month: "short", day: "numeric", year: "numeric" };
  const start = new Date(startTime).toLocaleDateString(undefined, opts);
  const end = new Date(endTime).toLocaleDateString(undefined, opts);
  return start === end ? start : `${start} – ${end}`;
}

export function ElectionRow({ election }: { election: Election }) {
  return (
    <li>
      <Link
        href={`/dashboard/elections/${election.id}`}
        className="group flex items-center justify-between gap-4 px-5 py-3.5 transition-colors duration-150 hover:bg-page"
      >
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className="truncate text-sm font-semibold text-ink group-hover:text-accent">
              {election.title}
            </span>
            <Badge status={election.status} />
          </div>
          {election.description && (
            <p className="mt-0.5 truncate text-sm text-ink-muted">{election.description}</p>
          )}
        </div>
        <p className="shrink-0 text-sm text-ink-faint">
          {formatRange(election.startTime, election.endTime)}
        </p>
      </Link>
    </li>
  );
}
