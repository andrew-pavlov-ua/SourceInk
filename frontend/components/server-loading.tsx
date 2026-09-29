type ServerLoadingProps = {
  label: string;
  rows?: number;
  variant?: "article" | "repository" | "settings";
};

export function ServerLoading({ label, rows = 3, variant = "article" }: ServerLoadingProps) {
  return (
    <div className={`server-loading server-loading-${variant}`} role="status" aria-label={label}>
      <span className="sr-only">{label}</span>
      {Array.from({ length: rows }, (_, index) => (
        <div className="server-loading-row" key={index} aria-hidden="true">
          <span className="server-loading-mark" />
          <div className="server-loading-copy">
            <span className="server-loading-line server-loading-title" />
            <span className="server-loading-line server-loading-detail" />
          </div>
          <span className="server-loading-meta" />
          <span className="server-loading-state" />
        </div>
      ))}
    </div>
  );
}
