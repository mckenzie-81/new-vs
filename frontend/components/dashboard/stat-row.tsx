import Link from "next/link";

export interface Stat {
  label: string;
  value: string | number;
  href?: string;
  // Reserve color for a stat that actually carries a meaning beyond "here
  // is a number" - a live count, something waiting on you, a warning. A
  // row where every number is the same flat ink gets the stat row listed
  // for the sake of it; tone is what makes the ones worth noticing stand
  // out from the ones that are just context.
  tone?: "success" | "warning" | "danger" | "accent";
}

const toneClasses: Record<NonNullable<Stat["tone"]>, string> = {
  success: "text-success",
  warning: "text-warning",
  danger: "text-danger",
  accent: "text-accent",
};

export function StatRow({ stats }: { stats: Stat[] }) {
  return (
    <ul className="grid grid-cols-2 gap-x-8 gap-y-8 sm:grid-cols-4">
      {stats.map((stat) => {
        const body = (
          <>
            <p className="text-sm text-ink-muted">{stat.label}</p>
            <p
              className={`mt-1 text-4xl font-semibold tracking-tight tabular-nums group-hover:underline ${
                stat.tone ? toneClasses[stat.tone] : "text-ink"
              }`}
            >
              {stat.value}
            </p>
          </>
        );
        return (
          <li key={stat.label}>
            {stat.href ? (
              <Link href={stat.href} className="group block">
                {body}
              </Link>
            ) : (
              <div>{body}</div>
            )}
          </li>
        );
      })}
    </ul>
  );
}
