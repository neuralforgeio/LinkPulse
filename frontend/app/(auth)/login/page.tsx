"use client";

import { useState, type FormEvent } from "react";
import Link from "next/link";
import { motion } from "framer-motion";
import { Eye, EyeOff } from "lucide-react";
import { api, ApiError } from "@/lib/api/client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { FadeIn } from "@/components/motion/fade";

interface AuthUser {
  id: string;
  name: string;
  email: string;
}

interface LoginResponse {
  user: AuthUser;
  access_token: string;
  token_type: string;
  expires_at: string;
}

export default function LoginPage() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [shakeKey, setShakeKey] = useState(0);
  const [user, setUser] = useState<AuthUser | null>(null);

  function fail(message: string) {
    setError(message);
    setShakeKey((k) => k + 1); // replays the shake animation
  }

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!email.trim() || !password) {
      fail("Please fill in both email and password.");
      return;
    }

    setLoading(true);
    try {
      const res = await api<LoginResponse>("/api/v1/auth/login", {
        method: "POST",
        body: JSON.stringify({ email: email.trim(), password }),
      });
      setUser(res.user);
    } catch (err) {
      if (err instanceof ApiError) {
        fail(err.message);
      } else {
        fail("Cannot reach the server. Is the backend running on :8080?");
      }
    } finally {
      setLoading(false);
    }
  }

  if (user) {
    return (
      <motion.div
        initial={{ opacity: 0, scale: 0.92, y: 24 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        transition={{ type: "spring", stiffness: 260, damping: 22 }}
        className="text-center"
      >
        <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-emerald-100 text-emerald-600 dark:bg-emerald-500/15 dark:text-emerald-400">
          <svg
            viewBox="0 0 24 24"
            className="h-8 w-8"
            fill="none"
            stroke="currentColor"
            strokeWidth={2.5}
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <motion.path
              d="M5 13l4 4L19 7"
              initial={{ pathLength: 0 }}
              animate={{ pathLength: 1 }}
              transition={{ duration: 0.45, delay: 0.25 }}
            />
          </svg>
        </div>
        <h1 className="mt-5 text-2xl font-bold tracking-tight text-slate-900 dark:text-white">
          You&apos;re signed in
        </h1>
        <p className="mt-2 text-slate-500 dark:text-slate-400">
          Welcome back,{" "}
          <span className="font-medium text-slate-700 dark:text-slate-200">
            {user.name}
          </span>
          .
        </p>
        <p className="mt-6 rounded-2xl bg-slate-100 px-4 py-3 text-sm text-slate-500 dark:bg-white/5 dark:text-slate-400">
          Your session cookie is active in this browser. The full dashboard
          lands in the next milestone.
        </p>
      </motion.div>
    );
  }

  return (
    <div>
      <FadeIn>
        <h1 className="text-2xl font-bold tracking-tight text-slate-900 dark:text-white">
          Welcome back
        </h1>
        <p className="mt-1.5 text-sm text-slate-500 dark:text-slate-400">
          Sign in to your LinkPulse workspace.
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

        <FadeIn delay={0.16}>
          <Input
            label="Password"
            type={showPassword ? "text" : "password"}
            autoComplete="current-password"
            placeholder="&bull;&bull;&bull;&bull;&bull;&bull;&bull;&bull;"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            trailing={
              <button
                type="button"
                onClick={() => setShowPassword((v) => !v)}
                aria-label={showPassword ? "Hide password" : "Show password"}
                className="text-slate-400 transition hover:text-slate-600 dark:hover:text-slate-300"
              >
                {showPassword ? (
                  <EyeOff className="h-4 w-4" />
                ) : (
                  <Eye className="h-4 w-4" />
                )}
              </button>
            }
          />
        </FadeIn>

        {error && (
          <motion.div
            key={shakeKey}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1, x: [0, -10, 10, -6, 6, -2, 0] }}
            transition={{ duration: 0.4 }}
          >
            <p className="rounded-xl bg-red-50 px-4 py-3 text-sm text-red-600 dark:bg-red-500/10 dark:text-red-400">
              {error}
            </p>
          </motion.div>
        )}

        <FadeIn delay={0.24}>
          <Button type="submit" loading={loading} className="w-full py-3">
            Sign in
          </Button>
        </FadeIn>
      </form>

      <FadeIn delay={0.32}>
        <p className="mt-8 text-center text-sm text-slate-500 dark:text-slate-400">
          New to LinkPulse?{" "}
          <Link
            href="/register"
            className="font-medium text-emerald-600 hover:text-emerald-500 dark:text-emerald-400 dark:hover:text-emerald-300"
          >
            Create an account
          </Link>
        </p>
      </FadeIn>
    </div>
  );
}
