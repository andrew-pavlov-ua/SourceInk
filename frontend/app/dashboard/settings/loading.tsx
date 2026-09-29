import { ServerLoading } from "@/components/server-loading";

export default function SettingsLoading() {
  return (
    <section className="app-page app-page-narrow" aria-labelledby="settings-loading-heading">
      <header className="app-page-heading">
        <div><h1 id="settings-loading-heading">Settings</h1><p>Account and GitHub connection preferences.</p></div>
      </header>
      <ServerLoading label="Loading account settings from the server" rows={2} variant="settings" />
    </section>
  );
}
