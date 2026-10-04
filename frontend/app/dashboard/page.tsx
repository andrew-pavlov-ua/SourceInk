import type { Metadata } from "next";
import Link from "next/link";
import { cookies } from "next/headers";
import { getArticles, type PublishedArticle, type UnpublishedArticle } from "@/lib/backend";

export const metadata: Metadata = { title: "Articles" };

function articleTitle(article: PublishedArticle | UnpublishedArticle) {
  return article.title || article.source_path.split("/").at(-1)?.replace(/\.md$/i, "") || "Untitled article";
}

function displayDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "Date unavailable";
  return new Intl.DateTimeFormat("en", { dateStyle: "medium" }).format(date);
}

export default async function DashboardPage() {
  const cookieStore = await cookies();
  let publishedArticles: PublishedArticle[] = [];
  let unpublishedArticles: UnpublishedArticle[] = [];
  let loadFailed = false;

  try {
    const articles = await getArticles(cookieStore.toString());
    publishedArticles = articles.published_articles;
    unpublishedArticles = articles.unpublished_articles;
  } catch {
    loadFailed = true;
  }

  const articleCount = publishedArticles.length + unpublishedArticles.length;

  return (
    <section className="app-page" aria-labelledby="articles-heading">
      <header className="articles-heading">
        <div>
          <h1 id="articles-heading">Articles</h1>
          <p>{loadFailed ? "Drafts and published articles from your repositories." : `${unpublishedArticles.length} ${unpublishedArticles.length === 1 ? "draft" : "drafts"}, ${publishedArticles.length} published.`}</p>
        </div>
      </header>

      {loadFailed ? (
        <div className="empty-row empty-row-full" role="alert">
          <div><h2>SourceInk couldn&apos;t load your articles</h2><p>Refresh the page. If it fails again, check the GitHub App&apos;s repository access.</p></div>
        </div>
      ) : articleCount === 0 ? (
        <div className="empty-row empty-row-full">
          <div><h2>No article drafts found</h2><p>Connect a repository that contains Markdown with frontmatter.</p></div>
          <Link className="button button-small empty-row-action" href="/dashboard/repositories">Manage repositories</Link>
        </div>
      ) : (
        <div className="article-groups">
          {unpublishedArticles.length > 0 && (
            <section className="article-group" aria-labelledby="draft-articles-heading">
              <header className="article-group-heading">
                <div><h2 id="draft-articles-heading">Repository drafts</h2><p>SourceInk found these files in GitHub. Readers cannot see them.</p></div>
                <span>{unpublishedArticles.length}</span>
              </header>
              <div className="article-table">
                <div className="article-table-head" aria-hidden="true"><span>Article</span><span>Source</span><span>Mode</span><span>State</span></div>
                {unpublishedArticles.map((article) => (
                  <Link className="article-row" href={`/dashboard/articles/${encodeURIComponent(article.id)}`} key={article.id} aria-label={`Inspect ${articleTitle(article)}`}>
                    <div>
                      <strong>{articleTitle(article)}</strong>
                      <p>{article.validation_error ? `Draft warning: ${article.validation_error}` : article.description || "No description in frontmatter"}</p>
                    </div>
                    <code>{article.source_path}</code>
                    <span>{article.publish_mode || "Not set"}</span>
                    <span className={article.validation_error ? "article-state article-state-warning" : "article-state"}>{article.validation_error ? "Warning" : "Draft"}</span>
                  </Link>
                ))}
              </div>
            </section>
          )}

          {publishedArticles.length > 0 && (
            <section className="article-group" aria-labelledby="published-articles-heading">
              <header className="article-group-heading">
                <div><h2 id="published-articles-heading">Published</h2><p>SourceInk serves these copies without calling GitHub.</p></div>
                <span>{publishedArticles.length}</span>
              </header>
              <div className="article-table">
                <div className="article-table-head" aria-hidden="true"><span>Article</span><span>Source</span><span>Published</span><span>State</span></div>
                {publishedArticles.map((article) => (
                  <Link className="article-row" href={`/dashboard/articles/${encodeURIComponent(article.id)}`} key={article.id} aria-label={`Inspect ${articleTitle(article)}`}>
                    <div><strong>{articleTitle(article)}</strong><p>/{article.slug}</p></div>
                    <code>{article.source_path}</code>
                    <span>{displayDate(article.published_at)}</span>
                    <span className={`article-state${article.source_state === "available" ? "" : " article-state-error"}`}>{article.source_state.replace("_", " ")}</span>
                  </Link>
                ))}
              </div>
            </section>
          )}
        </div>
      )}
    </section>
  );
}
