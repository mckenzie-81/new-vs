"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getSession, SessionUser } from "@/lib/api";
import { Sidebar, MobileSidebar } from "@/components/dashboard/sidebar";
import { Header } from "@/components/dashboard/header";
import { DashboardUserProvider } from "@/components/dashboard/user-context";

export function DashboardLayoutClient({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const [user, setUser] = useState<SessionUser | null>(null);
  const [checking, setChecking] = useState(true);
  const [mobileNavOpen, setMobileNavOpen] = useState(false);

  useEffect(() => {
    getSession()
      .then(setUser)
      .catch(() => router.replace("/login"))
      .finally(() => setChecking(false));
  }, [router]);

  if (checking || !user) {
    return <div className="min-h-screen bg-page" />;
  }

  return (
    <DashboardUserProvider user={user}>
      <div className="flex min-h-screen bg-page">
        <Sidebar role={user.role} />
        <MobileSidebar role={user.role} open={mobileNavOpen} onClose={() => setMobileNavOpen(false)} />

        <div className="flex min-h-screen min-w-0 flex-1 flex-col">
          <Header user={user} onMenuClick={() => setMobileNavOpen(true)} />
          <main className="min-w-0 flex-1 p-6">{children}</main>
        </div>
      </div>
    </DashboardUserProvider>
  );
}
