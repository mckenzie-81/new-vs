import {
  Squares2X2Icon,
  ClipboardDocumentCheckIcon,
  UsersIcon,
  CircleStackIcon,
  ListBulletIcon,
  Cog6ToothIcon,
} from "@heroicons/react/24/outline";
import type { ComponentType, SVGProps } from "react";

export type Icon = ComponentType<SVGProps<SVGSVGElement>>;

export interface NavItem {
  label: string;
  href: string;
  icon: Icon;
  roles: string[];
}

export interface NavSection {
  title: string;
  items: NavItem[];
}

export const navSections: NavSection[] = [
  {
    title: "Management",
    items: [
      {
        label: "Overview",
        href: "/dashboard",
        icon: Squares2X2Icon,
        roles: ["SYSTEMADMIN", "ADMIN", "APPROVER"],
      },
      {
        label: "Elections",
        href: "/dashboard/elections",
        icon: ClipboardDocumentCheckIcon,
        roles: ["SYSTEMADMIN", "ADMIN", "APPROVER"],
      },
      {
        label: "Administrators",
        href: "/dashboard/administrators",
        icon: UsersIcon,
        roles: ["SYSTEMADMIN"],
      },
      {
        label: "Electorate",
        href: "/dashboard/electorate",
        icon: CircleStackIcon,
        roles: ["SYSTEMADMIN"],
      },
    ],
  },
  {
    title: "System",
    items: [
      {
        label: "Audit log",
        href: "/dashboard/audit-log",
        icon: ListBulletIcon,
        roles: ["SYSTEMADMIN"],
      },
      {
        label: "Settings",
        href: "/dashboard/settings",
        icon: Cog6ToothIcon,
        roles: ["SYSTEMADMIN"],
      },
    ],
  },
];

export function flatNavItems(role: string): NavItem[] {
  return navSections.flatMap((section) =>
    section.items.filter((item) => item.roles.includes(role))
  );
}
