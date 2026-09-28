"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const sections = [
  { label: "Articles", href: "/dashboard" },
  { label: "Repositories", href: "/dashboard/repositories" },
  { label: "Domains", href: "/dashboard/domains" },
  { label: "Settings", href: "/dashboard/settings" },
] as const;

export function AppNavigation() {
  const pathname = usePathname();

  return (
    <nav className="app-tabs" aria-label="Publication sections">
      {sections.map((section) => {
        const active = section.href === "/dashboard"
          ? pathname === section.href || pathname.startsWith("/dashboard/articles/")
          : pathname.startsWith(section.href);
        return (
          <Link
            className={active ? "app-tab app-tab-active" : "app-tab"}
            href={section.href}
            aria-current={active ? "page" : undefined}
            key={section.href}
          >
            {section.label}
          </Link>
        );
      })}
    </nav>
  );
}
