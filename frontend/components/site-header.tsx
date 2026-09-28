import Link from "next/link";
import { cookies } from "next/headers";
import { ThemeToggle } from "@/components/theme-toggle";
import { getCurrentUser, SESSION_COOKIE_NAME, type User } from "@/lib/backend";

async function getHeaderUser(): Promise<User | null> {
  try {
    const cookieStore = await cookies();
    if (!cookieStore.has(SESSION_COOKIE_NAME)) return null;
    return await getCurrentUser(cookieStore.toString());
  } catch {
    return null;
  }
}

export async function SiteHeader() {
  const user = await getHeaderUser();

  return (
    <header className="site-header">
      <div className="site-brand-group">
        <Link className="wordmark" href="/" aria-label="SourceInk home">
          <span className="wordmark-mark" aria-hidden="true">S</span>
          <span>SourceInk</span>
        </Link>
        <nav className="site-product-nav" aria-label="Product navigation">
          <Link href="/product">Product</Link>
          <Link href="/docs">Docs</Link>
          <Link href="/pricing">Pricing</Link>
        </nav>
      </div>
      <div className="site-nav">
        <ThemeToggle />
        {user ? (
          <div className="site-session">
            <Link className="site-user" href="/dashboard" aria-label={`Open dashboard for ${user.username}`}>
              <span className="site-user-avatar" aria-hidden="true">{user.username.charAt(0).toUpperCase()}</span>
              <span className="site-username">@{user.username}</span>
            </Link>
            <Link className="button button-small" href="/dashboard">Dashboard</Link>
          </div>
        ) : (
          <>
            <Link className="text-link" href="/login">Log in</Link>
            <Link className="button button-small" href="/register">Create account</Link>
          </>
        )}
      </div>
    </header>
  );
}
