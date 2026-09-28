export type User = {
  id: string;
  email: string;
  username: string;
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

export type ArticlePublishMode = "manual" | "automatic";

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
  repository_id?: string;
  source_path: string;
  slug: string;
  publish_mode: ArticlePublishMode;
  source_state: "available" | "missing" | "access_lost";
  git_blob_sha: string;
  markdown: string;
  title: string;
  frontmatter?: unknown;
  rendered_html: string;
  published_at: string;
  created_at: string;
  updated_at: string;
};

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
  return body;
}

function isUserArticles(value: unknown): value is UserArticles {
  if (typeof value !== "object" || value === null) return false;
  const articles = value as Partial<UserArticles>;
  return typeof articles.user_id === "string"
    && Array.isArray(articles.unpublished_articles)
    && Array.isArray(articles.published_articles);
}
