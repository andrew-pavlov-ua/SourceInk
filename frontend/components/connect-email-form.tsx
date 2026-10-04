"use client";

import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";

export function ConnectEmailForm() {
  const router = useRouter();
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setError("");
    setSuccess("");

    const form = new FormData(event.currentTarget);
    const payload = {
      email: String(form.get("email") ?? ""),
      password: String(form.get("password") ?? ""),
    };

    try {
      const response = await fetch("/api/auth/email", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "same-origin",
        body: JSON.stringify(payload),
      });
      const body = (await response.json().catch(() => ({}))) as { error?: string };
      if (!response.ok) {
        setError(body.error ?? "Could not add email sign-in. Try again.");
        return;
      }

      setSuccess("Email sign-in connected. You can now use either method.");
      router.refresh();
    } catch {
      setError("Could not reach SourceInk. Try again.");
    } finally {
      setPending(false);
    }
  }

  return (
    <form className="settings-fields connect-email-form" onSubmit={submit}>
      <label htmlFor="connect-email">
        Email
        <input id="connect-email" name="email" type="email" autoComplete="email" maxLength={254} required disabled={pending} />
      </label>
      <label htmlFor="connect-password">
        Password
        <input id="connect-password" name="password" type="password" autoComplete="new-password" minLength={8} maxLength={256} required disabled={pending} />
        <span className="field-hint">Use at least 8 characters.</span>
      </label>
      <div className="connect-email-status" aria-live="polite">
        {error && <p className="form-error">{error}</p>}
        {success && <p className="form-success">{success}</p>}
      </div>
      <button className="button button-small connect-email-submit" type="submit" disabled={pending}>
        {pending ? "Connecting…" : "Add email sign-in"}
      </button>
    </form>
  );
}
