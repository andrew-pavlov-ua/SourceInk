import type { Metadata } from "next";
import { PublicPageShell } from "@/components/public-page-shell";
import { LEGAL_EFFECTIVE_DATE, LEGAL_OPERATOR_NAME } from "@/lib/legal";

export const metadata: Metadata = { title: "Terms & Conditions" };

export default function TermsPage() {
  return (
    <PublicPageShell>
      <article className="legal-page landing-frame">
        <header className="legal-page-header">
          <p>Effective {LEGAL_EFFECTIVE_DATE}</p>
          <h1>Terms &amp; Conditions</h1>
          <p>These terms govern your use of SourceInk, including its GitHub integration, publishing dashboard, and public article pages.</p>
        </header>

        <div className="legal-content">
          <section>
            <h2>Operator</h2>
            <p>{LEGAL_OPERATOR_NAME} operates this service. In these terms, “SourceInk,” “we,” and “us” refer to the SourceInk project.</p>
          </section>
          <section>
            <h2>1. Agreement</h2>
            <p>By creating an account or using SourceInk, you agree to these terms. If you use SourceInk for an organization, you confirm that you can accept these terms for that organization.</p>
          </section>
          <section>
            <h2>2. Your account</h2>
            <p>You must provide accurate account information and protect your sign-in credentials. You are responsible for activity performed through your account unless you report unauthorized access.</p>
          </section>
          <section>
            <h2>3. GitHub access</h2>
            <p>SourceInk reads only the repositories you select through the GitHub App. You keep ownership of those repositories and may revoke access through GitHub. Revoking access stops future repository synchronization but does not erase an existing published snapshot.</p>
          </section>
          <section>
            <h2>4. Your articles</h2>
            <p>You keep ownership of your Markdown, frontmatter, assets, and other content. You must have the rights needed to publish everything you submit. GitHub remains the canonical source, while SourceInk stores the copies needed for drafts and public delivery.</p>
          </section>
          <section>
            <h2>5. Publishing</h2>
            <p>SourceInk treats repository drafts and published articles as separate copies. Updating a repository file does not replace its published copy until the applicable publishing action succeeds.</p>
          </section>
          <section>
            <h2>6. Acceptable use</h2>
            <p>Do not use SourceInk to publish unlawful or infringing material, distribute malware, probe accounts or repositories you do not control, bypass access controls, or disrupt the service.</p>
          </section>
          <section>
            <h2>7. Service availability</h2>
            <p>SourceInk may change, suspend, or discontinue features. The service may experience errors, maintenance, GitHub outages, or data synchronization delays.</p>
          </section>
          <section>
            <h2>8. Suspension and termination</h2>
            <p>You may stop using SourceInk and revoke its GitHub access. SourceInk may suspend an account that violates these terms, creates security risk, or harms the service or another user.</p>
          </section>
          <section>
            <h2>9. Disclaimers</h2>
            <p>SourceInk provides the service on an “as is” and “as available” basis. To the extent permitted by law, SourceInk disclaims implied warranties of merchantability, fitness for a particular purpose, and non-infringement.</p>
          </section>
          <section>
            <h2>10. Liability</h2>
            <p>To the extent permitted by law, SourceInk is not liable for indirect, incidental, special, consequential, or punitive damages, or for lost profits, data, goodwill, or business opportunities arising from your use of the service.</p>
          </section>
          <section>
            <h2>11. Changes to these terms</h2>
            <p>SourceInk may update these terms as the service changes. The effective date at the top of this page identifies the current version. Continued use after an update means you accept the revised terms.</p>
          </section>
        </div>
      </article>
    </PublicPageShell>
  );
}
