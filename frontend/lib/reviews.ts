export type ReviewVerdict = "approve" | "request_changes";
export type ReviewReason = "incorrect" | "outdated" | "unclear";

export type ViewerReview = {
  verdict: ReviewVerdict;
  reason: ReviewReason | null;
};

export type ReviewSummary = {
  articleId: string;
  gitBlobSha: string;
  approveCount: number;
  requestChangesCount: number;
  requestReasons: Record<ReviewReason, number>;
  viewerReview: ViewerReview | null;
  authenticated: boolean;
};

export function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isCount(value: unknown): value is number {
  return typeof value === "number" && Number.isSafeInteger(value) && value >= 0;
}

function isReason(value: unknown): value is ReviewReason {
  return value === "incorrect" || value === "outdated" || value === "unclear";
}

function parseViewerReview(value: unknown): ViewerReview | null | undefined {
  if (value === null) return null;
  if (!isObject(value)) return undefined;
  if (value.verdict === "approve" && (value.reason === null || value.reason === undefined)) {
    return { verdict: "approve", reason: null };
  }
  if (value.verdict === "request_changes" && isReason(value.reason)) {
    return { verdict: "request_changes", reason: value.reason };
  }
  return undefined;
}

export function parseReviewSummary(value: unknown): ReviewSummary | null {
  if (!isObject(value) || !isObject(value.request_reasons)) return null;
  const viewerReview = parseViewerReview(value.viewer_review);
  const requestReasons = value.request_reasons;
  if (
    typeof value.article_id !== "string"
    || typeof value.git_blob_sha !== "string"
    || !isCount(value.approve_count)
    || !isCount(value.request_changes_count)
    || !isCount(requestReasons.incorrect)
    || !isCount(requestReasons.outdated)
    || !isCount(requestReasons.unclear)
    || viewerReview === undefined
    || typeof value.authenticated !== "boolean"
  ) {
    return null;
  }
  return {
    articleId: value.article_id,
    gitBlobSha: value.git_blob_sha,
    approveCount: value.approve_count,
    requestChangesCount: value.request_changes_count,
    requestReasons: {
      incorrect: requestReasons.incorrect,
      outdated: requestReasons.outdated,
      unclear: requestReasons.unclear,
    },
    viewerReview,
    authenticated: value.authenticated,
  };
}

export function reviewErrorMessage(value: unknown, fallback: string) {
  return isObject(value) && typeof value.error === "string" ? value.error : fallback;
}
