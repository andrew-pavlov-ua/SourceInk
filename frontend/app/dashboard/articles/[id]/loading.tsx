import { ServerLoading } from "@/components/server-loading";

export default function ArticleLoading() {
  return (
    <section className="app-page" aria-labelledby="article-loading-heading">
      <header className="app-page-heading">
        <div><h1 id="article-loading-heading">Article</h1><p>Frontmatter, source content, and publication status.</p></div>
      </header>
      <ServerLoading label="Loading article from the server" rows={4} />
    </section>
  );
}
