import type { Metadata } from "next";

export const metadata: Metadata = { title: "Connect repository" };

const installSteps = [
  ["1", "Install the GitHub App", "Choose the repositories SourceInk may read."],
  ["2", "Verify access", "Sign in to GitHub so SourceInk can verify the installation."],
  ["3", "Discover Markdown", "SourceInk scans your selection for Markdown with frontmatter."],
] as const;

const manageSteps = [
  ["1", "Choose repositories", "Change which repositories SourceInk can read."],
  ["2", "Verify access", "Sign in to GitHub so SourceInk can verify the installation."],
  ["3", "Save the list", "SourceInk adds your new choices and removes repositories you unchecked."],
] as const;

export default async function ConnectRepositoryPage({ searchParams }: { searchParams: Promise<{ mode?: string }> }) {
  const { mode } = await searchParams;
  const managing = mode === "manage";
  const steps = managing ? manageSteps : installSteps;

  return (
    <section className="app-page app-page-narrow" aria-labelledby="connect-heading">
      <header className="app-page-heading">
        <div><h1 id="connect-heading">{managing ? "Update GitHub access" : "Connect GitHub"}</h1><p>{managing ? "Choose a new repository list for this GitHub App installation." : "Choose the repositories SourceInk can read."}</p></div>
      </header>
      <ol className="connection-steps">
        {steps.map(([number, title, body], index) => (
          <li className={index === 0 ? "connection-step connection-step-active" : "connection-step"} key={number}>
            <span>{number}</span><div><h2>{title}</h2><p>{body}</p></div>
            {index === 0 && (
              <form className="connection-action" action="/api/github/install" method="get">
                <button className="button button-small" type="submit">{managing ? "Select other repositories" : "Install GitHub App"}</button>
              </form>
            )}
          </li>
        ))}
      </ol>
      <p className="articles-note">{managing ? "GitHub opens this installation’s repository access settings." : "GitHub shows the repository picker during App installation."}</p>
    </section>
  );
}
