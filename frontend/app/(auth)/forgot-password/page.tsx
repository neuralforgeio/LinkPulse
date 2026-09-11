"use client";

import { useState, type FormEvent } from "react";
import Link from "next/link";
import { motion } from "framer-motion";
import { CheckCircle2 } from "lucide-react";
import { api, ApiError } from "@/lib/api/client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { FadeIn } from "@/components/motion/fade";

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState("");
  const [loading, setLoading] = useState(false);
  const [sent, setSent] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!email.trim()) {
      setError("Please enter your email.");
      return;
    }

    setLoading(true);
    setError(null);
    try {
      await api("/api/v1/auth/password/reset-request", {
        method: "POST",
        body: JSON.stringify({ email: email.trim() }),
      });
      setSent(true);
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

  if (sent) {
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
          Check your email
        </h1>
        <p className="mt-2 text-sm text-zinc-500 dark:text-zinc-400">
          If an account exists for{" "}
          <span className="font-medium text-zinc-700 dark:text-zinc-200">
            {email}
          </span>
          , a reset link has been sent.
        </p>
        <p className="mt-4 rounded-lg bg-zinc-100 px-4 py-3 text-xs text-zinc-500 dark:bg-zinc-900 dark:text-zinc-400">
          Development mode: the reset link is written to the backend server log
          — no email is actually sent.
        </p>
        <Link
          href="/login"
          className="mt-6 inline-block text-sm font-medium text-blue-600 hover:text-blue-500 dark:text-blue-400"
        >
          Back to sign in
        </Link>
      </motion.div>
    );
  }

  return (
    <div>
      <FadeIn>
        <h1 className="text-2xl font-bold tracking-tight text-zinc-900 dark:text-white">
          Forgot your password?
        </h1>
        <p className="mt-1.5 text-sm text-zinc-500 dark:text-zinc-400">
          Enter your email and we&apos;ll send a reset link.
        </p>
      </FadeIn>

      <form className="mt-8 space-y-5" onSubmit={onSubmit} noValidate>
        <FadeIn delay={0.08}>
          <Input
            label="Email"
            type="email"
            autoComplete="email"
            placeholder="you@company.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
        </FadeIn>

        {error && (
          <p className="rounded-lg bg-rose-50 px-4 py-3 text-sm text-rose-600 dark:bg-rose-500/10 dark:text-rose-400">
            {error}
          </p>
        )}

        <FadeIn delay={0.16}>
          <Button type="submit" loading={loading} className="w-full py-3">
            Send reset link
          </Button>
        </FadeIn>
      </form>

      <FadeIn delay={0.24}>
        <p className="mt-8 text-center text-sm text-zinc-500 dark:text-zinc-400">
          Remembered it?{" "}
          <Link
            href="/login"
            className="font-medium text-blue-600 hover:text-blue-500 dark:text-blue-400"
          >
            Sign in
          </Link>
        </p>
      </FadeIn>
    </div>
  );
}
