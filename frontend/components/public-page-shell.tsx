import type { ReactNode } from "react";
import { SiteHeader } from "@/components/site-header";

export function PublicPageShell({ children }: { children: ReactNode }) {
  return (
    <>
      <SiteHeader />
      <main id="main-content" className="public-page">{children}</main>
      <footer className="site-footer landing-frame">
        <span>© {new Date().getFullYear()} SourceInk</span>
        <span>Git-first publishing for technical articles.</span>
      </footer>
    </>
  );
}
