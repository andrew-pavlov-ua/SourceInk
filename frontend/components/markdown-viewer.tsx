"use client";

import { useId, useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

type MarkdownViewerProps = {
  markdown: string;
};

export function MarkdownViewer({ markdown }: MarkdownViewerProps) {
  const [mode, setMode] = useState<"preview" | "code">("preview");
  const panelId = useId();
  const hasContent = markdown.trim().length > 0;

  return (
    <div className="markdown-viewer">
      <div className="markdown-viewer-toolbar" role="group" aria-label="Markdown view">
        <button
          type="button"
          aria-pressed={mode === "preview"}
          aria-controls={panelId}
          className={mode === "preview" ? "markdown-viewer-tab markdown-viewer-tab-active" : "markdown-viewer-tab"}
          onClick={() => setMode("preview")}
        >
          Preview
        </button>
        <button
          type="button"
          aria-pressed={mode === "code"}
          aria-controls={panelId}
          className={mode === "code" ? "markdown-viewer-tab markdown-viewer-tab-active" : "markdown-viewer-tab"}
          onClick={() => setMode("code")}
        >
          Code
        </button>
      </div>

      <div id={panelId} role="region" aria-label={mode === "preview" ? "Markdown preview" : "Markdown source"} className="markdown-viewer-panel">
        {!hasContent ? (
          <p className="markdown-viewer-empty">This Markdown file is empty.</p>
        ) : mode === "preview" ? (
          <div className="markdown-preview">
            <ReactMarkdown remarkPlugins={[remarkGfm]} skipHtml>{markdown}</ReactMarkdown>
          </div>
        ) : (
          <pre className="article-code markdown-source"><code>{markdown}</code></pre>
        )}
      </div>
    </div>
  );
}
