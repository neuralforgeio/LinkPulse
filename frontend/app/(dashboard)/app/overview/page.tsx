"use client";

import {
  BarChart3,
  Link2,
  MousePointerClick,
  ShieldCheck,
  Users,
} from "lucide-react";
import { useAuth } from "@/components/auth/auth-provider";
import { useActiveWorkspace } from "@/components/auth/workspace-context";
import { FadeIn } from "@/components/motion/fade";

const PLACEHOLDER_STATS = [
  { label: "Total clicks", icon: MousePointerClick },
  { label: "Active links", icon: Link2 },
  { label: "Unique visitors", icon: Users },
  { label: "Top referrer", icon: BarChart3 },
];

export default function OverviewPage() {
  const { user } = useAuth();
  const { active } = useActiveWorkspace();

  return (
    <div className="space-y-6">
      <FadeIn>
        <h1 className="text-2xl font-bold tracking-tight text-slate-900 dark:text-white">
          Welcome back, {user?.name}.
        </h1>
        <p className="mt-1 text-slate-500 dark:text-slate-400">
          Here&apos;s what&apos;s happening in{" "}
          <span className="font-medium text-slate-700 dark:text-slate-200">
            {active?.name}
          </span>
          .
        </p>
      </FadeIn>

      {/* Stats — honest empty state until analytics lands (M4–M5) */}
      <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
        {PLACEHOLDER_STATS.map((stat, i) => (
          <FadeIn key={stat.label} delay={0.08 + i * 0.06}>
            <div className="rounded-2xl border border-slate-200 bg-white p-5 dark:border-slate-800 dark:bg-slate-900">
              <div className="flex items-center justify-between">
                <p className="text-sm text-slate-500 dark:text-slate-400">
                  {stat.label}
                </p>
                <stat.icon className="h-4 w-4 text-slate-300 dark:text-slate-600" />
              </div>
              <p className="mt-2 text-3xl font-bold tracking-tight text-slate-300 dark:text-slate-700">
                —
              </p>
            </div>
          </FadeIn>
        ))}
      </div>

      {/* Workspace card */}
      <FadeIn delay={0.35}>
        <div className="rounded-2xl border border-slate-200 bg-white p-6 dark:border-slate-800 dark:bg-slate-900">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-lg font-semibold tracking-tight text-slate-900 dark:text-white">
                {active?.name}
              </h2>
              <p className="mt-1 font-mono text-sm text-slate-500 dark:text-slate-400">
                /{active?.slug}
              </p>
            </div>
            <span className="rounded-full bg-emerald-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-wide text-emerald-700 dark:text-emerald-300">
              {active?.role}
            </span>
          </div>
        </div>
      </FadeIn>

      {/* Empty state banner */}
      <FadeIn delay={0.45}>
        <div className="flex items-start gap-4 rounded-2xl border border-dashed border-slate-300 bg-white/50 p-6 dark:border-slate-700 dark:bg-slate-900/40">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
            <ShieldCheck className="h-5 w-5" />
          </div>
          <div>
            <h3 className="font-semibold text-slate-900 dark:text-white">
              No analytics yet — and that&apos;s expected.
            </h3>
            <p className="mt-1 text-sm leading-relaxed text-slate-500 dark:text-slate-400">
              The redirect engine and async click tracking arrive in Milestone
              4. Once links start receiving clicks, this page fills with real
              numbers in real time.
            </p>
          </div>
        </div>
      </FadeIn>
    </div>
  );
}
