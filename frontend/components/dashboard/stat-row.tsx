import Link from "next/link";

export interface Stat {
  label: string;
  value: string | number;
  href?: string;
}

export function StatRow({ stats }: { stats: Stat[] }) {
  return (
    <ul className="grid grid-cols-2 gap-x-8 gap-y-8 sm:grid-cols-4">
      {stats.map((stat) => {
        const body = (
          <>
            <p className="text-sm text-ink-muted">{stat.label}</p>
            <p className="mt-1 text-4xl font-semibold tracking-tight tabular-nums group-hover:underline">
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
