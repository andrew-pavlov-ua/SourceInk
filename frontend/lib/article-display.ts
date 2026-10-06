type ArticleIdentity = {
  title: string;
  source_path: string;
};

export function articleTitle(article: ArticleIdentity) {
  return article.title || article.source_path.split("/").at(-1)?.replace(/\.md$/i, "") || "Untitled article";
}

export function frontmatterRecord(value: unknown): Record<string, unknown> {
  return typeof value === "object" && value !== null ? value as Record<string, unknown> : {};
}

export function stringArray(value: unknown) {
  return Array.isArray(value) ? value.filter((item): item is string => typeof item === "string") : [];
}

export function articleMetadata(frontmatter: unknown) {
  const values = frontmatterRecord(frontmatter);
  return {
    description: typeof values.description === "string" ? values.description : "",
    tags: stringArray(values.tags),
  };
}

export function articleReadingTime(markdown: string) {
  const words = markdown.trim().split(/\s+/).filter(Boolean).length;
  return `${Math.max(1, Math.ceil(words / 200))} min read`;
}
