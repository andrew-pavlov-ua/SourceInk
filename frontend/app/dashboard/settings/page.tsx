import type { Metadata } from "next";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { getCurrentUser } from "@/lib/backend";
import { ConnectEmailForm } from "@/components/connect-email-form";
import { GitHubMark } from "@/components/github-mark";

export const metadata: Metadata = { title: "Settings" };

type SettingsPageProps = {
  searchParams: Promise<{ github?: string }>;
};

const githubMessages: Record<string, { text: string; error?: boolean }> = {
  connected: { text: "GitHub is now connected to this account." },
  "already-connected": { text: "This account already has a GitHub login." },
  conflict: { text: "That GitHub login belongs to another email account. Sign in to that account before changing the connection.", error: true },
  failed: { text: "SourceInk couldn't connect GitHub. Try again.", error: true },
};

export default async function SettingsPage({ searchParams }: SettingsPageProps) {
  const [{ github }, cookieStore] = await Promise.all([searchParams, cookies()]);
  const user = await getCurrentUser(cookieStore.toString());
  if (!user) redirect("/login");

  const message = github ? githubMessages[github] : undefined;
  const githubConnected = user.github_user_id !== undefined;

  return (
    <section className="app-page app-page-narrow" aria-labelledby="settings-heading">
      <header className="app-page-heading">
        <div><h1 id="settings-heading">Settings</h1><p>Manage your account and sign-in methods.</p></div>
      </header>

      {message && (
        <div className={`repository-status${message.error ? " repository-status-error" : ""}`} role={message.error ? "alert" : "status"}>
          <span aria-hidden="true" />
          <p>{message.text}</p>
        </div>
      )}

      <div className="settings-list">
        <section aria-labelledby="account-settings-heading">
          <div>
            <h2 id="account-settings-heading">Account</h2>
            <p>Your SourceInk identity stays the same when you add another sign-in method.</p>
          </div>
          <dl className="account-details">
            <div><dt>Username</dt><dd>@{user.username}</dd></div>
            <div><dt>Email</dt><dd>{user.email || "No email address"}</dd></div>
          </dl>
        </section>

        {!user.email && githubConnected && (
          <section aria-labelledby="email-settings-heading">
            <div>
              <h2 id="email-settings-heading">Email sign-in</h2>
              <p>Add an email and password so you can access this same account without GitHub.</p>
            </div>
            <ConnectEmailForm />
          </section>
        )}

        <section aria-labelledby="github-settings-heading">
          <div>
            <h2 id="github-settings-heading">GitHub login</h2>
            <p>Use GitHub or your email and password to open the same SourceInk account.</p>
          </div>
          <div className="github-account-card">
            <div className="github-account-copy">
              <strong>{githubConnected ? (user.github_login ? `@${user.github_login}` : "GitHub connected") : "Not connected"}</strong>
              <p>{githubConnected
                ? (user.email ? "You can sign in with either method." : "GitHub is currently your only sign-in method.")
                : "Existing GitHub-only repositories and articles will move to this account."}</p>
            </div>
            {!githubConnected && (
              <form action="/api/github/connect" method="get">
                <button className="button button-small" type="submit"><GitHubMark />Connect GitHub</button>
              </form>
            )}
          </div>
        </section>
      </div>
    </section>
  );
}
