"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Clock, Link2, MousePointerClick, Users } from "lucide-react";
import { api } from "@/lib/api/client";
import { useActiveWorkspace } from "@/components/auth/workspace-context";
import { FadeIn } from "@/components/motion/fade";
import { StatCard } from "@/components/analytics/stat-card";
import { ClicksChart } from "@/components/analytics/clicks-chart";
import { TopList } from "@/components/analytics/top-list";
import type { OverviewData } from "@/components/analytics/types";

const RANGES = [
  { days: 7, label: "7 days" },
  { days: 30, label: "30 days" },
  { days: 90, label: "90 days" },
];

const GRANULARITIES = [
  { value: "day", label: "Daily" },
  { value: "week", label: "Weekly" },
  { value: "month", label: "Monthly" },
];

function rangeFor(days: number) {
  const end = new Date();
  const start = new Date(end.getTime() - days * 86_400_000);
  return {
    start: start.toISOString().slice(0, 10),
    end: end.toISOString().slice(0, 10),
  };
}

export default function AnalyticsPage() {
  const { active } = useActiveWorkspace();
  const tenantId = active?.id ?? null;

  const [days, setDays] = useState(30);
  const [granularity, setGranularity] = useState("day");
  const { start, end } = rangeFor(days);

  const { data, isLoading, error } = useQuery({
    queryKey: ["analytics", tenantId, start, end, granularity],
    queryFn: () =>
      api<OverviewData>(
        `/api/v1/tenants/${tenantId}/analytics/overview?start_date=${start}&end_date=${end}&granularity=${granularity}`,
      ),
    enabled: !!tenantId,
  });

  const fmt = (n?: number) => (n ?? 0).toLocaleString("en-US");

  return (
    <div className="space-y-6">
      <FadeIn>
        <div className="flex flex-wrap items-end justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-slate-900 dark:text-white">
              Analytics
            </h1>
            <p className="mt-1 text-slate-500 dark:text-slate-400">
              Click performance across {active?.name}.
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {RANGES.map((r) => (
              <button
                key={r.days}
                type="button"
                onClick={() => setDays(r.days)}
                className={`rounded-full px-4 py-1.5 text-sm font-medium transition ${
                  days === r.days
                    ? "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300"
                    : "border border-slate-200 text-slate-500 hover:text-slate-900 dark:border-slate-800 dark:text-slate-400 dark:hover:text-white"
                }`}
              >
                {r.label}
              </button>
            ))}
            <select
              value={granularity}
              onChange={(e) => setGranularity(e.target.value)}
              className="rounded-full border border-slate-200 bg-white px-3.5 py-1.5 text-sm text-slate-700 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-200"
            >
              {GRANULARITIES.map((g) => (
                <option key={g.value} value={g.value}>
                  {g.label}
                </option>
              ))}
            </select>
          </div>
        </div>
      </FadeIn>

      {error && (
        <p className="rounded-xl bg-red-50 px-4 py-3 text-sm text-red-600 dark:bg-red-500/10 dark:text-red-400">
          Failed to load analytics. Is the backend running?
        </p>
      )}

      {/* Stat cards */}
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
          <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
            Clicks over time
          </h3>
          <div className="mt-4">
            {isLoading ? (
              <div className="h-64 animate-pulse rounded-xl bg-slate-100 dark:bg-slate-800/60" />
            ) : (
              <ClicksChart data={data?.clicks_over_time ?? []} />
            )}
          </div>
        </div>
      </FadeIn>

      {/* Top links */}
      <FadeIn delay={0.28}>
        <div className="rounded-2xl border border-slate-200 bg-white p-5 dark:border-slate-800 dark:bg-slate-900">
          <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
            Top links
          </h3>
          {isLoading ? (
            <div className="mt-4 h-24 animate-pulse rounded-xl bg-slate-100 dark:bg-slate-800/60" />
          ) : data && data.top_links.length > 0 ? (
            <div className="mt-2">
              {data.top_links.map((l) => (
                <div
                  key={l.link_id}
                  className="flex items-center justify-between border-b border-slate-100 py-2.5 last:border-0 dark:border-slate-800/60"
                >
                  <div className="min-w-0">
                    <p className="truncate font-mono text-sm font-semibold text-emerald-600 dark:text-emerald-400">
                      /{l.short_code}
                    </p>
                    <p className="truncate text-sm text-slate-500 dark:text-slate-400">
                      {l.title}
                    </p>
                  </div>
                  <p className="ml-3 shrink-0 font-semibold text-slate-900 dark:text-white">
                    {l.clicks.toLocaleString("en-US")}
                  </p>
                </div>
              ))}
            </div>
          ) : (
            <p className="mt-3 text-sm text-slate-400 dark:text-slate-500">
              No clicks yet.
            </p>
          )}
        </div>
      </FadeIn>

      {/* Top rankings */}
      <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-3">
        <TopList title="Top referrers" items={data?.top_referrers ?? []} />
        <TopList title="Top devices" items={data?.top_devices ?? []} />
        <TopList title="Top browsers" items={data?.top_browsers ?? []} />
        <TopList title="Top operating systems" items={data?.top_os ?? []} />
        <TopList title="Top campaigns" items={data?.top_campaigns ?? []} />
      </div>
    </div>
  );
}
