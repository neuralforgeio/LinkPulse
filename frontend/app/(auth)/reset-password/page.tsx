"use client";

import { useEffect, useState, type FormEvent } from "react";
import Link from "next/link";
import { motion } from "framer-motion";
import { CheckCircle2, Eye, EyeOff } from "lucide-react";
import { api, ApiError } from "@/lib/api/client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { FadeIn } from "@/components/motion/fade";

export default function ResetPasswordPage() {
  const [token, setToken] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [show, setShow] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [done, setDone] = useState(false);

  // Reset links arrive as /reset-password?token=lp_reset_xxx — read it
  // from the URL on mount (client-side only, no Suspense needed).
  useEffect(() => {
    const t = new URLSearchParams(window.location.search).get("token");
    if (t) setToken(t);
  }, []);

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!token.trim()) {
      setError("Please enter the reset token.");
      return;
    }
    if (
      password.length < 8 ||
      !/[A-Za-z]/.test(password) ||
      !/[0-9]/.test(password)
    ) {
      setError(
        "Password must be at least 8 characters with letters and numbers.",
      );
      return;
    }
    if (password !== confirm) {
      setError("Passwords do not match.");
      return;
    }

    setLoading(true);
    setError(null);
    try {
      await api("/api/v1/auth/password/reset-confirm", {
        method: "POST",
        body: JSON.stringify({ token: token.trim(), password }),
      });
      setDone(true);
    } catch (err) {
      setError(
        err instanceof ApiError
          ? err.message
          : "Cannot reach the server. Is the backend running?",
      );
    } finally {
      setLoading(false);
    }
  }

  if (done) {
    return (
      <motion.div
        initial={{ opacity: 0, scale: 0.92, y: 24 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        transition={{ type: "spring", stiffness: 260, damping: 22 }}
        className="text-center"
      >
        <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-blue-600/10 text-blue-600 dark:bg-blue-500/15 dark:text-blue-400">
          <CheckCircle2 className="h-8 w-8" />
        </div>
        <h1 className="mt-5 text-2xl font-bold tracking-tight text-zinc-900 dark:text-white">
          Password updated
        </h1>
        <p className="mt-2 text-sm text-zinc-500 dark:text-zinc-400">
          All other sessions have been signed out for security.
        </p>
        <Link
          href="/login"
          className="mt-6 flex w-full items-center justify-center rounded-lg bg-zinc-900 px-5 py-3 text-sm font-semibold text-white shadow-sm transition hover:bg-zinc-800 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
        >
          Sign in with your new password
        </Link>
      </motion.div>
    );
  }

  return (
    <div>
      <FadeIn>
        <h1 className="text-2xl font-bold tracking-tight text-zinc-900 dark:text-white">
          Set a new password
        </h1>
        <p className="mt-1.5 text-sm text-zinc-500 dark:text-zinc-400">
          Paste your reset token and choose a new password.
        </p>
      </FadeIn>

      <form className="mt-8 space-y-5" onSubmit={onSubmit} noValidate>
        <FadeIn delay={0.08}>
          <Input
            label="Reset token"
            placeholder="lp_reset_..."
            value={token}
            onChange={(e) => setToken(e.target.value)}
          />
        </FadeIn>

        <FadeIn delay={0.16}>
          <Input
            label="New password"
            type={show ? "text" : "password"}
            autoComplete="new-password"
            placeholder="&bull;&bull;&bull;&bull;&bull;&bull;&bull;&bull;"
            hint="At least 8 characters, with letters and numbers."
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            trailing={
              <button
                type="button"
                onClick={() => setShow((v) => !v)}
                aria-label={show ? "Hide password" : "Show password"}
                className="text-zinc-400 transition hover:text-zinc-600 dark:hover:text-zinc-300"
              >
                {show ? (
                  <EyeOff className="h-4 w-4" />
                ) : (
                  <Eye className="h-4 w-4" />
                )}
              </button>
            }
          />
        </FadeIn>

        <FadeIn delay={0.24}>
          <Input
            label="Confirm new password"
            type={show ? "text" : "password"}
            autoComplete="new-password"
            placeholder="&bull;&bull;&bull;&bull;&bull;&bull;&bull;&bull;"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
          />
        </FadeIn>

        {error && (
          <p className="rounded-lg bg-rose-50 px-4 py-3 text-sm text-rose-600 dark:bg-rose-500/10 dark:text-rose-400">
            {error}
          </p>
        )}

        <FadeIn delay={0.32}>
          <Button type="submit" loading={loading} className="w-full py-3">
            Update password
          </Button>
        </FadeIn>
      </form>
    </div>
  );
}
