import Link from "next/link";
import { cookies } from "next/headers";
import { Logo } from "@/components/logo";
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
        <Logo />
        <nav className="site-product-nav" aria-label="Product navigation">
          <Link href="/product">Product</Link>
          <Link href="/docs">Docs</Link>
          {/*<Link href="/pricing">Pricing</Link>*/}
        </nav>
      </div>
      <div className="site-nav">
        <ThemeToggle />
        {user ? (
          <Link className="button button-small" href="/dashboard">Dashboard</Link>
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
