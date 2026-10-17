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
    <li className="py-3">
      <div className="flex items-center gap-2">
        <span className="text-sm font-semibold text-ink">{election.title}</span>
        <Badge status={election.status} />
      </div>
      {election.description && (
        <p className="mt-0.5 text-sm text-ink-muted">{election.description}</p>
      )}
      <p className="mt-0.5 text-sm text-ink-faint">
        {formatRange(election.startTime, election.endTime)}
      </p>
    </li>
  );
}
