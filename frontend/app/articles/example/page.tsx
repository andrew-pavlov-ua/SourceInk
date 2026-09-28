import type { Metadata } from "next";
import Link from "next/link";
import { PublicPageShell } from "@/components/public-page-shell";

export const metadata: Metadata = { title: "A practical guide to PostgreSQL indexes" };

export default function ExampleArticlePage() {
  return (
    <PublicPageShell>
      <article className="reader-page">
        <header className="reader-header">
          <Link className="reader-publication" href="/">Field Notes</Link>
          <h1>A practical guide to PostgreSQL indexes</h1>
          <p className="reader-deck">Choose an index from a real query, then verify it with the planner.</p>
          <div className="reader-meta">
            <span>By Alice Chen</span><span>8 min read</span><span>Updated Sep 8, 2026</span>
          </div>
        </header>
        <div className="reader-source">
          <span>Source <code>alice/field-notes</code></span>
          <span>Revision <code>bc832f1</code></span>
        </div>
        <div className="reader-body">
          <p className="reader-lead">An index helps when its columns and order match a query your application runs.</p>
          <h2>Start with the query</h2>
          <p>Look at the filters, joins, and sort order in the slow path. The query shape tells you which columns belong in the index and in what order.</p>
          <pre><code>{`EXPLAIN (ANALYZE, BUFFERS)\nSELECT id, published_at\nFROM articles\nWHERE publication_id = $1\nORDER BY published_at DESC\nLIMIT 20;`}</code></pre>
          <h2>Check the execution plan</h2>
          <p>Compare execution time and buffer reads before and after the change. Use a dataset close to production size.</p>
          <h2>Keep the write cost visible</h2>
          <p>Each index adds work to inserts and updates. Remove duplicate indexes and check usage as query traffic changes.</p>
        </div>
      </article>
    </PublicPageShell>
  );
}
