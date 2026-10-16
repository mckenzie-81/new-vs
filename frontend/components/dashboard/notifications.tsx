"use client";

import { useEffect, useRef, useState } from "react";
import { BellIcon } from "@heroicons/react/24/outline";

export function Notifications() {
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  return (
    <div ref={containerRef} className="relative">
      <button
        aria-label="Notifications"
        onClick={() => setOpen((v) => !v)}
        className="p-1.5 text-ink-muted hover:text-ink"
      >
        <BellIcon className="h-[18px] w-[18px]" />
      </button>

      {open && (
        <div className="absolute right-0 top-full z-20 mt-1 w-64 border border-line bg-surface p-4">
          <p className="text-sm text-ink-faint">No notifications yet.</p>
        </div>
      )}
    </div>
  );
}
