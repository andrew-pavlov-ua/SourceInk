import Link from "next/link";
import { cookies } from "next/headers";
import { notFound } from "next/navigation";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { ArticleReview } from "@/components/article-review";
import { PublicPageShell } from "@/components/public-page-shell";
import { articleMetadata, articleReadingTime } from "@/lib/article-display";
import { getPublishedArticle, type ReviewedPublishedArticle } from "@/lib/backend";

type PublishedArticlePageProps = {
  params: Promise<{ slug: string }>;
};

function displayDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "Date unavailable";
  return new Intl.DateTimeFormat("en", { dateStyle: "medium" }).format(date);
}

export default async function PublishedArticlePage({ params }: PublishedArticlePageProps) {
  const { slug } = await params;
  let article: ReviewedPublishedArticle | null;
  try {
    const cookieStore = await cookies();
    article = await getPublishedArticle(slug, cookieStore.toString());
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
  if (!article) notFound();
  const description = articleMetadata(article.frontmatter).description;

  return (
    <PublicPageShell>
      <article className="reader-page">
        <header className="reader-header">
          <Link className="reader-publication" href="/">SourceInk</Link>
          <h1>{article.title}</h1>
          {description && <p className="reader-deck">{description}</p>}
          <div className="reader-meta">
            <span>By @{article.author_username}</span>
            <span>{articleReadingTime(article.markdown)}</span>
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
        <ArticleReview articleId={article.id} gitBlobSha={article.git_blob_sha} initialSummary={article.reviews} />
      </article>
    </PublicPageShell>
  );
}
