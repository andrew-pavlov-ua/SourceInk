import type { Metadata } from "next";
import Link from "next/link";
import { PublicPageShell } from "@/components/public-page-shell";

export const metadata: Metadata = { title: "Docs" };

const frontmatterExample = `---
title: Postgres indexes that scale
slug: postgres-indexes
description: How to choose indexes without slowing writes.
tags:
  - postgresql
  - performance
publish_mode: manual
---

# Postgres indexes that scale

Write the article here.`;

const sections = [
  { id: "connect", title: "Connect GitHub" },
  { id: "write", title: "Write a source file" },
  { id: "frontmatter", title: "Article frontmatter" },
  { id: "publish-mode", title: "Publish mode" },
  { id: "sync", title: "Sync and publish" },
] as const;

export default function DocsPage() {
  return (
    <PublicPageShell>
      <div className="docs-layout landing-frame">
        <aside className="docs-sidebar" aria-label="Documentation sections">
          <p>Publishing guide</p>
          {sections.map((section) => <a href={`#${section.id}`} key={section.id}>{section.title}</a>)}
        </aside>
        <article className="docs-content">
          <header>
            <h1>Publish from the repository you own</h1>
            <p>SourceInk reads Markdown from the GitHub repositories you select. Your repository stays the source of truth; SourceInk stores the draft and published copy it needs to serve readers.</p>
          </header>

          <section id="connect">
            <h2>Connect GitHub</h2>
            <p>Create a SourceInk account, open <strong>Repositories</strong>, then choose the GitHub repositories SourceInk may read. GitHub asks you to install the SourceInk App and choose the repositories.</p>
          </section>

          <section id="write">
            <h2>Write a source file</h2>
            <p>Create a Markdown file in a connected repository. Put its frontmatter on the first line, between an opening and closing <code>---</code>. Write the article body after the closing delimiter.</p>
          </section>

          <section id="frontmatter">
            <h2>Article frontmatter</h2>
            <p>Frontmatter gives SourceInk the metadata it needs to recognize, validate, and publish an article. <code>title</code>, <code>slug</code>, and <code>publish_mode</code> are required.</p>
            <pre className="docs-code"><code>{frontmatterExample}</code></pre>
            <dl className="docs-fields">
              <div><dt><code>title</code></dt><dd>The reader-facing title.</dd></div>
              <div><dt><code>slug</code></dt><dd>The URL-friendly article address. Choose a unique, stable value.</dd></div>
              <div><dt><code>publish_mode</code></dt><dd>Set this to <code>manual</code> or <code>auto</code>.</dd></div>
              <div><dt><code>description</code></dt><dd>An optional summary shown in SourceInk.</dd></div>
              <div><dt><code>tags</code></dt><dd>An optional YAML list used to describe the article.</dd></div>
            </dl>
          </section>

          <section id="publish-mode">
            <h2>Choose a publish mode</h2>
            <dl className="docs-publish-modes">
              <div>
                <dt><code>manual</code></dt>
                <dd>SourceInk saves each valid revision as a draft. Open the draft in the dashboard and publish the revision you want readers to see.</dd>
              </div>
              <div>
                <dt><code>auto</code></dt>
                <dd>SourceInk publishes each valid synced revision. Use this when a Git push should update the public article without a dashboard approval step.</dd>
              </div>
            </dl>
          </section>

          <section id="sync">
            <h2>Push to sync and publish</h2>
            <p>SourceInk syncs a connected repository when GitHub sends a push for its default branch. It validates the frontmatter and Markdown before updating the draft. A valid <code>auto</code> article publishes after the sync; a <code>manual</code> article stays a draft until you publish it.</p>
            <p className="docs-note">If SourceInk reports a frontmatter warning, fix the source file and push again. The last published copy remains available while you fix the draft.</p>
          </section>

          <p className="docs-next">Ready to publish? <Link href="/register">Create an account</Link> or <Link href="/login">log in</Link>.</p>
        </article>
      </div>
    </PublicPageShell>
  );
}
