import { structuredPatch, type StructuredPatchHunk } from "diff";

type ArticleDiffProps = {
  publishedSource: string;
  draftSource: string;
  sourcePath: string;
  publishedBlobSHA: string;
  draftBlobSHA: string;
};

type DiffLine = {
  content: string;
  kind: "added" | "context" | "deleted" | "note";
  oldLine?: number;
  newLine?: number;
};

function linesForHunk(hunk: StructuredPatchHunk): DiffLine[] {
  let oldLine = hunk.oldStart;
  let newLine = hunk.newStart;

  return hunk.lines.map((line) => {
    const marker = line.at(0);
    const content = line.slice(1);

    if (marker === "+") {
      return { content, kind: "added", newLine: newLine++ };
    }
    if (marker === "-") {
      return { content, kind: "deleted", oldLine: oldLine++ };
    }
    if (marker === "\\") {
      return { content: line, kind: "note" };
    }

    return { content, kind: "context", oldLine: oldLine++, newLine: newLine++ };
  });
}

function range(start: number, count: number) {
  return count === 1 ? `${start}` : `${start},${count}`;
}

function hunkLabel(hunk: StructuredPatchHunk) {
  return `@@ -${range(hunk.oldStart, hunk.oldLines)} +${range(hunk.newStart, hunk.newLines)} @@`;
}

export function ArticleDiff({
  publishedSource,
  draftSource,
  sourcePath,
  publishedBlobSHA,
  draftBlobSHA,
}: ArticleDiffProps) {
  const patch = structuredPatch(
    `a/${sourcePath}`,
    `b/${sourcePath}`,
    publishedSource,
    draftSource,
    "published",
    "repository draft",
    { context: 3 },
  );
  const additions = patch.hunks.reduce(
    (total, hunk) => total + hunk.lines.filter((line) => line.startsWith("+")).length,
    0,
  );
  const deletions = patch.hunks.reduce(
    (total, hunk) => total + hunk.lines.filter((line) => line.startsWith("-")).length,
    0,
  );

  return (
    <div className="article-diff" role="region" aria-label={`Changes to ${sourcePath}`}>
      <div className="article-diff-toolbar">
        <div className="article-diff-file">
          <strong>{sourcePath}</strong>
          <span>
            <code>{publishedBlobSHA.slice(0, 8)}</code>
            <span> to </span>
            <code>{draftBlobSHA.slice(0, 8)}</code>
          </span>
        </div>
        <div className="article-diff-summary" aria-label={`${additions} additions and ${deletions} deletions`}>
          <span className="article-diff-additions">+{additions}</span>
          <span className="article-diff-deletions">−{deletions}</span>
        </div>
      </div>

      <div className="article-diff-scroll">
        {patch.hunks.length > 0 ? (
          <table className="article-diff-table">
            <caption>Line-by-line changes from the published article to the repository draft.</caption>
            <colgroup>
              <col className="article-diff-column-number" />
              <col className="article-diff-column-number" />
              <col className="article-diff-column-marker" />
              <col />
            </colgroup>
            <thead>
              <tr>
                <th scope="col">Published line</th>
                <th scope="col">Draft line</th>
                <th scope="col">Change</th>
                <th scope="col">Content</th>
              </tr>
            </thead>
            <tbody>
              {patch.hunks.map((hunk) => (
                <Hunk key={`${hunk.oldStart}-${hunk.newStart}`} hunk={hunk} />
              ))}
            </tbody>
          </table>
        ) : (
          <p className="article-diff-empty">SourceInk found no parsed content changes in this Git blob.</p>
        )}
      </div>
    </div>
  );
}

function Hunk({ hunk }: { hunk: StructuredPatchHunk }) {
  return (
    <>
      <tr className="article-diff-hunk">
        <td colSpan={4}><code>{hunkLabel(hunk)}</code></td>
      </tr>
      {linesForHunk(hunk).map((line, index) => (
        <tr className={`article-diff-line article-diff-line-${line.kind}`} key={`${line.kind}-${line.oldLine ?? ""}-${line.newLine ?? ""}-${index}`}>
          <td className="article-diff-line-number">{line.oldLine}</td>
          <td className="article-diff-line-number">{line.newLine}</td>
          <td className="article-diff-marker" aria-label={line.kind === "added" ? "Added" : line.kind === "deleted" ? "Deleted" : undefined}>
            {line.kind === "added" ? "+" : line.kind === "deleted" ? "-" : " "}
          </td>
          <td className="article-diff-content"><code>{line.content || " "}</code></td>
        </tr>
      ))}
    </>
  );
}
