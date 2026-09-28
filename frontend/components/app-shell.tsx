import type { ReactNode } from "react";
import Link from "next/link";
import { AppNavigation } from "@/components/app-navigation";
import { LogoutButton } from "@/components/logout-button";
import { ThemeToggle } from "@/components/theme-toggle";

type AppShellProps = {
  username: string;
  repository: string;
  children: ReactNode;
};

export function AppShell({ username, repository, children }: AppShellProps) {
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
        <div className="app-context">
          <nav className="breadcrumbs" aria-label="Breadcrumb">
            <Link href="/dashboard">Repositories</Link>
            <span aria-hidden="true">/</span>
            <span>{repository.replace("github.com/", "")}</span>
          </nav>
          <button className="repository-selector" type="button" disabled aria-label={`Repository selector: ${repository}`}>
            <span className="repository-selector-label">Repository</span>
            <strong>{repository}</strong>
            <span aria-hidden="true">⌄</span>
          </button>
        </div>

        <AppNavigation />

        <div className="app-content">{children}</div>
      </div>
    </main>
  );
}
