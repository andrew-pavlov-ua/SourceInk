const commits = [
  { sha: "bc832f1", message: "Add EXPLAIN ANALYZE output", when: "2 hours ago", tag: "draft" },
  { sha: "9e01a4d", message: "Fix the composite index example", when: "yesterday" },
  { sha: "a17c29f", message: "Explain partial indexes", when: "Mar 4", tag: "published" },
  { sha: "3f2b8c0", message: "First draft", when: "Mar 1" },
] as const;

export function CommitHistory() {
  return (
    <figure className="commit-history">
      <figcaption>
        <code>articles/postgres-indexes.md</code>
        <span>4 commits</span>
      </figcaption>
      <ol>
        {commits.map((commit) => (
          <li className={"tag" in commit ? `commit commit-${commit.tag}` : "commit"} key={commit.sha}>
            <span className="commit-node" aria-hidden="true" />
            <div className="commit-text">
              <strong>{commit.message}</strong>
              <span><code>{commit.sha}</code> {commit.when}</span>
            </div>
            {"tag" in commit && (
              <span className="commit-tag">{commit.tag === "draft" ? "Latest push, draft" : "What readers see"}</span>
            )}
          </li>
        ))}
      </ol>
    </figure>
  );
}
