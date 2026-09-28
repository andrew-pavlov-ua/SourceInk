import type { Metadata } from "next";
import Link from "next/link";
import { PublicPageShell } from "@/components/public-page-shell";

export const metadata: Metadata = { title: "Pricing" };

export default function PricingPage() {
  return (
    <PublicPageShell>
      <section className="public-hero pricing-hero landing-frame">
        <p className="page-kicker"><span aria-hidden="true" />Pricing</p>
        <h1>SourceInk has no paid plans yet.</h1>
        <p>Run the current build locally while the product is in development.</p>
      </section>
      <section className="pricing-table landing-frame" aria-label="Pricing preview">
        <header><span>Plan</span><span>For</span><span>Status</span><span /></header>
        <div>
          <strong>Early access</strong>
          <p>Writers running SourceInk on their own machine.</p>
          <span className="status-label"><i aria-hidden="true" />Local development</span>
          <Link className="button button-small" href="/register">Create an account</Link>
        </div>
      </section>
    </PublicPageShell>
  );
}
