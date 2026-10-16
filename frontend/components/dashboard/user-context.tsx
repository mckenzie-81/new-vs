"use client";

import { createContext, useContext } from "react";
import { SessionUser } from "@/lib/api";

const DashboardUserContext = createContext<SessionUser | null>(null);

export function DashboardUserProvider({
  user,
  children,
}: {
  user: SessionUser;
  children: React.ReactNode;
}) {
  return (
    <DashboardUserContext.Provider value={user}>
      {children}
    </DashboardUserContext.Provider>
  );
}

// Safe to call only from within the authenticated /dashboard route tree -
// the layout guarantees a user exists before rendering children at all.
export function useDashboardUser(): SessionUser {
  const user = useContext(DashboardUserContext);
  if (!user) {
    throw new Error("useDashboardUser called outside the dashboard layout");
  }
  return user;
}
