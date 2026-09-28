export function ProductPreview() {
  return (
    <section className="product-preview" id="product-preview" aria-label="SourceInk product preview">
      <header className="preview-topbar">
        <div className="preview-brand"><span aria-hidden="true">S</span><strong>SourceInk</strong></div>
        <nav aria-label="Preview navigation"><span>Articles</span><span>Repositories</span></nav>
        <span className="preview-avatar" aria-hidden="true">A</span>
      </header>

      <div className="preview-body">
        <aside className="preview-sidebar" aria-label="Repository navigation">
          <p>Repositories</p>
          <strong><span aria-hidden="true">⌘</span> field-notes</strong>
          <span>Articles</span>
          <span>Settings</span>
        </aside>

        <div className="preview-main">
          <div className="preview-breadcrumb"><span>field-notes</span><span aria-hidden="true">/</span><strong>articles</strong></div>
          <div className="preview-heading">
            <div><h2>Articles</h2><p>Source commits and published copies</p></div>
            <button type="button" disabled>github.com/alice/field-notes <span aria-hidden="true">⌄</span></button>
          </div>
          <nav className="preview-tabs" aria-label="Article preview sections"><span className="preview-tab-active">Articles</span><span>Source files</span></nav>

          <div className="preview-list" aria-label="Article revision list">
            <div className="preview-list-head" aria-hidden="true"><span>Source file</span><span>Revisions</span><span>Actions</span></div>
            <article className="preview-row">
              <div className="preview-source"><strong>postgres-indexes.md</strong><span>articles/postgres-indexes.md</span></div>
              <dl className="preview-revisions">
                <div><dt>Published</dt><dd>a17c29f</dd></div>
                <div><dt>Latest</dt><dd>bc832f1</dd></div>
              </dl>
              <div className="preview-controls">
                <span className="preview-update"><i aria-hidden="true" />Update available</span>
                <div><button type="button" disabled>Preview</button><button type="button" disabled>Diff</button><button className="preview-publish" type="button" disabled>Publish</button></div>
              </div>
            </article>
          </div>
        </div>
      </div>
    </section>
  );
}
