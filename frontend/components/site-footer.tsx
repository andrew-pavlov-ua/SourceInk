import Link from "next/link";

export function SiteFooter() {
  return (
    <footer className="site-footer landing-frame">
      <span>© {new Date().getFullYear()} SourceInk</span>
      <nav className="site-footer-links" aria-label="Legal">
        <Link href="/terms">Terms &amp; Conditions</Link>
        <Link href="/license">License</Link>
      </nav>
    </footer>
  );
}
