import Link from "next/link";
import { notFound } from "next/navigation";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { PublicPageShell } from "@/components/public-page-shell";
import { getPublishedArticles, type PublishedArticle } from "@/lib/backend";

type PublishedArticlePageProps = {
  params: Promise<{ slug: string }>;
};

function description(article: PublishedArticle) {
  if (typeof article.frontmatter !== "object" || article.frontmatter === null) return "";
  const value = (article.frontmatter as Record<string, unknown>).description;
  return typeof value === "string" ? value : "";
}

function displayDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "Date unavailable";
  return new Intl.DateTimeFormat("en", { dateStyle: "medium" }).format(date);
}

function readingTime(markdown: string) {
  const words = markdown.trim().split(/\s+/).filter(Boolean).length;
  return `${Math.max(1, Math.ceil(words / 200))} min read`;
}

export default async function PublishedArticlePage({ params }: PublishedArticlePageProps) {
  const { slug } = await params;
  let articles: PublishedArticle[];
  try {
    articles = await getPublishedArticles();
  } catch {
    return (
      <PublicPageShell>
        <section className="reader-page" role="alert">
          <header className="reader-header">
            <h1>SourceInk couldn&apos;t load this article</h1>
            <p className="reader-deck">Return to the article list and try again.</p>
          </header>
          <div className="reader-source"><Link className="text-link" href="/">Back to articles</Link></div>
        </section>
      </PublicPageShell>
    );
  }

  const article = articles.find((candidate) => candidate.slug === slug);
  if (!article) notFound();

  return (
    <PublicPageShell>
      <article className="reader-page">
        <header className="reader-header">
          <Link className="reader-publication" href="/">SourceInk</Link>
          <h1>{article.title}</h1>
          {description(article) && <p className="reader-deck">{description(article)}</p>}
          <div className="reader-meta">
            <span>By @{article.author_username}</span>
            <span>{readingTime(article.markdown)}</span>
            <span>Published {displayDate(article.published_at)}</span>
          </div>
        </header>
        <div className="reader-source">
          <span>Source <code>{article.source_path}</code></span>
          <span>Revision <code>{article.git_blob_sha.slice(0, 8)}</code></span>
        </div>
        <div className="reader-body">
          <ReactMarkdown remarkPlugins={[remarkGfm]} skipHtml>{article.markdown}</ReactMarkdown>
        </div>
      </article>
    </PublicPageShell>
  );
}
