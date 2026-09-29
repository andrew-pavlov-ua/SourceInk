"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

type PublishArticleButtonProps = {
  draftId: string;
  hasPublishedCopy: boolean;
  disabledReason?: string;
};

type PublishResponse = {
  id?: unknown;
  error?: unknown;
};

export function PublishArticleButton({ draftId, hasPublishedCopy, disabledReason }: PublishArticleButtonProps) {
  const router = useRouter();
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState("");
  const [failed, setFailed] = useState(false);
  const messageId = `publish-message-${draftId}`;

  async function publish() {
    setPending(true);
    setFailed(false);
    setMessage("");

    try {
      const response = await fetch(`/api/articles/${encodeURIComponent(draftId)}/publish`, {
        method: "POST",
        headers: { accept: "application/json" },
      });
      const body = (await response.json().catch(() => ({}))) as PublishResponse;

      if (!response.ok) {
        setFailed(true);
        setMessage(typeof body.error === "string" ? body.error : "SourceInk couldn't publish this article.");
        return;
      }
      if (typeof body.id !== "string") {
        setFailed(true);
        setMessage("SourceInk published the article but returned an invalid response.");
        return;
      }

      setMessage("Published. Opening the published snapshot…");
      router.replace(`/dashboard/articles/${encodeURIComponent(body.id)}`);
      router.refresh();
    } catch {
      setFailed(true);
      setMessage("SourceInk couldn't reach the publishing service. Try again.");
    } finally {
      setPending(false);
    }
  }

  const label = pending
    ? "Publishing…"
    : hasPublishedCopy
      ? "Publish latest version"
      : "Publish article";

  return (
    <div className="article-publish-action">
      <button
        className="button"
        type="button"
        onClick={publish}
        disabled={pending || Boolean(disabledReason)}
        aria-busy={pending}
        aria-describedby={messageId}
      >
        {label}
      </button>
      <p id={messageId} className={failed ? "article-publish-message article-publish-message-error" : "article-publish-message"} aria-live="polite">
        {message || disabledReason || ""}
      </p>
    </div>
  );
}
