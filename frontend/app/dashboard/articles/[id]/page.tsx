import type { Metadata } from "next";
import Link from "next/link";
import { cookies } from "next/headers";
import { notFound } from "next/navigation";
import { MarkdownViewer } from "@/components/markdown-viewer";
import { PublishArticleButton } from "@/components/publish-article-button";
import { getArticles, type PublishedArticle, type UnpublishedArticle } from "@/lib/backend";

export const metadata: Metadata = { title: "Article configuration" };

type ArticlePageProps = {
  params: Promise<{ id: string }>;
};

type ConfigurationItem = {
  label: string;
  value: string;
  code?: boolean;
};

function displayDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "Not recorded";
  return new Intl.DateTimeFormat("en", { dateStyle: "medium", timeStyle: "short" }).format(date);
}

function articleTitle(article: PublishedArticle | UnpublishedArticle) {
  return article.title || article.source_path.split("/").at(-1)?.replace(/\.md$/i, "") || "Untitled article";
}

function Configuration({ items }: { items: ConfigurationItem[] }) {
  return (
    <dl className="article-configuration">
      {items.map((item) => (
        <div key={item.label}>
          <dt>{item.label}</dt>
          <dd className={item.code ? "mono" : undefined}>{item.value || "Not set"}</dd>
        </div>
      ))}
    </dl>
  );
}

export default async function ArticlePage({ params }: ArticlePageProps) {
  const { id } = await params;
  const cookieStore = await cookies();
  let articles;

  try {
    articles = await getArticles(cookieStore.toString());
  } catch {
    return (
      <section className="app-page app-page-narrow" aria-labelledby="article-error-heading">
        <Link className="article-back-link" href="/dashboard">Back to articles</Link>
        <div className="empty-row empty-row-full" role="alert">
          <div><h1 id="article-error-heading">SourceInk couldn&apos;t load this article</h1><p>Go back to the article list and try again.</p></div>
        </div>
      </section>
    );
  }

  const draft = articles.unpublished_articles.find((article) => article.id === id);
  const published = articles.published_articles.find((article) => article.id === id);
  if (!draft && !published) notFound();

  const article = draft ?? published!;
  const isDraft = Boolean(draft);
  const existingPublication = draft
    ? articles.published_articles.find((candidate) => candidate.unpublished_article_id === draft.id)
    : undefined;
  const isPublishedVersionCurrent = Boolean(
    draft && existingPublication?.git_blob_sha === draft.git_blob_sha,
  );
  const publishDisabledReason = draft
    ? draft.validation_error
      ? "Fix the frontmatter error before publishing."
      : !draft.present
        ? "This source file is no longer present in the repository."
        : !draft.title.trim() || !draft.slug.trim() || !draft.publish_mode
          ? "Add a title, slug, and publish mode before publishing."
          : isPublishedVersionCurrent
            ? "This repository version is already published."
          : undefined
    : undefined;
  const configuration: ConfigurationItem[] = draft ? [
    { label: "Article ID", value: draft.id, code: true },
    { label: "Repository ID", value: draft.repository_id, code: true },
    { label: "Source path", value: draft.source_path, code: true },
    { label: "Git blob SHA", value: draft.git_blob_sha, code: true },
    { label: "Slug", value: draft.slug, code: true },
    { label: "Publish mode", value: draft.publish_mode },
    { label: "Present in source", value: draft.present ? "Yes" : "No" },
    { label: "Discovered", value: displayDate(draft.discovered_at) },
    { label: "Last updated", value: displayDate(draft.updated_at) },
  ] : [
    { label: "Article ID", value: published!.id, code: true },
    { label: "Draft ID", value: published!.unpublished_article_id ?? "Not linked", code: true },
    { label: "Owner ID", value: published!.owner_id, code: true },
    { label: "Repository ID", value: published!.repository_id ?? "Source access removed", code: true },
    { label: "Source path", value: published!.source_path, code: true },
    { label: "Git blob SHA", value: published!.git_blob_sha, code: true },
    { label: "Slug", value: published!.slug, code: true },
    { label: "Publish mode", value: published!.publish_mode },
    { label: "Source state", value: published!.source_state.replace("_", " ") },
    { label: "Published", value: displayDate(published!.published_at) },
    { label: "Created", value: displayDate(published!.created_at) },
    { label: "Last updated", value: displayDate(published!.updated_at) },
  ];

  return (
    <article className="app-page article-inspector" aria-labelledby="article-title">
      <Link className="article-back-link" href="/dashboard">Back to articles</Link>
      <header className="article-inspector-heading">
        <div>
          <h1 id="article-title">{articleTitle(article)}</h1>
          <p>{isDraft ? "Repository draft configuration" : "Published snapshot configuration"}</p>
        </div>
        <div className="article-heading-actions">
          <span className={`article-status${draft?.validation_error || (published && published.source_state !== "available") ? " article-status-error" : ""}`}>
            {draft?.validation_error ? "Needs attention" : isDraft ? "Draft" : published!.source_state.replace("_", " ")}
          </span>
          {draft && (
            <PublishArticleButton
              draftId={draft.id}
              hasPublishedCopy={Boolean(existingPublication)}
              disabledReason={publishDisabledReason}
            />
          )}
        </div>
      </header>

      {draft?.validation_error && (
        <section className="article-validation" aria-labelledby="validation-heading" role="alert">
          <h2 id="validation-heading">Frontmatter needs attention</h2>
          <p>{draft.validation_error}</p>
        </section>
      )}

      <section className="article-inspector-section" aria-labelledby="configuration-heading">
        <header><h2 id="configuration-heading">Configuration</h2><p>IDs, source details, and publishing fields saved in SourceInk.</p></header>
        <Configuration items={configuration} />
      </section>

      {draft && (
        <section className="article-inspector-section" aria-labelledby="frontmatter-heading">
          <header><h2 id="frontmatter-heading">Frontmatter</h2><p>Fields SourceInk read from the repository file.</p></header>
          <Configuration items={[
            { label: "Title", value: draft.title },
            { label: "Description", value: draft.description },
            { label: "Tags", value: draft.tags?.join(", ") || "None" },
          ]} />
        </section>
      )}

      {published?.frontmatter !== undefined && (
        <section className="article-inspector-section" aria-labelledby="frontmatter-json-heading">
          <header><h2 id="frontmatter-json-heading">Frontmatter</h2><p>Metadata saved with this published copy.</p></header>
          <pre className="article-code"><code>{JSON.stringify(published.frontmatter, null, 2)}</code></pre>
        </section>
      )}

      <section className="article-inspector-section" aria-labelledby="markdown-heading">
        <header><h2 id="markdown-heading">Markdown content</h2><p>{isDraft ? "Preview the last repository scan or inspect its source." : "Preview this publication or inspect its saved source."}</p></header>
        <MarkdownViewer markdown={draft ? draft.content : published!.markdown} />
      </section>
    </article>
  );
}
