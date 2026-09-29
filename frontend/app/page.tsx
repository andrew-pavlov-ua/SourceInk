import Link from "next/link";
import { cookies } from "next/headers";
import { ArticleFeed } from "@/components/article-feed";
import { ProductPreview } from "@/components/product-preview";
import { SiteHeader } from "@/components/site-header";
import { getCurrentUser, getPublishedArticles, SESSION_COOKIE_NAME, type PublishedArticle, type User } from "@/lib/backend";

const workflow = [
  { step: "01", title: "Install the App", body: "Choose the repositories SourceInk may read on GitHub." },
  { step: "02", title: "Open a draft", body: "Check its frontmatter, source path, and Markdown." },
  { step: "03", title: "Publish", body: "Choose the draft that readers should receive." },
];

export const dynamic = "force-dynamic";

type HomepageData = {
  user: User | null;
  publishedArticles: PublishedArticle[];
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
    const publishedArticles = await getPublishedArticles();
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
        <section className="landing-hero">
          <div className="landing-frame hero-layout">
            <div className="hero-copy">
              <p className="page-kicker"><span aria-hidden="true" />Git-first publishing</p>
              <h1>Publish an article<br />from a specific commit.</h1>
              <p className="hero-lede">Write in GitHub. SourceInk keeps your working draft separate from the copy readers see.</p>
              <div className="hero-actions">
                <Link className="button" href="/register">Create an account</Link>
                <a className="text-link" href="#workflow">See how it works</a>
              </div>
              <p className="hero-footnote">Your repository owns the source.</p>
            </div>
            <ProductPreview />
          </div>
        </section>

        <section className="platform-rail" aria-label="Platform properties">
          <div className="landing-frame">
            <div><span>Source</span><strong>GitHub repositories</strong></div>
            <div><span>Revision</span><strong>Commit-linked publishing</strong></div>
            <div><span>Delivery</span><strong>Stored public output</strong></div>
          </div>
        </section>

        <section className="workflow landing-frame" id="workflow">
          <header className="workflow-heading">
            <p className="page-kicker">Publishing workflow</p>
            <h2>Review the repository copy before you publish it.</h2>
            <p>A push updates the draft. You choose when readers get that change.</p>
          </header>
          <ol className="workflow-list">
            {workflow.map((item) => (
              <li key={item.step}>
                <span>{item.step}</span>
                <div><h3>{item.title}</h3><p>{item.body}</p></div>
              </li>
            ))}
          </ol>
        </section>

        <section className="landing-cta landing-frame" id="pricing">
          <div><p className="page-kicker">Start with the repository</p><h2>Connect GitHub and review the Markdown SourceInk finds.</h2></div>
          <Link className="button" href="/register">Create an account</Link>
        </section>
      </main>
    </>
  );
}
