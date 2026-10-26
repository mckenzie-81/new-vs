"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getSession, getBranding, SessionUser } from "@/lib/api";
import { Sidebar, MobileSidebar } from "@/components/dashboard/sidebar";
import { Header } from "@/components/dashboard/header";
import { DashboardUserProvider } from "@/components/dashboard/user-context";
import { ThemeScope } from "@/components/branding/theme-scope";
import { IntegrityAlertBanner } from "@/components/dashboard/integrity-alert-banner";

export function DashboardLayoutClient({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const [user, setUser] = useState<SessionUser | null>(null);
  const [checking, setChecking] = useState(true);
  const [mobileNavOpen, setMobileNavOpen] = useState(false);
  const [accentColor, setAccentColor] = useState<string | null>(null);
  const [accentColorSecondary, setAccentColorSecondary] = useState<string | null>(null);
  const [accentColorTertiary, setAccentColorTertiary] = useState<string | null>(null);

  useEffect(() => {
    getSession()
      .then(setUser)
      .catch(() => router.replace("/login"))
      .finally(() => setChecking(false));
    // Not gated on the session check above - the system accent is public
    // and applies to the whole dashboard shell regardless of who's
    // signed in, so there's no reason to wait on it.
    getBranding()
      .then((settings) => {
        setAccentColor(settings.accentColor);
        setAccentColorSecondary(settings.accentColorSecondary);
        setAccentColorTertiary(settings.accentColorTertiary);
      })
      .catch(() => {});
  }, [router]);

  if (checking || !user) {
    return <div className="min-h-screen bg-page" />;
  }

  return (
    <DashboardUserProvider user={user}>
      <ThemeScope
        accentColor={accentColor}
        accentColorSecondary={accentColorSecondary}
        accentColorTertiary={accentColorTertiary}
      >
        <div className="flex min-h-screen bg-page">
          <Sidebar role={user.role} />
          <MobileSidebar role={user.role} open={mobileNavOpen} onClose={() => setMobileNavOpen(false)} />

          <div className="flex min-h-screen min-w-0 flex-1 flex-col">
            <Header user={user} onMenuClick={() => setMobileNavOpen(true)} />
            {user.role === "SYSTEMADMIN" && <IntegrityAlertBanner />}
            <main className="min-w-0 flex-1 px-4 pb-10 pt-6 md:px-8 lg:px-10">
              <div className="mx-auto max-w-6xl">{children}</div>
            </main>
          </div>
        </div>
      </ThemeScope>
    </DashboardUserProvider>
  );
}
