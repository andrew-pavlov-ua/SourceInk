import type { Metadata } from "next";
import Link from "next/link";
import { CommitHistory } from "@/components/commit-history";
import { PublicPageShell } from "@/components/public-page-shell";

export const metadata: Metadata = { title: "Product" };

const capabilities = [
  ["Keep the source in GitHub", "Your repository keeps the Markdown, frontmatter, assets, and history."],
  ["Choose the public revision", "A push updates the draft. Readers keep the published copy until you publish again."],
  ["Serve a stored copy", "Reader requests use the article saved in SourceInk instead of calling GitHub."],
] as const;

export default function ProductPage() {
  return (
    <PublicPageShell>
      <section className="public-hero landing-frame">
        <h1>Write in GitHub. Publish with SourceInk.</h1>
        <p>SourceInk reads the repository. You decide which commit readers get.</p>
        <div className="hero-actions">
          <Link className="button" href="/register">Create an account</Link>
          <Link className="text-link" href="/docs">Read the model</Link>
        </div>
      </section>

      <section className="product-stage landing-frame">
        <CommitHistory />
      </section>

      <section className="capability-list landing-frame" aria-label="Product capabilities">
        {capabilities.map(([title, body]) => (
          <article key={title}>
            <h2>{title}</h2>
            <p>{body}</p>
          </article>
        ))}
      </section>
    </PublicPageShell>
  );
}
