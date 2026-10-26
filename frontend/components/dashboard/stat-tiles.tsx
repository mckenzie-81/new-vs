import Link from "next/link";
import { Card, CardHeader, CardContent } from "@/components/ui/card";
import type { Icon } from "./nav-items";

export interface StatTile {
  label: string;
  value: string | number;
  icon: Icon;
  href?: string;
  // Same reasoning as StatRow's tone: reserved for a number that means
  // something beyond "here is a count" - a live election, something
  // waiting on an admin's attention.
  tone?: "success" | "warning" | "danger";
}

const toneClasses: Record<NonNullable<StatTile["tone"]>, string> = {
  success: "text-success",
  warning: "text-warning",
  danger: "text-danger",
};

// The card-tile grid an admin console dashboard opens with - a small
// label and icon up top, the number large underneath. StatRow (plain
// text, no card) stays as it is for the public pulse page, a different
// audience this tile treatment isn't meant for.
export function StatTiles({ stats }: { stats: StatTile[] }) {
  return (
    <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
      {stats.map((stat) => {
        const Icon = stat.icon;
        const content = (
          <Card className={stat.href ? "transition-colors duration-150 hover:border-accent" : undefined}>
            <CardHeader>
              <span className="text-sm font-medium text-ink-muted">{stat.label}</span>
              <Icon className="h-4 w-4 shrink-0 text-ink-faint" strokeWidth={2} />
            </CardHeader>
            <CardContent>
              <p
                className={`text-3xl font-bold tracking-tight tabular-nums ${
                  stat.tone ? toneClasses[stat.tone] : "text-ink"
                }`}
              >
                {stat.value}
              </p>
            </CardContent>
          </Card>
        );
        return stat.href ? (
          <Link key={stat.label} href={stat.href}>
            {content}
          </Link>
        ) : (
          <div key={stat.label}>{content}</div>
        );
      })}
    </div>
  );
}
