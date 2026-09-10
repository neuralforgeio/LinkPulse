"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { motion } from "framer-motion";
import { api, ApiError } from "@/lib/api/client";
import { useAuth } from "@/components/auth/auth-provider";
import { Logo } from "@/components/brand/logo";
import { Button } from "@/components/ui/button";
import { MailOpen, X } from "lucide-react";

interface AcceptInviteResult {
  joined: boolean;
  already_member: boolean;
  tenant: {
    id: string;
    name: string;
    slug: string;
    role: string;
    created_at: string;
  };
}

export default function InvitePage() {
  const { status, user } = useAuth();
  const router = useRouter();
  const params = useParams<{ code: string }>();
  const code = params?.code ?? "";

  const [result, setResult] = useState<AcceptInviteResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  // Guests are sent to sign in, then brought straight back here
  // (PRD 13.9) via the ?next= parameter.
  useEffect(() => {
    if (status === "guest") {
      router.replace(`/login?next=${encodeURIComponent(`/invite/${code}`)}`);
    }
  }, [status, router, code]);

  async function accept() {
    setLoading(true);
    setError(null);
    try {
      const res = await api<AcceptInviteResult>(
        "/api/v1/invitations/accept",
        {
          method: "POST",
          body: JSON.stringify({ invite_code: code }),
        },
      );
      setResult(res);
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("Cannot reach the server. Is the backend running on :8080?");
      }
    } finally {
      setLoading(false);
    }
  }

  if (status !== "authenticated") {
    return (
      <div className="flex min-h-dvh items-center justify-center bg-slate-50 dark:bg-slate-950">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-slate-300 border-t-emerald-500 dark:border-slate-700" />
      </div>
    );
  }

  return (
    <div className="relative flex min-h-dvh items-center justify-center overflow-hidden bg-slate-50 px-6 dark:bg-slate-950">
      <div className="dot-grid-light absolute inset-0 dark:hidden" aria-hidden="true" />
      <div className="dot-grid absolute inset-0 hidden dark:block" aria-hidden="true" />
      <div
        className="absolute -top-32 h-96 w-96 rounded-full bg-emerald-200/50 blur-[120px] dark:bg-emerald-500/15"
        aria-hidden="true"
      />

      <div className="relative w-full max-w-sm">
        <div className="mb-8 flex justify-center">
          <Logo />
        </div>

        {result ? (
          <motion.div
            initial={{ opacity: 0, scale: 0.92, y: 24 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            transition={{ type: "spring", stiffness: 260, damping: 22 }}
            className="rounded-3xl border border-slate-200 bg-white p-8 text-center shadow-xl dark:border-slate-800 dark:bg-slate-900"
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
              {result.joined ? "You're in!" : "Already a member"}
            </h1>
            <p className="mt-2 text-slate-500 dark:text-slate-400">
              {result.joined ? "You've joined" : "You already belong to"}{" "}
              <span className="font-medium text-slate-700 dark:text-slate-200">
                {result.tenant.name}
              </span>{" "}
              as <span className="font-medium">{result.tenant.role}</span>.
            </p>
            <Link
              href="/app/overview"
              className="mt-6 flex w-full items-center justify-center rounded-full bg-gradient-to-r from-emerald-500 to-teal-500 px-5 py-3 text-sm font-semibold text-slate-950 shadow-lg shadow-emerald-500/25 transition hover:from-emerald-400 hover:to-teal-400"
            >
              Go to dashboard
            </Link>
          </motion.div>
        ) : error ? (
          <motion.div
            initial={{ opacity: 0, y: 24 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, ease: "easeOut" }}
            className="rounded-3xl border border-red-200 bg-white p-8 text-center shadow-xl dark:border-red-500/30 dark:bg-slate-900"
          >
            <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-red-50 text-red-500 dark:bg-red-500/10">
              <X />icon? — need an icon. Use X from lucide:
            </div>
            <h1 className="mt-5 text-2xl font-bold tracking-tight text-slate-900 dark:text-white">
              Invitation unavailable
            </h1>
            <p className="mt-2 font-mono text-sm text-slate-400">{code}</p>
            <p className="mt-1 text-sm text-red-600 dark:text-red-400">{error}</p>
            <Link
              href="/"
              className="mt-6 inline-block text-sm font-medium text-emerald-600 hover:text-emerald-500 dark:text-emerald-400"
            >
              Back to home
            </Link>
          </motion.div>
        ) : (
          <motion.div
            initial={{ opacity: 0, y: 24 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, ease: "easeOut" }}
            className="rounded-3xl border border-slate-200 bg-white p-8 shadow-xl dark:border-slate-800 dark:bg-slate-900"
          >
            <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
              <MailOpen className="h-7 w-7" />
            </div>
            <h1 className="mt-5 text-center text-2xl font-bold tracking-tight text-slate-900 dark:text-white">
              You&apos;ve been invited
            </h1>
            <p className="mt-2 text-center text-sm text-slate-500 dark:text-slate-400">
              Signed in as{" "}
              <span className="font-medium text-slate-700 dark:text-slate-200">
                {user?.email}
              </span>
              . Accept to join the workspace.
            </p>
            <p className="mt-5 rounded-xl bg-slate-100 px-4 py-3 text-center font-mono text-lg font-semibold tracking-wider text-slate-800 dark:bg-slate-950/60 dark:text-slate-200">
              {code}
            </p>
            <Button onClick={accept} loading={loading} className="mt-5 w-full py-3">
              Accept invitation
            </Button>
          </motion.div>
        )}
      </div>
    </div>
  );
}
