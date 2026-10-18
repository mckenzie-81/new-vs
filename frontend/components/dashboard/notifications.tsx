"use client";

import { useState } from "react";
import { BellIcon } from "@heroicons/react/24/outline";
import { Sheet } from "@/components/ui/sheet";

export function Notifications() {
  const [open, setOpen] = useState(false);

  return (
    <>
      <button
        aria-label="Notifications"
        onClick={() => setOpen(true)}
        className="p-1.5 text-ink-muted transition-colors duration-150 hover:text-ink"
      >
        <BellIcon className="h-[18px] w-[18px]" />
      </button>

      <Sheet open={open} onClose={() => setOpen(false)} title="Notifications">
        <div className="p-5">
          <p className="text-sm text-ink-faint">No notifications yet.</p>
        </div>
      </Sheet>
    </>
  );
}
