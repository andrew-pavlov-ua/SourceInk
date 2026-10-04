import type { Metadata } from "next";

export const metadata: Metadata = { title: "Domains" };

export default function DomainsPage() {
  return (
    <section className="app-page" aria-labelledby="domains-heading">
      <header className="app-page-heading">
        <div><h1 id="domains-heading">Domains</h1><p>Custom domains require public article delivery.</p></div>
        <button className="button button-small" type="button" disabled>Add domain</button>
      </header>
      <div className="empty-row empty-row-full">
        <div><h2>No custom domains yet</h2><p>SourceInk cannot serve public articles yet.</p></div>
      </div>
    </section>
  );
}
