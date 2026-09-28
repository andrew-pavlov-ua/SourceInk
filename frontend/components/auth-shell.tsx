import type { ReactNode } from "react";
import { SiteHeader } from "@/components/site-header";

export function AuthShell({ title, description, children }: { title: string; description: string; children: ReactNode }) {
  return (
    <>
      <SiteHeader />
      <main id="main-content" className="auth-layout shell">
        <section className="auth-intro">
          <h1>{title}</h1>
          <p>{description}</p>
        </section>
        <section className="auth-panel" aria-label={title}>{children}</section>
      </main>
    </>
  );
}
