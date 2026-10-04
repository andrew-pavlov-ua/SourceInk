import type { Metadata } from "next";
import Link from "next/link";
import { cookies } from "next/headers";
import { getRepositories, type Repository } from "@/lib/backend";

export const metadata: Metadata = { title: "Repositories" };

type RepositoriesPageProps = {
  searchParams: Promise<{ github?: string }>;
};

export default async function RepositoriesPage({ searchParams }: RepositoriesPageProps) {
  const { github } = await searchParams;
  const cookieStore = await cookies();
  let repositories: Repository[];
  let loadFailed = false;

  try {
    repositories = await getRepositories(cookieStore.toString());
  } catch {
    repositories = [];
    loadFailed = true;
  }
  const hasConnectedRepositories = repositories.length > 0;
  const connectionMessage = github === "existing"
    ? "GitHub App connected. SourceInk refreshed the repository list."
    : github === "connected"
      ? "GitHub App connected. SourceInk saved your repository list."
      : github === "refresh-failed"
        ? "SourceInk couldn't refresh the repository list. Open GitHub access and try again."
        : "";
  const refreshFailed = github === "refresh-failed";
  return (
    <section className="app-page" aria-labelledby="repositories-heading">
      <header className="app-page-heading">
        <div><h1 id="repositories-heading">Repositories</h1><p>{repositories.length} connected GitHub {repositories.length === 1 ? "repository" : "repositories"}.</p></div>
        <Link className="button button-small" href={hasConnectedRepositories ? "/dashboard/repositories/connect?mode=manage" : "/dashboard/repositories/connect"}>
          {hasConnectedRepositories ? "Connect other repositories" : "Connect repositories"}
        </Link>
      </header>
      {connectionMessage && (
        <div className={`repository-status${refreshFailed ? " repository-status-error" : ""}`} role={refreshFailed ? "alert" : "status"}>
          <span aria-hidden="true" />
          <p>{connectionMessage}</p>
        </div>
      )}
      {loadFailed ? (
        <div className="empty-row empty-row-full" role="alert">
          <div><h2>SourceInk couldn&apos;t load your repositories</h2><p>Refresh the page. If it fails again, reconnect the GitHub App.</p></div>
        </div>
      ) : repositories.length === 0 ? (
        <div className="empty-row empty-row-full">
          <div><h2>No repositories connected</h2><p>Install the GitHub App and choose the repositories you want SourceInk to read.</p></div>
        </div>
      ) : (
        <div className="data-list" aria-label="Connected GitHub repositories">
          <div className="data-list-head" aria-hidden="true"><span>Repository</span><span>Default branch</span><span>Visibility</span><span>State</span></div>
          {repositories.map((repository) => (
            <article className="data-row" key={repository.github_id}>
              <div><strong>{repository.full_name}</strong><p>Owned by {repository.owner}</p></div>
              <code>{repository.default_branch || "No default branch"}</code>
              <span>{repository.private ? "Private" : "Public"}</span>
              <span>{repository.archived ? "Archived" : "Active"}</span>
            </article>
          ))}
        </div>
      )}
    </section>
  );
}
