import { ElectionLayoutClient } from "@/components/dashboard/election-layout-client";

export default function ElectionLayout({ children }: { children: React.ReactNode }) {
  return <ElectionLayoutClient>{children}</ElectionLayoutClient>;
}
