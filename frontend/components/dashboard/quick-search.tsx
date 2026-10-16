"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { MagnifyingGlassIcon } from "@heroicons/react/24/outline";
import { flatNavItems } from "./nav-items";

export function QuickSearch({ role }: { role: string }) {
  const router = useRouter();
  const [query, setQuery] = useState("");
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

  const items = flatNavItems(role);
  const results = query.trim()
    ? items.filter((item) => item.label.toLowerCase().includes(query.trim().toLowerCase()))
    : items;

  function go(href: string) {
    router.push(href);
    setQuery("");
    setOpen(false);
  }

  return (
    <div ref={containerRef} className="relative w-full max-w-xs">
      <div className="flex items-center gap-2 border-b border-line py-1.5">
        <MagnifyingGlassIcon className="h-4 w-4 text-ink-faint" />
        <input
          type="text"
          placeholder="Search..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onFocus={() => setOpen(true)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && results.length > 0) go(results[0].href);
            if (e.key === "Escape") setOpen(false);
          }}
          className="w-full bg-transparent text-sm text-ink outline-none placeholder:text-ink-faint"
        />
      </div>

      {open && (
        <div className="absolute left-0 right-0 top-full z-20 mt-1 border border-line bg-surface">
          {results.length === 0 ? (
            <p className="p-3 text-sm text-ink-faint">No pages match &ldquo;{query}&rdquo;.</p>
          ) : (
            <ul>
              {results.map((item) => (
                <li key={item.href}>
                  <button
                    onClick={() => go(item.href)}
                    className="flex w-full items-center gap-2.5 px-3 py-2 text-left text-sm text-ink hover:bg-page"
                  >
                    <item.icon className="h-4 w-4 text-ink-faint" />
                    {item.label}
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}
