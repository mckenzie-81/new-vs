"use client";

import { createContext, useContext } from "react";
import { Election, AdminAssignment } from "@/lib/api";

interface ElectionContextValue {
  election: Election;
  assignedAdmins: AdminAssignment[];
  refresh: () => void;
}

const ElectionContext = createContext<ElectionContextValue | null>(null);

export function ElectionProvider({
  value,
  children,
}: {
  value: ElectionContextValue;
  children: React.ReactNode;
}) {
  return <ElectionContext.Provider value={value}>{children}</ElectionContext.Provider>;
}

// Safe to call only from within an election's layout tree - the layout
// guarantees the election and its assignments are loaded before rendering
// any tab content.
export function useElection(): ElectionContextValue {
  const ctx = useContext(ElectionContext);
  if (!ctx) {
    throw new Error("useElection called outside an election's layout");
  }
  return ctx;
}
