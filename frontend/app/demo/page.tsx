"use client";

import { useState } from "react";
import Link from "next/link";
import { motion } from "framer-motion";
import {
  BarChart3, Check, Copy, ExternalLink, FlaskConical, Link2,
  Lock, MousePointerClick, TrendingUp, Users,
} from "lucide-react";
import {
  demoLinks, demoOverview, demoRecentClicks, demoLinkAnalytics,
  type DemoLink,
} from "@/lib/mock/data";
import { Logo } from "@/components/brand/logo";
import { ThemeToggle } from "@/components/theme-toggle";
import { ClicksChart } from "@/components/analytics/clicks-chart";
import { TopList } from "@/components/analytics/top-list";
import { formatDate, formatWhen, fmt, STATUS_BADGE } from "@/lib/utils";

type Tab = "overview" | "links" | "analytics";

export default function DemoPage() {
  const [tab, setTab] = useState<Tab>("overview");
  const [copied, setCopied] = useState<string | null>(null);
  const [selected, setSelected] = useState<DemoLink>(demoLinks[0]);

  function copy(url: string, id: string) {
    navigator.clipboard.writeText(url);
    setCopied(id);
    setTimeout(() => setCopied(null), 2000);
  }

  const tabs: { id: Tab; label: string; icon: typeof BarChart3 }[] = [
    { id: "overview", label: "Overview", icon: BarChart3 },
    { id: "links", label: "Links", icon: Link2 },
    { id: "analytics", label: "Analytics", icon: MousePointerClick },
  ];

  return (
    <div className="min-h-dvh bg-zinc-50 text-zinc-900 dark:bg-zinc-950 dark:text-white">
      {/* Demo banner — always visible (PRD 9.16) */}
      <div className="sticky top-0 z-50 flex items-center justify-center gap-2 bg-blue-600 px-4 py-2 text-center text-xs font-semibold text-white">
        <FlaskConical className="h-3.5 w-3.5" />
        Demo Mode — sample data only, nothing is saved
      </div>

      <header className="mx-auto flex max-w-7xl items-center justify-between px-6 py-5">
        <div className="flex items-center gap-2.5">
          <Logo />
          <span className="rounded-full bg-blue-600/10 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-blue-600 dark:text-blue-400">
            demo
          </span>
        </div>
        <div className="flex items-center gap-3">
          <ThemeToggle />
          <Link
            href="/register"
            className="inline-flex items-center gap-2 rounded-lg bg-zinc-900 px-4 py-2 text-sm font-semibold text-white shadow-sm transition hover:bg-zinc-800 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
          >
            Create free account
          </Link>
        </div>
      </header>

      <main className="mx-auto max-w-7xl space-y-6 px-6 pb-16">
        {/* Workspace header */}
        <div className="flex flex-wrap items-end justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight">Demo Workspace</h1>
            <p className="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
              Signed in as <span className="font-medium">demo@linkpulse.local</span> · owner
            </p>
          </div>
          <div className="flex gap-1 rounded-xl border border-zinc-200 bg-white p-1 dark:border-zinc-800 dark:bg-zinc-900">
            {tabs.map((t) => (
              <button
                key={t.id}
                type="button"
                onClick={() => setTab(t.id)}
                className={`inline-flex items-center gap-1.5 rounded-lg px-3.5 py-2 text-sm font-medium transition ${
                  tab === t.id
                    ? "bg-zinc-900 text-white dark:bg-white dark:text-zinc-900"
                    : "text-zinc-500 hover:text-zinc-900 dark:hover:text-white"
                }`}
              >
                <t.icon className="h-4 w-4" />
                {t.label}
              </button>
            ))}
          </div>
        </div>

        {tab === "overview" && (
          <motion.div initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} className="space-y-6">
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
              {[
                { label: "Total clicks", value: fmt(demoOverview.total_clicks), icon: MousePointerClick },
                { label: "Unique visitors", value: fmt(demoOverview.unique_clicks_estimate), icon: Users },
                { label: "Active links", value: String(demoOverview.active_links), icon: Link2 },
                { label: "Expired links", value: String(demoOverview.expired_links), icon: TrendingUp },
              ].map((s, i) => (
                <div
                  key={s.label}
                  className="rounded-xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900"
                  style={{ animationDelay: `${i * 60}ms` }}
                >
                  <div className="flex items-center justify-between">
                    <p className="text-xs font-medium uppercase tracking-wider text-zinc-400">{s.label}</p>
                    <s.icon className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                  </div>
                  <p className="mt-2 text-3xl font-bold tabular-nums">{s.value}</p>
                </div>
              ))}
            </div>

            <div className="grid gap-4 lg:grid-cols-[2fr_1fr]">
              <div className="rounded-xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
                <div className="mb-3 flex items-center justify-between">
                  <h3 className="text-sm font-semibold">Clicks over time</h3>
                  <span className="rounded-full bg-zinc-100 px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400">
                    Last 30 days
                  </span>
                </div>
                <ClicksChart data={demoOverview.clicks_over_time} height={240} />
              </div>
              <div className="rounded-xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
                <h3 className="text-sm font-semibold">Top links</h3>
                <ul className="mt-3 space-y-3">
                  {demoOverview.top_links.slice(0, 5).map((l) => (
                    <li key={l.link_id} className="flex items-center justify-between gap-3 text-sm">
                      <span className="truncate font-mono text-blue-600 dark:text-blue-400">/{l.short_code}</span>
                      <span className="shrink-0 font-medium">{fmt(l.clicks)}</span>
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          </motion.div>
        )}

        {tab === "links" && (
          <motion.div initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }}>
            <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
              {demoLinks.map((link) => (
                <div
                  key={link.id}
                  className="flex items-center gap-4 border-b border-zinc-100 px-5 py-4 last:border-0 dark:border-zinc-800/60"
                >
                  <div className="w-44 shrink-0">
                    <div className="flex items-center gap-1.5">
                      <span className="truncate font-mono text-sm font-semibold text-blue-600 dark:text-blue-400">
                        /{link.short_code}
                      </span>
                      <button
                        type="button"
                        onClick={() => copy(link.short_url, link.id)}
                        title="Copy short URL"
                        className="shrink-0 text-zinc-400 transition hover:text-blue-600 dark:hover:text-blue-400"
                      >
                        {copied === link.id ? (
                          <Check className="h-4 w-4 text-blue-600" />
                        ) : (
                          <Copy className="h-4 w-4" />
                        )}
                      </button>
                    </div>
                    <p className="mt-0.5 text-xs text-zinc-400">{fmt(link.click_count)} clicks</p>
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="truncate font-medium">{link.title || "Untitled"}</p>
                    <p className="truncate text-sm text-zinc-500 dark:text-zinc-400">{link.destination_url}</p>
                    {link.tags.length > 0 && (
                      <div className="mt-1.5 flex flex-wrap gap-1.5">
                        {link.tags.map((tag) => (
                          <span
                            key={tag}
                            className="rounded-full bg-zinc-100 px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400"
                          >
                            {tag}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                  <div className="hidden shrink-0 items-center gap-1.5 sm:flex">
                    <span
                      className={`rounded-full px-2.5 py-1 text-[10px] font-semibold uppercase tracking-wide ${
                        STATUS_BADGE[link.status] ?? STATUS_BADGE.disabled
                      }`}
                    >
                      {link.status.replace(/_/g, " ")}
                    </span>
                    {link.password_protected && (
                      <span className="flex items-center rounded-full bg-zinc-500/10 px-2 py-1 text-zinc-500 dark:text-zinc-400">
                        <Lock className="h-3 w-3" />
                      </span>
                    )}
                  </div>
                  <p className="hidden shrink-0 text-xs text-zinc-400 lg:block">{formatDate(link.created_at)}</p>
                </div>
              ))}
            </div>
          </motion.div>
        )}

        {tab === "analytics" && (
          <motion.div initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} className="space-y-4">
            {/* Pick a link to inspect */}
            <div className="rounded-xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
              <p className="text-xs font-semibold uppercase tracking-wider text-zinc-400">Per-link analytics</p>
              <div className="mt-3 flex flex-wrap gap-2">
                {demoLinks.map((l) => (
                  <button
                    key={l.id}
                    type="button"
                    onClick={() => setSelected(l)}
                    className={`rounded-full px-3.5 py-1.5 text-sm font-medium transition ${
                      selected.id === l.id
                        ? "bg-zinc-900 text-white dark:bg-white dark:text-zinc-900"
                        : "bg-zinc-100 text-zinc-600 hover:bg-zinc-200 dark:bg-zinc-800 dark:text-zinc-300 dark:hover:bg-zinc-700"
                    }`}
                  >
                    /{l.short_code}
                  </button>
                ))}
              </div>
            </div>

            {(() => {
              const stats = demoLinkAnalytics(selected);
              const clicks = demoRecentClicks(selected);
              return (
                <>
                  <div className="grid gap-4 sm:grid-cols-3">
                    <div className="rounded-xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
                      <p className="text-2xl font-bold tabular-nums">{fmt(stats.total_clicks)}</p>
                      <p className="text-xs uppercase tracking-wide text-zinc-400">Total clicks</p>
                    </div>
                    <div className="rounded-xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
                      <p className="text-2xl font-bold tabular-nums">{fmt(stats.unique_clicks_estimate)}</p>
                      <p className="text-xs uppercase tracking-wide text-zinc-400">Unique visitors (est.)</p>
                    </div>
                    <div className="rounded-xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
                      <p className="text-2xl font-bold tabular-nums">{clicks.length}</p>
                      <p className="text-xs uppercase tracking-wide text-zinc-400">Recent clicks shown</p>
                    </div>
                  </div>
                  <div className="rounded-xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
                    <ClicksChart data={stats.clicks_over_time} height={220} />
                  </div>
                  <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                    <TopList title="Top referrers" items={stats.top_referrers} />
                    <TopList title="Top devices" items={stats.top_devices} />
                    <TopList title="Top browsers" items={stats.top_browsers} />
                  </div>
                  <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
                    <p className="border-b border-zinc-100 px-5 py-3 text-sm font-semibold dark:border-zinc-800/60">
                      Recent clicks
                    </p>
                    <div className="divide-y divide-zinc-100 dark:divide-zinc-800/60">
                      {clicks.map((c) => (
                        <div key={c.id} className="flex flex-wrap items-center gap-x-6 gap-y-1 px-5 py-3 text-sm">
                          <span className="w-44 shrink-0 text-zinc-500 dark:text-zinc-400">{formatWhen(c.clicked_at)}</span>
                          <span className="min-w-20 capitalize">{c.browser}</span>
                          <span className="min-w-20 capitalize">{c.os}</span>
                          <span className="min-w-16 capitalize">{c.device_type}</span>
                          <span className="truncate text-zinc-500 dark:text-zinc-400">{c.referrer}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                </>
              );
            })()}
          </motion.div>
        )}

        <div className="rounded-xl border border-blue-200 bg-blue-50/60 p-6 text-center dark:border-blue-500/20 dark:bg-blue-500/5">
          <p className="font-semibold text-blue-900 dark:text-blue-100">Like what you see?</p>
          <p className="mt-1 text-sm text-blue-700 dark:text-blue-300">
            This is sample data. Create an account and run the open-source backend to track real clicks.
          </p>
          <div className="mt-4 flex flex-wrap justify-center gap-3">
            <Link
              href="/register"
              className="inline-flex items-center gap-2 rounded-lg bg-zinc-900 px-5 py-2.5 text-sm font-semibold text-white transition hover:bg-zinc-800 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
            >
              Get started
              <ExternalLink className="h-4 w-4" />
            </Link>
            <a
              href="https://github.com/neuralforgeio/LinkPulse"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 rounded-lg border border-zinc-300 px-5 py-2.5 text-sm font-semibold text-zinc-700 transition hover:border-zinc-400 dark:border-zinc-700 dark:text-zinc-300"
            >
              View source
            </a>
          </div>
        </div>
      </main>
    </div>
  );
}
