import { isObject, parseReviewSummary, type ReviewSummary } from "@/lib/reviews";

export type User = {
  id: string;
  email: string;
  username: string;
  github_user_id?: number;
  github_login?: string;
  github_avatar_url?: string;
  role: "user" | "admin";
  created_at: string;
};

export type Repository = {
  github_id: number;
  owner: string;
  name: string;
  full_name: string;
  default_branch: string;
  private: boolean;
  archived: boolean;
};

export type ArticlePublishMode = "manual" | "auto";

type UnpublishedArticleFrontmatter = {
  title?: string;
  slug?: string;
  description?: string;
  tags?: string[] | null;
  publish_mode?: ArticlePublishMode | "";
};

export type UnpublishedArticle = {
  id: string;
  repository_id: string;
  source_path: string;
  git_blob_sha: string;
  title: string;
  slug: string;
  description: string;
  tags: string[];
  publish_mode: ArticlePublishMode | "";
  content: string;
  validation_error?: string;
  present: boolean;
  discovered_at: string;
  updated_at: string;
};

export type PublishedArticle = {
  id: string;
  unpublished_article_id?: string;
  owner_id: string;
  author_username: string;
  repository_id?: string;
  source_path: string;
  slug: string;
  publish_mode: ArticlePublishMode;
  source_state: "available" | "missing" | "access_lost";
  git_blob_sha: string;
  markdown: string;
  title: string;
  frontmatter?: unknown;
  view_count: number;
  published_at: string;
  created_at: string;
  updated_at: string;
};

export type ReviewedPublishedArticle = PublishedArticle & {
  reviews: ReviewSummary;
};

export type PublishedArticleOrder = "newest" | "oldest" | "rating-desc" | "rating-asc";

export type UserArticles = {
  user_id: string;
  unpublished_articles: UnpublishedArticle[];
  published_articles: PublishedArticle[];
};

export const SESSION_COOKIE_NAME = "sourceink_session";

export async function getCurrentUser(cookieHeader: string): Promise<User | null> {
  const baseURL = process.env.API_INTERNAL_URL ?? "http://127.0.0.1:8080";
  const response = await fetch(`${baseURL}/api/auth/me`, {
    headers: { cookie: cookieHeader },
    cache: "no-store",
  });
  if (response.status === 401) return null;
  if (!response.ok) throw new Error(`account lookup failed with status ${response.status}`);
  const body = (await response.json()) as { user: User };
  return body.user;
}

export async function getRepositories(cookieHeader: string): Promise<Repository[]> {
  const baseURL = process.env.API_INTERNAL_URL ?? "http://127.0.0.1:8080";
  const response = await fetch(`${baseURL}/api/repos`, {
    headers: { cookie: cookieHeader },
    cache: "no-store",
  });
  if (!response.ok) throw new Error(`repository lookup failed with status ${response.status}`);
  const body: unknown = await response.json();
  if (body === null) return [];
  if (!Array.isArray(body)) throw new Error("repository lookup returned an invalid response");
  return body as Repository[];
}

export async function getArticles(cookieHeader: string): Promise<UserArticles> {
  const baseURL = process.env.API_INTERNAL_URL ?? "http://127.0.0.1:8080";
  const response = await fetch(`${baseURL}/api/articles`, {
    headers: { cookie: cookieHeader },
    cache: "no-store",
  });
  if (!response.ok) throw new Error(`article lookup failed with status ${response.status}`);

  const body: unknown = await response.json();
  if (!isUserArticles(body)) throw new Error("article lookup returned an invalid response");
  return {
    ...body,
    unpublished_articles: body.unpublished_articles.map(normalizeUnpublishedArticle),
  };
}

export async function getPublishedArticles(cookieHeader = "", order: PublishedArticleOrder = "newest"): Promise<ReviewedPublishedArticle[]> {
  const baseURL = process.env.API_INTERNAL_URL ?? "http://127.0.0.1:8080";
  const response = await fetch(`${baseURL}/api/published-articles?sort=${encodeURIComponent(order)}`, {
    headers: cookieHeader ? { cookie: cookieHeader } : undefined,
    cache: "no-store",
  });
  if (!response.ok) throw new Error(`published article lookup failed with status ${response.status}`);

  const body: unknown = await response.json();
  if (!Array.isArray(body)) throw new Error("published article lookup returned an invalid response");
  const articles = body.map(parsePublishedArticle);
  if (articles.some((article) => article === null)) {
    throw new Error("published article lookup returned an invalid response");
  }
  return articles as ReviewedPublishedArticle[];
}

export async function getPublishedArticle(slug: string, cookieHeader = ""): Promise<ReviewedPublishedArticle | null> {
  const baseURL = process.env.API_INTERNAL_URL ?? "http://127.0.0.1:8080";
  const response = await fetch(`${baseURL}/api/published-articles/${encodeURIComponent(slug)}`, {
    headers: cookieHeader ? { cookie: cookieHeader } : undefined,
    cache: "no-store",
  });
  if (response.status === 404) return null;
  if (!response.ok) throw new Error(`published article lookup failed with status ${response.status}`);

  const article = parsePublishedArticle(await response.json());
  if (!article) throw new Error("published article lookup returned an invalid response");
  return article;
}

function parsePublishedArticle(value: unknown): ReviewedPublishedArticle | null {
  if (!isObject(value)) return null;
  const reviews = parseReviewSummary(value.reviews);
  if (
    typeof value.id !== "string"
    || typeof value.owner_id !== "string"
    || typeof value.author_username !== "string"
    || typeof value.source_path !== "string"
    || typeof value.slug !== "string"
    || (value.publish_mode !== "manual" && value.publish_mode !== "auto")
    || (value.source_state !== "available" && value.source_state !== "missing" && value.source_state !== "access_lost")
    || typeof value.git_blob_sha !== "string"
    || typeof value.markdown !== "string"
    || typeof value.title !== "string"
    || typeof value.view_count !== "number"
    || typeof value.published_at !== "string"
    || typeof value.created_at !== "string"
    || typeof value.updated_at !== "string"
    || !reviews
    || reviews.articleId !== value.id
    || reviews.gitBlobSha !== value.git_blob_sha
  ) {
    return null;
  }
  return { ...(value as PublishedArticle), reviews };
}

function normalizeUnpublishedArticle(article: UnpublishedArticle): UnpublishedArticle {
  const responseArticle = article as UnpublishedArticle & {
    frontmatter?: UnpublishedArticleFrontmatter;
  };
  const frontmatter = responseArticle.frontmatter;

  return {
    ...article,
    title: frontmatter?.title ?? article.title ?? "",
    slug: frontmatter?.slug ?? article.slug ?? "",
    description: frontmatter?.description ?? article.description ?? "",
    tags: frontmatter?.tags ?? article.tags ?? [],
    publish_mode: frontmatter?.publish_mode ?? article.publish_mode ?? "",
  };
}

function isUserArticles(value: unknown): value is UserArticles {
  if (typeof value !== "object" || value === null) return false;
  const articles = value as Partial<UserArticles>;
  return typeof articles.user_id === "string"
    && Array.isArray(articles.unpublished_articles)
    && Array.isArray(articles.published_articles);
}
