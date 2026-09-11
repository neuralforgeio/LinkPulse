"use client";

import { useEffect, useState, type FormEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { motion } from "framer-motion";
import { Eye, EyeOff } from "lucide-react";
import { ApiError } from "@/lib/api/client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { FadeIn } from "@/components/motion/fade";
import { useAuth } from "@/components/auth/auth-provider";

function nextRedirect(): string {
  const next = new URLSearchParams(window.location.search).get("next");
  if (next && next.startsWith("/") && !next.startsWith("//")) {
    return next;
  }
  return "/app/overview";
}

export default function LoginPage() {
  const { login, status } = useAuth();
  const router = useRouter();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [shakeKey, setShakeKey] = useState(0);

  useEffect(() => {
    if (status === "authenticated") {
      router.replace(nextRedirect());
    }
  }, [status, router]);

  function fail(message: string) {
    setError(message);
    setShakeKey((k) => k + 1);
  }

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);

    if (!email.trim() || !password) {
      fail("Please fill in both email and password.");
      return;
    }

    setLoading(true);
    try {
      await login(email.trim(), password);
      router.push(nextRedirect());
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

  return (
    <div>
      <FadeIn>
        <h1 className="text-2xl font-bold tracking-tight text-zinc-900 dark:text-white">
          Welcome back
        </h1>
        <p className="mt-1.5 text-sm text-zinc-500 dark:text-zinc-400">
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
                className="text-zinc-400 transition hover:text-zinc-600 dark:hover:text-zinc-300"
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
            <p className="rounded-lg bg-rose-50 px-4 py-3 text-sm text-rose-600 dark:bg-rose-500/10 dark:text-rose-400">
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
        <div className="mt-8 flex flex-col items-center gap-3 text-sm">
          <Link
            href="/forgot-password"
            className="font-medium text-blue-600 hover:text-blue-500 dark:text-blue-400 dark:hover:text-blue-300"
          >
            Forgot your password?
          </Link>
          <p className="text-zinc-500 dark:text-zinc-400">
            New to LinkPulse?{" "}
            <Link
              href="/register"
              className="font-medium text-blue-600 hover:text-blue-500 dark:text-blue-400"
            >
              Create an account
            </Link>
          </p>
        </div>
      </FadeIn>
    </div>
  );
}
