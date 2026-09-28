import type { Metadata } from "next";
import { AuthForm } from "@/components/auth-form";
import { AuthShell } from "@/components/auth-shell";

export const metadata: Metadata = { title: "Create account" };

export default function RegisterPage() {
  return (
    <AuthShell title="Create an account" description="Use GitHub, or create an account with an email and password.">
      <AuthForm mode="register" />
    </AuthShell>
  );
}
