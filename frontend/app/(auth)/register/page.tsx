"use client";

import { useState, type FormEvent } from "react";
import Link from "next/link";
import { motion } from "framer-motion";
import { Eye, EyeOff } from "lucide-react";
import { api, ApiError } from "@/lib/api/client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { FadeIn } from "@/components/motion/fade";

interface RegisterResponse {
  user: { id: string; name: string; email: string };
  tenant: { id: string; name: string; slug: string };
}

export default function RegisterPage() {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [errors, setErrors] = useState<string[]>([]);
  const [shakeKey, setShakeKey] = useState(0);
  const [done, setDone] = useState<RegisterResponse | null>(null);

  function fail(problems: string[]) {
    setErrors(problems);
    setShakeKey((k) => k + 1);
  }

  function validate(): string[] {
    const problems: string[] = [];
    if (name.trim().length < 2) {
      problems.push("Name must be at least 2 characters.");
    }
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim())) {
      problems.push("Please enter a valid email address.");
    }
    if (password.length < 8) {
      problems.push("Password must be at least 8 characters.");
    } else if (!/[A-Za-z]/.test(password) || !/[0-9]/.test(password)) {
      problems.push("Password must contain letters and numbers.");
    }
    return problems;
  }

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const problems = validate();
    if (problems.length > 0) {
      fail(problems);
      return;
    }

    setLoading(true);
    try {
      const res = await api<RegisterResponse>("/api/v1/auth/register", {
        method: "POST",
        body: JSON.stringify({
          name: name.trim(),
          email: email.trim(),
          password,
        }),
      });
      setDone(res);
    } catch (err) {
      if (err instanceof ApiError) {
        fail([err.message]);
      } else {
        fail(["Cannot reach the server. Is the backend running on :8080?"]);
      }
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
        <h1 className="mt-5 text-2xl font-bold tracking-tight text-zinc-900 dark:text-white">
          Account created
        </h1>
        <p className="mt-2 text-zinc-500 dark:text-zinc-400">
          Welcome aboard,{" "}
          <span className="font-medium text-zinc-700 dark:text-zinc-200">
            {done.user.name}
          </span>
          .
        </p>
        <motion.p
          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.3, duration: 0.5 }}
          className="mt-4 rounded-lg bg-zinc-100 px-4 py-3 text-sm text-zinc-600 dark:bg-zinc-900 dark:text-zinc-300"
        >
          Your workspace{" "}
          <span className="font-semibold text-zinc-800 dark:text-white">
            &ldquo;{done.tenant.name}&rdquo;
          </span>{" "}
          is ready.
        </motion.p>
        <Link
          href="/login"
          className="mt-6 flex w-full items-center justify-center rounded-lg bg-zinc-900 px-5 py-3 text-sm font-semibold text-white shadow-sm transition hover:bg-zinc-800 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
        >
          Continue to sign in
        </Link>
      </motion.div>
    );
  }

  return (
    <div>
      <FadeIn>
        <h1 className="text-2xl font-bold tracking-tight text-zinc-900 dark:text-white">
          Create your account
        </h1>
        <p className="mt-1.5 text-sm text-zinc-500 dark:text-zinc-400">
          Your workspace is one minute away.
        </p>
      </FadeIn>

      <form className="mt-8 space-y-5" onSubmit={onSubmit} noValidate>
        <FadeIn delay={0.08}>
          <Input
            label="Name"
            type="text"
            autoComplete="name"
            placeholder="Dearly"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </FadeIn>

        <FadeIn delay={0.16}>
          <Input
            label="Email"
            type="email"
            autoComplete="email"
            placeholder="you@company.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
        </FadeIn>

        <FadeIn delay={0.24}>
          <Input
            label="Password"
            type={showPassword ? "text" : "password"}
            autoComplete="new-password"
            placeholder="&bull;&bull;&bull;&bull;&bull;&bull;&bull;&bull;"
            hint="At least 8 characters, with letters and numbers."
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

        {errors.length > 0 && (
          <motion.div
            key={shakeKey}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1, x: [0, -10, 10, -6, 6, -2, 0] }}
            transition={{ duration: 0.4 }}
          >
            <div className="rounded-lg bg-rose-50 px-4 py-3 text-sm text-rose-600 dark:bg-rose-500/10 dark:text-rose-400">
              <ul className="list-disc space-y-1 pl-4">
                {errors.map((problem) => (
                  <li key={problem}>{problem}</li>
                ))}
              </ul>
            </div>
          </motion.div>
        )}

        <FadeIn delay={0.32}>
          <Button type="submit" loading={loading} className="w-full py-3">
            Create account
          </Button>
        </FadeIn>
      </form>

      <FadeIn delay={0.4}>
        <p className="mt-8 text-center text-sm text-zinc-500 dark:text-zinc-400">
          Already have an account?{" "}
          <Link
            href="/login"
            className="font-medium text-blue-600 hover:text-blue-500 dark:text-blue-400 dark:hover:text-blue-300"
          >
            Sign in
          </Link>
        </p>
      </FadeIn>
    </div>
  );
}
