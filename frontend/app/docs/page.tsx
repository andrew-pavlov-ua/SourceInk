import type { Metadata } from "next";
import Link from "next/link";
import { PublicPageShell } from "@/components/public-page-shell";

export const metadata: Metadata = { title: "Docs" };

const docs = [
  { id: "connect", title: "Install the App", body: "Choose the repositories SourceInk can read. OAuth checks that your GitHub account can access the installation." },
  { id: "discover", title: "Find source files", body: "SourceInk scans the selected repositories for Markdown files with frontmatter." },
  { id: "review", title: "Inspect a draft", body: "Open a discovered file to check its metadata, source location, and Markdown." },
  { id: "publish", title: "Publish", body: "Open a valid draft in the dashboard and publish its current repository snapshot." },
] as const;

export default function DocsPage() {
  return (
    <PublicPageShell>
      <div className="docs-layout landing-frame">
        <aside className="docs-sidebar">
          <p>Getting started</p>
          {docs.map((item) => <a href={`#${item.id}`} key={item.id}>{item.title}</a>)}
        </aside>
        <article className="docs-content">
          <header>
            <h1>Repository setup and article discovery</h1>
            <p>Connect GitHub, then inspect the drafts SourceInk finds.</p>
          </header>
          {docs.map((item, index) => (
            <section id={item.id} key={item.id}>
              <span>{index + 1}</span>
              <div><h2>{item.title}</h2><p>{item.body}</p></div>
            </section>
          ))}
          <p className="docs-next">Ready to connect a repository? <Link href="/register">Create an account</Link></p>
        </article>
      </div>
    </PublicPageShell>
  );
}
