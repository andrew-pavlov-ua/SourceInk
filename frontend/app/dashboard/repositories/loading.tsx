import { ServerLoading } from "@/components/server-loading";

export default function RepositoriesLoading() {
  return (
    <section className="app-page" aria-labelledby="repositories-loading-heading">
      <header className="app-page-heading">
        <div><h1 id="repositories-loading-heading">Repositories</h1><p>Repositories connected to your SourceInk account.</p></div>
      </header>
      <ServerLoading label="Loading repositories from the server" variant="repository" />
    </section>
  );
}
