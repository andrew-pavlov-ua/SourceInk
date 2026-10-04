import Link from "next/link";

export function LogoMark({ size = 24 }: { size?: number }) {
  return (
    <svg
      className="logo-mark"
      width={size}
      height={size}
      viewBox="0 0 24 24"
      aria-hidden="true"
    >
      <path
        d="M12 1.5C12 1.5 4.5 9.6 4.5 15a7.5 7.5 0 0 0 15 0C19.5 9.6 12 1.5 12 1.5Z"
        fill="var(--accent)"
      />
      <path
        d="M6.6 15h3M14.4 15h3"
        stroke="var(--canvas)"
        strokeWidth="1.8"
        strokeLinecap="round"
      />
      <circle
        cx="12"
        cy="15"
        r="2.3"
        fill="none"
        stroke="var(--canvas)"
        strokeWidth="1.8"
      />
    </svg>
  );
}

export function Logo() {
  return (
    <Link className="logo" href="/" aria-label="SourceInk home">
      <LogoMark />
      <span>SourceInk</span>
    </Link>
  );
}
