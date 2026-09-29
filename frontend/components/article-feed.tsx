"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import type { PublishedArticle } from "@/lib/backend";

type SortOrder = "desc" | "asc";

type ArticleMetadata = {
  description: string;
  tags: string[];
};

function articleTitle(article: PublishedArticle) {
  return article.title || article.source_path.split("/").at(-1)?.replace(/\.md$/i, "") || "Untitled article";
}

function articleMetadata(article: PublishedArticle): ArticleMetadata {
  if (typeof article.frontmatter !== "object" || article.frontmatter === null) {
    return { description: "", tags: [] };
  }

  const frontmatter = article.frontmatter as Record<string, unknown>;
  return {
    description: typeof frontmatter.description === "string" ? frontmatter.description : "",
    tags: Array.isArray(frontmatter.tags) ? frontmatter.tags.filter((tag): tag is string => typeof tag === "string") : [],
  };
}

function publishedTimestamp(value: string) {
  const timestamp = Date.parse(value);
  return Number.isNaN(timestamp) ? null : timestamp;
}

function displayDate(value: string) {
  const timestamp = publishedTimestamp(value);
  if (timestamp === null) return "Date unavailable";
  return new Intl.DateTimeFormat("en", { dateStyle: "medium" }).format(timestamp);
}

function readingTime(markdown: string) {
  const words = markdown.trim().split(/\s+/).filter(Boolean).length;
  return `${Math.max(1, Math.ceil(words / 200))} min read`;
}

function sortArticles(articles: PublishedArticle[], order: SortOrder) {
  return [...articles].sort((first, second) => {
    const firstTimestamp = publishedTimestamp(first.published_at);
    const secondTimestamp = publishedTimestamp(second.published_at);
    if (firstTimestamp === null && secondTimestamp === null) return articleTitle(first).localeCompare(articleTitle(second));
    if (firstTimestamp === null) return 1;
    if (secondTimestamp === null) return -1;

    const dateDifference = order === "desc"
      ? secondTimestamp - firstTimestamp
      : firstTimestamp - secondTimestamp;
    return dateDifference || articleTitle(first).localeCompare(articleTitle(second));
  });
}

export function ArticleFeed({ username, articles, loadFailed }: { username: string; articles: PublishedArticle[]; loadFailed: boolean }) {
  const [sortOrder, setSortOrder] = useState<SortOrder>("desc");
  const sortedArticles = useMemo(() => sortArticles(articles, sortOrder), [articles, sortOrder]);

  return (
    <main id="main-content" className="article-feed">
      <div className="article-feed-frame">
        <header className="article-feed-header">
          <div>
            <h1>Hello, @{username}.</h1>
            <p>All articles published through SourceInk.</p>
          </div>
          <Link className="button button-small" href="/dashboard/repositories">Manage sources</Link>
        </header>

        <section className="feed-section" aria-labelledby="published-articles-heading">
          <header className="feed-section-heading">
            <h2 id="published-articles-heading">Published articles</h2>
            <div className="feed-section-controls">
              <span className="feed-count">{articles.length} {articles.length === 1 ? "article" : "articles"}</span>
              <label className="feed-sort">
                <span>Sort by</span>
                <select value={sortOrder} onChange={(event) => setSortOrder(event.target.value as SortOrder)} disabled={articles.length < 2}>
                  <option value="desc">Newest first</option>
                  <option value="asc">Oldest first</option>
                </select>
              </label>
            </div>
          </header>
          {loadFailed ? (
            <div className="feed-empty" role="alert">
              <h3>SourceInk couldn&apos;t load published articles</h3>
              <p>Refresh the page to try again.</p>
            </div>
          ) : sortedArticles.length === 0 ? (
            <div className="feed-empty">
              <h3>No published articles yet</h3>
              <p>Publish a repository draft and it will appear here.</p>
              <Link className="text-link" href="/dashboard">Review article drafts</Link>
            </div>
          ) : (
            <div className="feed-list">
              {sortedArticles.map((article) => {
                const metadata = articleMetadata(article);
                const title = articleTitle(article);
                const articleHref = `/articles/${encodeURIComponent(article.slug)}`;
                return (
                  <article className="feed-article" key={article.id}>
                    <div className="feed-article-body">
                      <p className="feed-topic">{metadata.tags[0] ?? "Published article"}</p>
                      <h3><Link href={articleHref}>{title}</Link></h3>
                      <p className="feed-description">{metadata.description || "No description was saved with this publication."}</p>
                      <p className="feed-meta">
                        <span>By @{article.author_username || username}</span>
                        <span aria-hidden="true">·</span>
                        <time dateTime={article.published_at}>{displayDate(article.published_at)}</time>
                        <span aria-hidden="true">·</span>
                        <span>{readingTime(article.markdown)}</span>
                      </p>
                    </div>
                    <Link className="feed-read-link" href={articleHref} aria-label={`Read ${title}`}>Read article</Link>
                  </article>
                );
              })}
            </div>
          )}
        </section>
      </div>
    </main>
  );
}
