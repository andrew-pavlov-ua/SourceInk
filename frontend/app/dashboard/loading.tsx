import { ServerLoading } from "@/components/server-loading";

export default function DashboardLoading() {
  return (
    <section className="app-page" aria-labelledby="articles-loading-heading">
      <header className="articles-heading">
        <div><h1 id="articles-loading-heading">Articles</h1><p>Your repository drafts and publications.</p></div>
      </header>
      <ServerLoading label="Loading articles from the server" />
    </section>
  );
}
