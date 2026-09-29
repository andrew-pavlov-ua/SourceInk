import type { Metadata } from "next";
import { PublicPageShell } from "@/components/public-page-shell";
import { LEGAL_EFFECTIVE_DATE, LEGAL_OPERATOR_NAME } from "@/lib/legal";

export const metadata: Metadata = { title: "License" };

export default function LicensePage() {
  return (
    <PublicPageShell>
      <article className="legal-page landing-frame">
        <header className="legal-page-header">
          <p>Effective {LEGAL_EFFECTIVE_DATE}</p>
          <h1>License</h1>
          <p>This page explains the rights you keep, the permission SourceInk needs to publish your work, and the limits on using the SourceInk service.</p>
        </header>

        <div className="legal-content">
          <section>
            <h2>Licensor</h2>
            <p>{LEGAL_OPERATOR_NAME} provides the SourceInk service and grants the service license described below.</p>
          </section>
          <section>
            <h2>Your content remains yours</h2>
            <p>You retain all ownership rights in the repositories, Markdown, frontmatter, images, and other material you connect to or publish through SourceInk.</p>
          </section>
          <section>
            <h2>Permission to operate the service</h2>
            <p>You grant SourceInk a non-exclusive, worldwide, royalty-free license to access, copy, process, render, cache, store, publish, and deliver your content only as needed to operate SourceInk and follow your publishing choices.</p>
          </section>
          <section>
            <h2>Published content</h2>
            <p>Publishing through SourceInk does not assign a license to readers. Readers may use an article only under the license its author provides or as permitted by applicable law.</p>
          </section>
          <section>
            <h2>Your right to use SourceInk</h2>
            <p>SourceInk grants you a limited, non-exclusive, non-transferable, revocable right to use the service under the Terms &amp; Conditions. You may not copy, resell, reverse engineer, or exploit the service except where applicable law permits it.</p>
          </section>
          <section>
            <h2>SourceInk software and branding</h2>
            <p>SourceInk and its branding, interface, and service software remain protected by copyright and other intellectual-property laws. Unless a separate open-source license accompanies specific source code, SourceInk reserves all rights in that code.</p>
          </section>
          <section>
            <h2>Third-party services</h2>
            <p>GitHub and other third-party services apply their own terms and licenses. This license does not grant rights to their software, trademarks, APIs, or content.</p>
          </section>
        </div>
      </article>
    </PublicPageShell>
  );
}
