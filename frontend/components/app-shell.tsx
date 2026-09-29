import type { ReactNode } from "react";
import Link from "next/link";
import { AppNavigation } from "@/components/app-navigation";
import { LogoutButton } from "@/components/logout-button";
import { ThemeToggle } from "@/components/theme-toggle";

type AppShellProps = {
  username: string;
  children: ReactNode;
};

export function AppShell({ username, children }: AppShellProps) {
  return (
    <main id="main-content" className="app-shell">
      <header className="app-topbar">
        <div className="app-frame app-topbar-inner">
          <Link className="wordmark app-wordmark" href="/" aria-label="SourceInk home">
            <span className="wordmark-mark" aria-hidden="true">S</span>
            <span>SourceInk</span>
          </Link>
          <span className="app-product-name">Publishing desk</span>
          <div className="app-account">
            <span>Signed in as <strong>@{username}</strong></span>
            <ThemeToggle />
            <LogoutButton />
          </div>
        </div>
      </header>

      <div className="app-frame">
        <AppNavigation />

        <div className="app-content">{children}</div>
      </div>
    </main>
  );
}
