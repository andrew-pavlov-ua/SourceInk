import type { Metadata } from "next";

export const metadata: Metadata = { title: "Settings" };

export default function SettingsPage() {
  return (
    <section className="app-page app-page-narrow" aria-labelledby="settings-heading">
      <header className="app-page-heading">
        <div><h1 id="settings-heading">Settings</h1><p>This build has no publishing controls.</p></div>
      </header>
      <div className="empty-row empty-row-full"><div><h2>No article settings yet</h2><p>You can inspect drafts now. Publish controls come later.</p></div></div>
    </section>
  );
}
