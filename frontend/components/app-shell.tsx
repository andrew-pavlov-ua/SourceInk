import type { ReactNode } from "react";
import { Logo } from "@/components/logo";
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
          <Logo />
          <div className="app-account">
            <span className="app-username">@{username}</span>
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
