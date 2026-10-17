import { DashboardLayoutClient } from "@/components/dashboard/dashboard-layout-client";

// `instant` is route-segment config, which Next only allows in a Server
// Component module - hence this thin wrapper instead of putting the export
// directly in the "use client" file. Session comes from a separate Go
// backend behind an httpOnly cookie, so checking it means a client-side
// fetch, not a Server Component awaiting it directly - DashboardLayoutClient
// returns before {children} on purpose while that check is in flight, which
// Next 16.3's instant-navigation validator (Cache Components) flags. Opting
// out here is correct per Next's own guidance for an intentionally-excluded
// segment, not a workaround for a bug.
export const instant = false;

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return <DashboardLayoutClient>{children}</DashboardLayoutClient>;
}
