import type { Metadata } from "next";
import { AuthForm } from "@/components/auth-form";
import { AuthShell } from "@/components/auth-shell";

export const metadata: Metadata = { title: "Log in" };

export default function LoginPage() {
  return (
    <AuthShell title="Log in" description="Use GitHub or your SourceInk email and password.">
      <AuthForm mode="login" />
    </AuthShell>
  );
}
