import Link from "next/link";
import { cookies } from "next/headers";
import { ArticleFeed } from "@/components/article-feed";
import { CommitHistory } from "@/components/commit-history";
import { SiteHeader } from "@/components/site-header";
import { getCurrentUser, getPublishedArticles, SESSION_COOKIE_NAME, type ReviewedPublishedArticle, type User } from "@/lib/backend";

const workflow = [
  { title: "Install the GitHub App", body: "Choose the repositories SourceInk may read. It finds every Markdown file with frontmatter." },
  { title: "Review the draft", body: "Check the frontmatter, the rendered Markdown, and the diff against the published copy." },
  { title: "Publish", body: "Readers get that exact version. Later pushes stay drafts, unless the article is set to auto-publish." },
];

export const dynamic = "force-dynamic";

type HomepageData = {
  user: User | null;
  publishedArticles: ReviewedPublishedArticle[];
  articlesLoadFailed: boolean;
};

async function getHomepageData(): Promise<HomepageData> {
  const cookieStore = await cookies();
  if (!cookieStore.has(SESSION_COOKIE_NAME)) {
    return { user: null, publishedArticles: [], articlesLoadFailed: false };
  }

  const cookieHeader = cookieStore.toString();
  let user: User | null;
  try {
    user = await getCurrentUser(cookieHeader);
  } catch {
    return { user: null, publishedArticles: [], articlesLoadFailed: false };
  }
  if (!user) return { user: null, publishedArticles: [], articlesLoadFailed: false };

  try {
    const publishedArticles = await getPublishedArticles(cookieHeader);
    return { user, publishedArticles, articlesLoadFailed: false };
  } catch {
    return { user, publishedArticles: [], articlesLoadFailed: true };
  }
}

export default async function HomePage() {
  const { user, publishedArticles, articlesLoadFailed } = await getHomepageData();

  if (user) {
    return (
      <>
        <SiteHeader />
        <ArticleFeed username={user.username} articles={publishedArticles} loadFailed={articlesLoadFailed} />
      </>
    );
  }

  return (
    <>
      <SiteHeader />
      <main id="main-content" className="landing">
        <section className="landing-hero landing-frame">
          <div className="hero-copy">
            <h1>Publish an article from a specific commit.</h1>
            <p className="hero-lede">Write Markdown in your GitHub repository. Every push updates a draft. Readers only see the version you publish.</p>
            <div className="hero-actions">
              <Link className="button" href="/register">Create an account</Link>
              <a className="text-link" href="#workflow">How it works</a>
            </div>
          </div>
          <CommitHistory />
        </section>

        <section className="workflow landing-frame" id="workflow" aria-labelledby="workflow-heading">
          <h2 id="workflow-heading">From repository to reader in three steps.</h2>
          <ol className="workflow-list">
            {workflow.map((item) => (
              <li key={item.title}>
                <h3>{item.title}</h3>
                <p>{item.body}</p>
              </li>
            ))}
          </ol>
        </section>

        <section className="landing-cta landing-frame">
          <h2>Your repository keeps the source. SourceInk keeps the published copy.</h2>
          <Link className="button" href="/register">Create an account</Link>
        </section>
      </main>
    </>
  );
}
