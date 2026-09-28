import Link from "next/link";

type FeedArticle = {
  topic: string;
  title: string;
  description: string;
  author: string;
  publishedAt: string;
  readTime: string;
};

const sampleArticles: FeedArticle[] = [
  {
    topic: "Distributed systems",
    title: "The costs of a harmless retry",
    description: "A retry can charge a card twice, send two emails, or record the same click again.",
    author: "Maya Chen",
    publishedAt: "Today",
    readTime: "7 min read",
  },
  {
    topic: "Frontend engineering",
    title: "Making loading states tell the truth",
    description: "A useful loading message says which request is running and leaves the surrounding page in place.",
    author: "Elias Romero",
    publishedAt: "Yesterday",
    readTime: "5 min read",
  },
  {
    topic: "Databases",
    title: "A migration is a public interface",
    description: "A schema change sets rules for old and new application versions during a deployment.",
    author: "Nora Okafor",
    publishedAt: "Sep 17",
    readTime: "9 min read",
  },
  {
    topic: "Writing in public",
    title: "Keep the source close to the explanation",
    description: "Keep examples beside the code and commit that they explain.",
    author: "Jon Bell",
    publishedAt: "Sep 15",
    readTime: "6 min read",
  },
];

export function ArticleFeed({ username }: { username: string }) {
  return (
    <main id="main-content" className="article-feed">
      <div className="article-feed-frame">
        <header className="article-feed-header">
          <div>
            <h1>Hello, @{username}.</h1>
            <p>Recent articles from publications you follow.</p>
          </div>
          <Link className="button button-small" href="/dashboard/repositories">Manage sources</Link>
        </header>

        <div className="feed-notice" role="note">
          <span aria-hidden="true" />
          <p><strong>Demo data.</strong> Connected repository articles appear in the dashboard, not in this feed.</p>
        </div>

        <section className="feed-section" aria-labelledby="recent-articles-heading">
          <header className="feed-section-heading">
            <h2 id="recent-articles-heading">Recent articles</h2>
            <span>{sampleArticles.length} sample entries</span>
          </header>
          <div className="feed-list">
            {sampleArticles.map((article) => (
              <article className="feed-article" key={article.title}>
                <div className="feed-article-body">
                  <p className="feed-topic">{article.topic}</p>
                  <h3><Link href="/articles/example">{article.title}</Link></h3>
                  <p className="feed-description">{article.description}</p>
                  <p className="feed-meta">By {article.author}<span aria-hidden="true">·</span>{article.publishedAt}<span aria-hidden="true">·</span>{article.readTime}</p>
                </div>
                <Link className="feed-read-link" href="/articles/example" aria-label={`Read ${article.title}`}>Read article</Link>
              </article>
            ))}
          </div>
        </section>
      </div>
    </main>
  );
}
