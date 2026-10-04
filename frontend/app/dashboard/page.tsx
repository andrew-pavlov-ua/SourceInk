import type { Metadata } from "next";
import Link from "next/link";
import { cookies } from "next/headers";
import { FrontmatterGuide } from "@/components/frontmatter-guide";
import { getArticles, type PublishedArticle, type UnpublishedArticle } from "@/lib/backend";

export const metadata: Metadata = { title: "Articles" };

const frontmatterExample = `---
title: Postgres indexes that scale
slug: postgres-indexes
description: How to choose indexes without slowing writes.
tags:
  - postgresql
  - performance
publish_mode: manual
---

# Postgres indexes that scale`;

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

      <FrontmatterGuide example={frontmatterExample} />

      {loadFailed ? (
        <section className="dashboard-unavailable" role="alert" aria-labelledby="dashboard-unavailable-heading">
          <div className="dashboard-unavailable-mark" aria-hidden="true">
            <svg viewBox="0 0 24 24"><path d="M8.5 16.5 12 13m0 0 3.5 3.5M12 13V4.5M5.5 9.5a7.5 7.5 0 1 0 13 0" /></svg>
          </div>
          <div>
            <h2 id="dashboard-unavailable-heading">The dashboard is temporarily unavailable</h2>
            <p>SourceInk could not reach its API. No repository or publishing changes were attempted. Try again in a moment, or start the backend if you are running SourceInk locally.</p>
            <form action="/dashboard" method="get">
              <button className="button button-small dashboard-unavailable-retry" type="submit">
                Try again
                <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M15.5 7.5V4.8m0 0h-2.7m2.7 0-2 2A6 6 0 1 0 16 12" /></svg>
              </button>
            </form>
          </div>
        </section>
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
