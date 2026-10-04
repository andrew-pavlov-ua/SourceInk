import Link from "next/link";
import { SOURCE_CODE_URL, SUPPORT_URL } from "@/lib/legal";

export function SiteFooter() {
  return (
    <footer className="site-footer landing-frame">
      <span>© {new Date().getFullYear()} SourceInk</span>
      <nav className="site-footer-links" aria-label="Footer links">
        <a className="footer-support-link" href={SUPPORT_URL} target="_blank" rel="noopener noreferrer">
          Support SourceInk
        </a>
        <Link href="/terms">Terms &amp; Conditions</Link>
        <Link href="/license">License</Link>
        <a href={SOURCE_CODE_URL}>Source code</a>
      </nav>
    </footer>
  );
}
