import type { ReactNode } from "react";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { AppShell } from "@/components/app-shell";
import { getCurrentUser, SESSION_COOKIE_NAME } from "@/lib/backend";

export const dynamic = "force-dynamic";

export default async function DashboardLayout({ children }: { children: ReactNode }) {
  const cookieStore = await cookies();
  if (!cookieStore.has(SESSION_COOKIE_NAME)) redirect("/login");

  const user = await getCurrentUser(cookieStore.toString());
  if (!user) redirect("/login");

  return (
    <AppShell repository="github.com/example/field-notes" username={user.username}>
      {children}
    </AppShell>
  );
}
