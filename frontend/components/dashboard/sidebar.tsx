"use client";

import { useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { ChevronDownIcon } from "@heroicons/react/24/outline";
import { navSections } from "./nav-items";
import { Sheet } from "@/components/ui/sheet";

function isActive(pathname: string, href: string) {
  if (href === "/dashboard") return pathname === "/dashboard";
  return pathname.startsWith(href);
}

function Brand() {
  return (
    <div className="px-6 pb-8 pt-7">
      <span className="text-sm font-bold uppercase tracking-wide text-white">
        Voting System
      </span>
      <span aria-hidden="true" className="mt-2 block h-px w-8 bg-accent-tertiary" />
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
    <nav className="flex flex-col gap-6 px-3 pb-6">
      {sections.map((section) => {
        const isCollapsed = collapsed.has(section.title);
        return (
          <div key={section.title}>
            <button
              onClick={() => toggleSection(section.title)}
              aria-expanded={!isCollapsed}
              className="mb-1 flex w-full items-center justify-between px-3 text-xs font-semibold uppercase tracking-wide text-white/50 transition-colors duration-150 hover:text-white/80"
            >
              {section.title}
              <ChevronDownIcon
                className={`h-3.5 w-3.5 shrink-0 transition-transform duration-150 ${
                  isCollapsed ? "-rotate-90" : ""
                }`}
              />
            </button>
            <div
              className={`grid transition-[grid-template-rows] duration-200 ease-in-out ${
                isCollapsed ? "grid-rows-[0fr]" : "grid-rows-[1fr]"
              }`}
            >
              <div
                // Content stays mounted so the height transition above has
                // something to measure - inert keeps it out of tab order
                // and off-screen-reader while visually collapsed, matching
                // what unmounting used to do for free.
                inert={isCollapsed}
                aria-hidden={isCollapsed}
                className={`flex flex-col overflow-hidden transition-opacity duration-150 ${
                  isCollapsed ? "opacity-0" : "opacity-100"
                }`}
              >
                {section.items.map((item) => {
                  const Icon = item.icon;
                  const active = isActive(pathname, item.href);
                  return (
                    <Link
                      key={item.href}
                      href={item.href}
                      onClick={onNavigate}
                      className={`relative flex items-center gap-2.5 px-3 py-2 text-sm transition-colors duration-150 ${
                        active
                          ? "bg-accent-secondary font-semibold text-white"
                          : "text-white/70 hover:bg-white/5 hover:text-white"
                      }`}
                    >
                      {/* Gold marks the current page - an accent, never
                          text, same rule the brand manual gives it
                          everywhere else it appears. */}
                      {active && (
                        <span
                          aria-hidden="true"
                          className="absolute inset-y-0 left-0 w-[3px] bg-accent-tertiary"
                        />
                      )}
                      <Icon
                        className={`h-4 w-4 ${active ? "text-accent-tertiary" : "text-white/50"}`}
                        strokeWidth={2}
                      />
                      {item.label}
                    </Link>
                  );
                })}
              </div>
            </div>
          </div>
        );
      })}
    </nav>
  );
}

export function Sidebar({ role }: { role: string }) {
  return (
    <aside className="hidden w-60 shrink-0 flex-col bg-accent md:flex">
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
    <Sheet
      open={open}
      onClose={onClose}
      side="left"
      widthClassName="w-60"
      className="md:hidden"
      panelClassName="bg-accent text-white"
    >
      <Brand />
      <div className="flex-1 overflow-y-auto">
        <SidebarContent role={role} onNavigate={onClose} />
      </div>
    </Sheet>
  );
}
