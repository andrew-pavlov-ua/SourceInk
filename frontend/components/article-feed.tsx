"use client";

import Link from "next/link";
import { useEffect, useId, useRef, useState } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { articleMetadata, articleReadingTime, articleTitle } from "@/lib/article-display";
import type { PublishedArticleOrder, ReviewedPublishedArticle } from "@/lib/backend";

type SortOrder = PublishedArticleOrder;

const sortOptions: { value: SortOrder; label: string; description: string }[] = [
  { value: "newest", label: "Newest first", description: "Recently published" },
  { value: "oldest", label: "Oldest first", description: "Earliest published" },
  { value: "rating-desc", label: "Highest rated", description: "Most approved" },
  { value: "rating-asc", label: "Lowest rated", description: "Needs attention" },
];

function publishedTimestamp(value: string) {
  const timestamp = Date.parse(value);
  return Number.isNaN(timestamp) ? null : timestamp;
}

function displayDate(value: string) {
  const timestamp = publishedTimestamp(value);
  if (timestamp === null) return "Date unavailable";
  return new Intl.DateTimeFormat("en", { dateStyle: "medium" }).format(timestamp);
}

export function ArticleFeed({ username, articles, loadFailed, order }: { username: string; articles: ReviewedPublishedArticle[]; loadFailed: boolean; order: SortOrder }) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const [sortMenuOpen, setSortMenuOpen] = useState(false);
  const sortMenuID = useId();
  const sortControlRef = useRef<HTMLDivElement>(null);
  const sortButtonRef = useRef<HTMLButtonElement>(null);
  const sortOptionRefs = useRef<(HTMLButtonElement | null)[]>([]);
  const selectedSort = sortOptions.find((option) => option.value === order) ?? sortOptions[0];

  useEffect(() => {
    if (!sortMenuOpen) return;

    const handlePointerDown = (event: PointerEvent) => {
      if (!sortControlRef.current?.contains(event.target as Node)) setSortMenuOpen(false);
    };
    document.addEventListener("pointerdown", handlePointerDown);
    return () => document.removeEventListener("pointerdown", handlePointerDown);
  }, [sortMenuOpen]);

  useEffect(() => {
    if (!sortMenuOpen) return;
    const selectedIndex = sortOptions.findIndex((option) => option.value === order);
    sortOptionRefs.current[selectedIndex]?.focus();
  }, [sortMenuOpen, order]);

  function changeSortOrder(nextOrder: SortOrder) {
    const params = new URLSearchParams(searchParams.toString());
    if (nextOrder === "newest") params.delete("sort");
    else params.set("sort", nextOrder);
    const query = params.toString();
    router.replace(query ? `${pathname}?${query}` : pathname);
  }

  function chooseSortOrder(nextOrder: SortOrder) {
    setSortMenuOpen(false);
    changeSortOrder(nextOrder);
  }

  function moveSortFocus(currentIndex: number, direction: number) {
    const nextIndex = (currentIndex + direction + sortOptions.length) % sortOptions.length;
    sortOptionRefs.current[nextIndex]?.focus();
  }

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
              <div className="feed-sort">
                <span>Sort by</span>
                <div className="feed-sort-control" ref={sortControlRef}>
                  <button
                    ref={sortButtonRef}
                    className="feed-sort-trigger"
                    type="button"
                    aria-haspopup="menu"
                    aria-expanded={sortMenuOpen}
                    aria-controls={sortMenuID}
                    onClick={() => setSortMenuOpen((open) => !open)}
                    onKeyDown={(event) => {
                      if (event.key === "ArrowDown" || event.key === "ArrowUp") {
                        event.preventDefault();
                        setSortMenuOpen(true);
                      }
                    }}
                  >
                    <span>{selectedSort.label}</span>
                    <svg className="feed-sort-chevron" aria-hidden="true" viewBox="0 0 12 12" fill="none">
                      <path d="m3 4.5 3 3 3-3" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
                    </svg>
                  </button>
                  {sortMenuOpen ? (
                    <div className="feed-sort-menu" id={sortMenuID} role="menu" aria-label="Sort published articles">
                      {sortOptions.map((option, index) => {
                        const selected = option.value === order;
                        return (
                          <button
                            ref={(element) => { sortOptionRefs.current[index] = element; }}
                            className={selected ? "feed-sort-option feed-sort-option-selected" : "feed-sort-option"}
                            key={option.value}
                            type="button"
                            role="menuitemradio"
                            aria-checked={selected}
                            onClick={() => chooseSortOrder(option.value)}
                            onKeyDown={(event) => {
                              if (event.key === "ArrowDown") {
                                event.preventDefault();
                                moveSortFocus(index, 1);
                              } else if (event.key === "ArrowUp") {
                                event.preventDefault();
                                moveSortFocus(index, -1);
                              } else if (event.key === "Home") {
                                event.preventDefault();
                                sortOptionRefs.current[0]?.focus();
                              } else if (event.key === "End") {
                                event.preventDefault();
                                sortOptionRefs.current[sortOptions.length - 1]?.focus();
                              } else if (event.key === "Escape") {
                                event.preventDefault();
                                setSortMenuOpen(false);
                                sortButtonRef.current?.focus();
                              }
                            }}
                          >
                            <span>{option.label}</span>
                            <small>{option.description}</small>
                            {selected ? <svg aria-hidden="true" viewBox="0 0 16 16" fill="none"><path d="m3.5 8 2.8 2.8 6.2-6.2" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" /></svg> : null}
                          </button>
                        );
                      })}
                    </div>
                  ) : null}
                </div>
              </div>
            </div>
          </header>
          {loadFailed ? (
            <div className="feed-empty" role="alert">
              <h3>SourceInk couldn&apos;t load published articles</h3>
              <p>Refresh the page to try again.</p>
            </div>
          ) : articles.length === 0 ? (
            <div className="feed-empty">
              <h3>No published articles yet</h3>
              <p>Publish a repository draft and it will appear here.</p>
              <Link className="text-link" href="/dashboard">Review article drafts</Link>
            </div>
          ) : (
            <div className="feed-list">
              {articles.map((article) => {
                const metadata = articleMetadata(article.frontmatter);
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
                        <time dateTime={article.published_at}>{displayDate(article.published_at)}</time>
                        <span>{articleReadingTime(article.markdown)}</span>
                        <span className="feed-review-signals" aria-label={`${article.reviews.approveCount} approvals and ${article.reviews.requestChangesCount} change requests`}>
                          <span className="feed-review-approvals">{article.reviews.approveCount.toLocaleString()} approved</span>
                          <span className="feed-review-changes">{article.reviews.requestChangesCount.toLocaleString()} changes</span>
                        </span>
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
