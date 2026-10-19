"use client";

import { usePathname } from "next/navigation";
import Link from "next/link";
import { AdminHeader } from "@/components/dashboard/admin-header";
import { Breadcrumbs } from "@/components/dashboard/breadcrumbs";

const tabs = [
  { label: "Branding", href: "" },
  { label: "Email", href: "/email" },
];

const base = "/dashboard/settings";

export function SettingsLayoutClient({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();

  return (
    <div>
      <Breadcrumbs items={[{ label: "Dashboard", href: "/dashboard" }, { label: "Settings" }]} />
      <AdminHeader title="Settings" intro="System-wide configuration." />

      <div className="mb-8 flex gap-6 border-b border-line">
        {tabs.map((tab) => {
          const href = base + tab.href;
          const active = pathname === href;
          return (
            <Link
              key={tab.href}
              href={href}
              className={`-mb-px border-b-2 pb-3 text-sm font-semibold transition-colors duration-150 ${
                active ? "border-accent text-ink" : "border-transparent text-ink-muted hover:text-ink"
              }`}
            >
              {tab.label}
            </Link>
          );
        })}
      </div>

      {children}
    </div>
  );
}
