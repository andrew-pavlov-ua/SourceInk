"use client";

import { useId, useState } from "react";

type FrontmatterGuideProps = {
  example: string;
};

export function FrontmatterGuide({ example }: FrontmatterGuideProps) {
  const [isOpen, setIsOpen] = useState(false);
  const contentID = useId();
  const toggleID = useId();

  return (
    <section className="frontmatter-guide" data-open={isOpen}>
      <h2 className="frontmatter-guide-heading">
        <button
          id={toggleID}
          className="frontmatter-guide-toggle"
          type="button"
          aria-expanded={isOpen}
          aria-controls={contentID}
          onClick={() => setIsOpen((open) => !open)}
        >
          <span>Article frontmatter</span>
          <svg className="frontmatter-guide-chevron" viewBox="0 0 20 20" aria-hidden="true">
            <path d="m7 5 5 5-5 5" />
          </svg>
        </button>
      </h2>
      <div id={contentID} className="frontmatter-guide-panel" aria-hidden={!isOpen} aria-labelledby={toggleID}>
        <div>
          <div className="frontmatter-guide-content">
            <div>
              <p className="frontmatter-intro">Frontmatter is the YAML metadata at the top of your Markdown file. SourceInk uses it to name your article, set its public URL, and decide how repository updates reach readers. Place it before the article body, between the opening and closing <code>---</code> lines.</p>
              <dl className="frontmatter-fields">
                <div><dt><code>title</code></dt><dd>The reader-facing article title.</dd></div>
                <div><dt><code>slug</code></dt><dd>The URL-friendly name used in the article&apos;s public address.</dd></div>
                <div><dt><code>publish_mode</code></dt><dd>Use <code>manual</code> to review and publish synced changes yourself. Use <code>auto</code> to publish each valid synced revision.</dd></div>
                <div><dt><code>description</code> and <code>tags</code></dt><dd>Optional details that help describe and organize the article.</dd></div>
              </dl>
            </div>
            <pre className="article-code"><code>{example}</code></pre>
          </div>
        </div>
      </div>
    </section>
  );
}
