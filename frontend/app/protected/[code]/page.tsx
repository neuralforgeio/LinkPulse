"use client";

import { useState, type FormEvent } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Eye, EyeOff, Lock } from "lucide-react";
import { api, ApiError } from "@/lib/api/client";
import { Logo } from "@/components/brand/logo";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

const API_BASE =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

export default function ProtectedLinkPage() {
  const params = useParams<{ code: string }>();
  const code = params?.code ?? "";

  const [password, setPassword] = useState("");
  const [show, setShow] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!password) {
      setError("Please enter the password.");
      return;
    }

    setLoading(true);
    setError(null);
    try {
      // Correct password → the backend sets the link-access cookie.
      await api(`/api/v1/public/links/${code}/verify-password`, {
        method: "POST",
        body: JSON.stringify({ password }),
      });
      // Cookie is set — go to the short link itself; the redirect now passes.
      window.location.href = `${API_BASE}/${code}`;
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("Cannot reach the server. Is the backend running?");
      }
      setLoading(false);
    }
  }

  return (
    <div className="relative flex min-h-dvh items-center justify-center overflow-hidden bg-zinc-50 px-6 dark:bg-zinc-950">
      <div
        className="dot-grid-light absolute inset-0 dark:hidden"
        aria-hidden="true"
      />
      <div
        className="dot-grid absolute inset-0 hidden dark:block"
        aria-hidden="true"
      />
      <div
        className="absolute -top-32 h-96 w-96 rounded-full bg-blue-200/50 blur-[120px] dark:bg-blue-500/15"
        aria-hidden="true"
      />

      <div className="fade-up relative w-full max-w-sm">
        <div className="mb-8 flex justify-center">
          <Logo />
        </div>

        <div className="rounded-xl border border-zinc-200 bg-white p-8 shadow-xl dark:border-zinc-800 dark:bg-zinc-900">
          <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-xl bg-blue-600/10 text-blue-600 dark:text-blue-400">
            <Lock className="h-7 w-7" />
          </div>
          <h1 className="mt-5 text-center text-2xl font-bold tracking-tight text-zinc-900 dark:text-white">
            This link is protected
          </h1>
          <p className="mt-2 text-center text-sm text-zinc-500 dark:text-zinc-400">
            Enter the password to continue to{" "}
            <span className="font-mono font-medium text-zinc-700 dark:text-zinc-200">
              /{code}
            </span>
            .
          </p>

          <form className="mt-6 space-y-5" onSubmit={onSubmit} noValidate>
            <Input
              label="Password"
              type={show ? "text" : "password"}
              placeholder="&bull;&bull;&bull;&bull;&bull;&bull;&bull;&bull;"
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

            {error && (
              <p className="rounded-lg bg-rose-50 px-4 py-3 text-sm text-rose-600 dark:bg-rose-500/10 dark:text-rose-400">
                {error}
              </p>
            )}

            <Button type="submit" loading={loading} className="w-full py-3">
              Unlock link
            </Button>
          </form>

          <p className="mt-6 text-center">
            <Link
              href="/"
              className="text-sm font-medium text-blue-600 hover:text-blue-500 dark:text-blue-400"
            >
              Back to home
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
}
