"use client";

import { useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { ChevronDownIcon, ChevronRightIcon } from "@heroicons/react/24/outline";
import { navSections } from "./nav-items";
import { Sheet } from "@/components/ui/sheet";

function isActive(pathname: string, href: string) {
  if (href === "/dashboard") return pathname === "/dashboard";
  return pathname.startsWith(href);
}

function Brand() {
  return (
    <div className="p-5">
      <span className="text-sm font-semibold text-ink">Voting System</span>
    </div>
  );
}

export function SidebarContent({
  role,
  onNavigate,
}: {
  role: string;
  onNavigate?: () => void;
}) {
  const pathname = usePathname();
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set());

  const sections = navSections
    .map((section) => ({
      ...section,
      items: section.items.filter((item) => item.roles.includes(role)),
    }))
    .filter((section) => section.items.length > 0);

  function toggleSection(title: string) {
    setCollapsed((prev) => {
      const next = new Set(prev);
      if (next.has(title)) {
        next.delete(title);
      } else {
        next.add(title);
      }
      return next;
    });
  }

  return (
    <nav className="flex flex-col gap-5 p-5">
      {sections.map((section) => {
        const isCollapsed = collapsed.has(section.title);
        return (
          <div key={section.title}>
            <button
              onClick={() => toggleSection(section.title)}
              aria-expanded={!isCollapsed}
              className="mb-1 flex w-full items-center justify-between text-xs font-semibold uppercase tracking-wide text-ink-faint hover:text-ink-muted"
            >
              {section.title}
              {isCollapsed ? (
                <ChevronRightIcon className="h-3.5 w-3.5" />
              ) : (
                <ChevronDownIcon className="h-3.5 w-3.5" />
              )}
            </button>
            {!isCollapsed && (
              <div className="flex flex-col">
                {section.items.map((item) => {
                  const Icon = item.icon;
                  const active = isActive(pathname, item.href);
                  return (
                    <Link
                      key={item.href}
                      href={item.href}
                      onClick={onNavigate}
                      className={`flex items-center gap-2.5 border-l-2 py-1.5 pl-3 text-sm ${
                        active
                          ? "border-ink font-semibold text-ink"
                          : "border-transparent text-ink-muted hover:text-ink"
                      }`}
                    >
                      <Icon className="h-4 w-4" strokeWidth={2} />
                      {item.label}
                    </Link>
                  );
                })}
              </div>
            )}
          </div>
        );
      })}
    </nav>
  );
}

export function Sidebar({ role }: { role: string }) {
  return (
    <aside className="hidden w-56 shrink-0 flex-col border-r border-line md:flex">
      <Brand />
      <div className="flex-1 overflow-y-auto">
        <SidebarContent role={role} />
      </div>
    </aside>
  );
}

export function MobileSidebar({
  role,
  open,
  onClose,
}: {
  role: string;
  open: boolean;
  onClose: () => void;
}) {
  return (
    <Sheet open={open} onClose={onClose} side="left" widthClassName="w-56" className="md:hidden">
      <Brand />
      <div className="flex-1 overflow-y-auto">
        <SidebarContent role={role} onNavigate={onClose} />
      </div>
    </Sheet>
  );
}
