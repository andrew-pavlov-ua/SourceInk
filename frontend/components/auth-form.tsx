"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";

type AuthMode = "login" | "register";
type PendingAction = "email" | "github" | null;

export function AuthForm({ mode }: { mode: AuthMode }) {
  const router = useRouter();
  const [error, setError] = useState("");
  const [pendingAction, setPendingAction] = useState<PendingAction>(null);
  const register = mode === "register";

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setPendingAction("email");

    const form = new FormData(event.currentTarget);
    const payload = {
      email: String(form.get("email") ?? ""),
      password: String(form.get("password") ?? ""),
      ...(register ? { username: String(form.get("username") ?? "") } : {}),
    };

    try {
      const response = await fetch(`/api/auth/${mode}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "same-origin",
        body: JSON.stringify(payload),
      });
      const body = (await response.json().catch(() => ({}))) as { error?: string };
      if (!response.ok) {
        setError(body.error ?? "Check the fields and try again.");
        return;
      }
      router.replace("/dashboard");
      router.refresh();
    } catch {
      setError("Could not reach SourceInk. Try again.");
    } finally {
      setPendingAction(null);
    }
  }

  function beginGitHubLogin() {
    setError("");
    setPendingAction("github");
    // OAuth must leave the Next.js app instead of requesting an RSC payload.
    // eslint-disable-next-line @next/next/no-location-assign-relative-destination
    window.location.assign("/api/auth/github");
  }

  return (
    <form className="auth-form" onSubmit={submit}>
      <button className="button auth-submit auth-github" type="button" onClick={beginGitHubLogin} disabled={pendingAction !== null}>
        {pendingAction === "github" ? "Connecting to GitHub…" : "Continue with GitHub"}
      </button>
      <div className="auth-divider" role="separator">or use email</div>
      {register && (
        <div className="field-group">
          <label htmlFor="username">Username</label>
          <input id="username" name="username" type="text" autoComplete="username" minLength={3} maxLength={32} pattern="[A-Za-z0-9][A-Za-z0-9_.-]{2,31}" required />
          <span className="field-hint">Your public handle. Letters, numbers, dots, dashes, or underscores.</span>
        </div>
      )}
      <div className="field-group">
        <label htmlFor="email">Email</label>
        <input id="email" name="email" type="email" autoComplete="email" maxLength={254} required />
      </div>
      <div className="field-group">
        <label htmlFor="password">Password</label>
        <input id="password" name="password" type="password" autoComplete={register ? "new-password" : "current-password"} minLength={8} maxLength={256} required />
        {register && <span className="field-hint">Use at least 8 characters.</span>}
      </div>
      <div className="form-status" aria-live="polite">
        {error && <p className="form-error">{error}</p>}
      </div>
      <button className="button auth-submit" type="submit" disabled={pendingAction !== null}>
        {pendingAction === "email" ? "Please wait…" : register ? "Create account" : "Log in"}
      </button>
      <p className="auth-switch">
        {register ? "Already have an account?" : "Need an account?"}{" "}
        <Link href={register ? "/login" : "/register"}>{register ? "Log in" : "Create one"}</Link>
      </p>
    </form>
  );
}
