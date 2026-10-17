"use client";

import { Bars3Icon } from "@heroicons/react/24/outline";
import { SessionUser } from "@/lib/api";
import { QuickSearch } from "./quick-search";
import { Notifications } from "./notifications";
import { ProfileMenu } from "./profile-menu";

export function Header({
  user,
  onMenuClick,
}: {
  user: SessionUser;
  onMenuClick: () => void;
}) {
  return (
    <header className="flex h-16 shrink-0 items-center gap-4 px-4 sm:px-6">
      <button
        aria-label="Open menu"
        onClick={onMenuClick}
        className="-ml-1.5 p-1.5 text-ink md:hidden"
      >
        <Bars3Icon className="h-5 w-5" />
      </button>

      <div className="hidden flex-1 sm:block">
        <QuickSearch role={user.role} />
      </div>
      <div className="flex-1 sm:hidden" />

      <div className="flex items-center gap-4">
        <Notifications />
        <ProfileMenu user={user} />
      </div>
    </header>
  );
}
