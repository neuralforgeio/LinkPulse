"use client";

import { useQuery } from "@tanstack/react-query";
import {
  Clock,
  Link2,
  MousePointerClick,
  ShieldCheck,
  Users,
} from "lucide-react";
import { api } from "@/lib/api/client";
import { useAuth } from "@/components/auth/auth-provider";
import { useActiveWorkspace } from "@/components/auth/workspace-context";
import { FadeIn } from "@/components/motion/fade";
import { StatCard } from "@/components/analytics/stat-card";
import { ClicksChart } from "@/components/analytics/clicks-chart";
import type { OverviewData } from "@/components/analytics/types";

export default function OverviewPage() {
  const { user } = useAuth();
  const { active } = useActiveWorkspace();
  const tenantId = active?.id ?? null;

  // Fixed 30-day window for the landing view of the dashboard.
  const end = new Date();
  const start = new Date(end.getTime() - 30 * 86_400_000);
  const startStr = start.toISOString().slice(0, 10);
  const endStr = end.toISOString().slice(0, 10);

  const { data, isLoading } = useQuery({
    queryKey: ["analytics", tenantId, startStr, endStr, "day"],
    queryFn: () =>
      api<OverviewData>(
        `/api/v1/tenants/${tenantId}/analytics/overview?start_date=${startStr}&end_date=${endStr}&granularity=day`,
      ),
    enabled: !!tenantId,
  });

  const fmt = (n?: number) => (n ?? 0).toLocaleString("en-US");

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

      {/* Stats — real numbers now (PRD 13.3) */}
      <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
        {isLoading ? (
          [0, 1, 2, 3].map((i) => (
            <div
              key={i}
              className="h-28 animate-pulse rounded-2xl bg-slate-100 dark:bg-slate-800/60"
            />
          ))
        ) : (
          <>
            <StatCard
              label="Total clicks"
              value={fmt(data?.total_clicks)}
              icon={MousePointerClick}
            />
            <StatCard
              label="Unique visitors"
              value={fmt(data?.unique_clicks_estimate)}
              icon={Users}
            />
            <StatCard
              label="Active links"
              value={fmt(data?.active_links)}
              icon={Link2}
            />
            <StatCard
              label="Expired links"
              value={fmt(data?.expired_links)}
              icon={Clock}
            />
          </>
        )}
      </div>

      {/* Chart */}
      <FadeIn delay={0.2}>
        <div className="rounded-2xl border border-slate-200 bg-white p-5 dark:border-slate-800 dark:bg-slate-900">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
              Clicks over time
            </h3>
            <span className="rounded-md bg-slate-100 px-2.5 py-1 text-xs text-slate-500 dark:bg-slate-800 dark:text-slate-400">
              Last 30 days
            </span>
          </div>
          <div className="mt-4">
            {isLoading ? (
              <div className="h-56 animate-pulse rounded-xl bg-slate-100 dark:bg-slate-800/60" />
            ) : (
              <ClicksChart data={data?.clicks_over_time ?? []} height={240} />
            )}
          </div>
        </div>
      </FadeIn>

      {/* Workspace card */}
      <FadeIn delay={0.3}>
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

      {/* Top links or honest empty state */}
      {!isLoading && data && data.top_links.length > 0 && (
        <FadeIn delay={0.4}>
          <div className="rounded-2xl border border-slate-200 bg-white p-5 dark:border-slate-800 dark:bg-slate-900">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
              Top links
            </h3>
            <div className="mt-2">
              {data.top_links.slice(0, 5).map((l) => (
                <div
                  key={l.link_id}
                  className="flex items-center justify-between border-b border-slate-100 py-2.5 last:border-0 dark:border-slate-800/60"
                >
                  <p className="truncate font-mono text-sm font-semibold text-emerald-600 dark:text-emerald-400">
                    /{l.short_code}
                  </p>
                  <p className="ml-3 shrink-0 text-sm text-slate-500 dark:text-slate-400">
                    {l.clicks.toLocaleString("en-US")} clicks
                  </p>
                </div>
              ))}
            </div>
          </div>
        </FadeIn>
      )}

      {!isLoading && (data?.total_clicks ?? 0) === 0 && (
        <FadeIn delay={0.4}>
          <div className="flex items-start gap-4 rounded-2xl border border-dashed border-slate-300 bg-white/50 p-6 dark:border-slate-700 dark:bg-slate-900/40">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
              <ShieldCheck className="h-5 w-5" />
            </div>
            <div>
              <h3 className="font-semibold text-slate-900 dark:text-white">
                No clicks yet — and that&apos;s expected.
              </h3>
              <p className="mt-1 text-sm leading-relaxed text-slate-500 dark:text-slate-400">
                Create a link, share it, and watch this page fill up with real
                numbers in real time.
              </p>
            </div>
          </div>
        </FadeIn>
      )}
    </div>
  );
}
