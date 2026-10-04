"use client";

import Link from "next/link";
import { useId, useState } from "react";
import {
  parseReviewSummary,
  reviewErrorMessage,
  type ReviewReason,
  type ReviewSummary,
  type ReviewVerdict,
} from "@/lib/reviews";

type ArticleReviewProps = {
  articleId: string;
  gitBlobSha: string;
  initialSummary: ReviewSummary;
};

const reasons: Array<{ value: ReviewReason; label: string; description: string }> = [
  { value: "incorrect", label: "Incorrect", description: "A claim or technical detail is wrong." },
  { value: "outdated", label: "Outdated", description: "The guidance no longer matches current behavior." },
  { value: "unclear", label: "Unclear", description: "An important part is difficult to understand." },
];

export function ArticleReview({ articleId, gitBlobSha, initialSummary }: ArticleReviewProps) {
  const statusId = useId();
  const reasonHeadingId = useId();
  const [summary, setSummary] = useState(initialSummary);
  const [reasonOpen, setReasonOpen] = useState(false);
  const [pending, setPending] = useState<ReviewVerdict | "delete" | null>(null);
  const [message, setMessage] = useState("");
  const [stale, setStale] = useState(false);

  const endpoint = `/api/published-articles/${encodeURIComponent(articleId)}`;

  async function saveReview(verdict: ReviewVerdict, reason: ReviewReason | null) {
    if (!summary?.authenticated) {
      setMessage("Sign in to review this revision.");
      return;
    }
    setPending(verdict);
    setMessage("");
    setStale(false);
    try {
      const response = await fetch(`${endpoint}/review`, {
        method: "PUT",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ git_blob_sha: gitBlobSha, verdict, reason }),
      });
      const body: unknown = await response.json().catch(() => null);
      if (response.status === 401) {
        setSummary((current) => current ? { ...current, authenticated: false, viewerReview: null } : current);
        setMessage("Your session ended. Sign in to review this revision.");
        return;
      }
      if (response.status === 409) {
        setStale(true);
        setMessage(reviewErrorMessage(body, "A newer revision is available. Refresh before reviewing."));
        return;
      }
      const parsed = parseReviewSummary(body);
      if (!response.ok || !parsed || parsed.articleId !== articleId || parsed.gitBlobSha !== gitBlobSha) {
        setMessage(reviewErrorMessage(body, "SourceInk couldn't save your review. Try again."));
        return;
      }
      setSummary(parsed);
      setReasonOpen(false);
      setMessage(verdict === "approve" ? "You approved this revision." : "Your change request was saved.");
    } catch {
      setMessage("SourceInk couldn't reach the review service. Check your connection and try again.");
    } finally {
      setPending(null);
    }
  }

  async function removeReview() {
    if (!summary?.authenticated || !summary.viewerReview) return;
    setPending("delete");
    setMessage("");
    setStale(false);
    try {
      const response = await fetch(`${endpoint}/review?git_blob_sha=${encodeURIComponent(gitBlobSha)}`, {
        method: "DELETE",
        credentials: "same-origin",
      });
      const body: unknown = await response.json().catch(() => null);
      if (response.status === 409) {
        setStale(true);
        setMessage(reviewErrorMessage(body, "A newer revision is available. Refresh before changing your review."));
        return;
      }
      const parsed = parseReviewSummary(body);
      if (!response.ok || !parsed || parsed.articleId !== articleId || parsed.gitBlobSha !== gitBlobSha) {
        setMessage(reviewErrorMessage(body, "SourceInk couldn't remove your review. Try again."));
        return;
      }
      setSummary(parsed);
      setReasonOpen(false);
      setMessage("Your review was removed.");
    } catch {
      setMessage("SourceInk couldn't reach the review service. Check your connection and try again.");
    } finally {
      setPending(null);
    }
  }

  const approved = summary.viewerReview?.verdict === "approve";
  const requestedChanges = summary.viewerReview?.verdict === "request_changes";
  const signInHref = "/login";

  return (
    <section className="article-review" aria-labelledby="article-review-heading">
      <div className="article-review-heading">
        <div>
          <h2 id="article-review-heading">Review this revision</h2>
          <p>Leave a Git-style signal for the author. Feedback resets when a new revision is published.</p>
        </div>
        <code>{gitBlobSha.slice(0, 8)}</code>
      </div>

      <div className="article-review-actions">
        <button
          type="button"
          className={`review-action review-action-approve${approved ? " review-action-selected" : ""}`}
          aria-pressed={approved}
          aria-describedby={statusId}
          disabled={pending !== null}
          onClick={() => void saveReview("approve", null)}
        >
          <svg viewBox="0 0 20 20" aria-hidden="true"><path d="m4 10 4 4 8-9" /></svg>
          <span>Approve</span>
          <strong>{summary.approveCount.toLocaleString()}</strong>
        </button>
        <button
          type="button"
          className={`review-action review-action-changes${requestedChanges ? " review-action-selected" : ""}`}
          aria-pressed={requestedChanges}
          aria-expanded={reasonOpen}
          aria-controls="review-reasons"
          aria-describedby={statusId}
          disabled={pending !== null}
          onClick={() => {
            if (!summary.authenticated) {
              setMessage("Sign in to review this revision.");
              return;
            }
            setReasonOpen((open) => !open);
            setMessage("");
          }}
        >
          <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M10 3v8M10 15v.01" /><path d="M9 2.8 2.7 14a2 2 0 0 0 1.7 3h11.2a2 2 0 0 0 1.7-3L11 2.8a1.15 1.15 0 0 0-2 0Z" /></svg>
          <span>Request changes</span>
          <strong>{summary.requestChangesCount.toLocaleString()}</strong>
        </button>
      </div>

      {reasonOpen && (
        <div className="review-reasons" id="review-reasons" aria-labelledby={reasonHeadingId}>
          <p id={reasonHeadingId}>What needs attention?</p>
          <div className="review-reason-list">
            {reasons.map((reason) => {
              const selected = summary.viewerReview?.reason === reason.value;
              return (
                <button
                  type="button"
                  key={reason.value}
                  className={selected ? "review-reason review-reason-selected" : "review-reason"}
                  aria-pressed={selected}
                  disabled={pending !== null}
                  onClick={() => void saveReview("request_changes", reason.value)}
                >
                  <span><strong>{reason.label}</strong><small>{reason.description}</small></span>
                  <b>{summary.requestReasons[reason.value].toLocaleString()}</b>
                </button>
              );
            })}
          </div>
        </div>
      )}

      <div className="article-review-status" id={statusId} aria-live="polite">
        {pending && <span>{pending === "delete" ? "Removing your review…" : "Saving your review…"}</span>}
        {!pending && message && <span>{message}</span>}
        {!pending && message && !summary.authenticated && <Link href={signInHref}>Sign in</Link>}
        {!pending && stale && <button type="button" className="review-text-button" onClick={() => window.location.reload()}>Refresh article</button>}
        {!pending && summary.viewerReview && !stale && (
          <button type="button" className="review-text-button" onClick={() => void removeReview()}>Remove my review</button>
        )}
      </div>
    </section>
  );
}
