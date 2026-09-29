import type { ReactNode } from "react";
import { SiteHeader } from "@/components/site-header";

export function PublicPageShell({ children }: { children: ReactNode }) {
  return (
    <>
      <SiteHeader />
      <main id="main-content" className="public-page">{children}</main>
    </>
  );
}
