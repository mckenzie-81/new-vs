import { ElectionLayoutClient } from "@/components/dashboard/election-layout-client";

// `instant` is route-segment config, which Next only allows in a Server
// Component module - hence this thin wrapper instead of putting the export
// directly in the "use client" file. ElectionLayoutClient fetches the
// election client-side and returns early (not-found / loading) before
// {children} while that's in flight - intentional, same reasoning as the
// dashboard layout's own instant=false.
export const instant = false;

export default function ElectionLayout({ children }: { children: React.ReactNode }) {
  return <ElectionLayoutClient>{children}</ElectionLayoutClient>;
}
