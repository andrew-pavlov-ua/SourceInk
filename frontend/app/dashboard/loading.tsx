export default function DashboardLoading() {
  return (
    <section className="app-page" aria-labelledby="articles-loading-heading" aria-live="polite">
      <header className="articles-heading">
        <div><h1 id="articles-loading-heading">Articles</h1><p>Loading drafts and published articles.</p></div>
      </header>
      <div className="article-loading-list" aria-hidden="true">
        <span /><span /><span />
      </div>
    </section>
  );
}
