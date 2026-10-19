import { SettingsLayoutClient } from "@/components/dashboard/settings-layout-client";

export default function SettingsLayout({ children }: { children: React.ReactNode }) {
  return <SettingsLayoutClient>{children}</SettingsLayoutClient>;
}
