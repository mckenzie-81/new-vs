"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { ChevronDownIcon } from "@heroicons/react/24/outline";
import { SessionUser, logout as apiLogout } from "@/lib/api";

export function ProfileMenu({ user }: { user: SessionUser }) {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const initial = user.name.trim().charAt(0).toUpperCase() || "?";

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  async function handleLogout() {
    setLoggingOut(true);
    try {
      await apiLogout();
    } finally {
      router.replace("/login");
    }
  }

  return (
    <div ref={containerRef} className="relative">
      <button
        onClick={() => setOpen((v) => !v)}
        className="flex items-center gap-2.5"
        aria-expanded={open}
      >
        <div className="flex h-8 w-8 shrink-0 items-center justify-center bg-page text-sm font-semibold text-ink-muted">
          {initial}
        </div>
        <div className="hidden text-left leading-tight sm:block">
          <div className="text-sm font-medium text-ink">{user.name}</div>
          <div className="text-xs text-ink-muted">{user.email}</div>
        </div>
        <ChevronDownIcon className="hidden h-4 w-4 text-ink-faint sm:block" />
      </button>

      {open && (
        <div className="absolute right-0 top-full z-20 mt-2 w-48 border border-line bg-surface py-1">
          <div className="border-b border-line px-3 py-2 sm:hidden">
            <div className="text-sm font-medium text-ink">{user.name}</div>
            <div className="text-xs text-ink-muted">{user.email}</div>
          </div>
          <button
            onClick={handleLogout}
            disabled={loggingOut}
            className="block w-full px-3 py-2 text-left text-sm text-ink hover:bg-page disabled:opacity-60"
          >
            {loggingOut ? "Signing out..." : "Sign out"}
          </button>
        </div>
      )}
    </div>
  );
}
