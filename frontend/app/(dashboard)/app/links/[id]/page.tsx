"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowLeft, BarChart3, Check, Copy, ExternalLink, Lock,
  MousePointerClick, QrCode, Settings2, Trash2,
} from "lucide-react";
import { api, ApiError } from "@/lib/api/client";
import { useActiveWorkspace } from "@/components/auth/workspace-context";
import { FadeIn } from "@/components/motion/fade";
import { CountUp } from "@/components/motion/count-up";
import { QrCode as QrCanvas } from "@/components/links/qr-code";
import { ClicksChart } from "@/components/analytics/clicks-chart";
import { TopList } from "@/components/analytics/top-list";
import { formatDate, formatWhen, fmt, STATUS_BADGE } from "@/lib/utils";

type Tab = "overview" | "analytics" | "qr" | "settings";

interface LinkOut {
  id: string;
  short_code: string;
  short_url: string;
  destination_url: string;
  title: string;
  status: string;
  password_protected: boolean;
  expires_at: string | null;
  max_clicks: number | null;
  click_count: number;
  tags: string[];
  utm_source: string;
  utm_medium: string;
  utm_campaign: string;
  utm_term: string;
  utm_content: string;
  created_at: string;
  updated_at: string;
}

interface LinkAnalytics {
  link_id: string;
  short_code: string;
  total_clicks: number;
  unique_clicks_estimate: number;
  clicks_over_time: { date: string; clicks: number }[];
  top_referrers: { name: string; clicks: number }[];
  top_browsers: { name: string; clicks: number }[];
  top_devices: { name: string; clicks: number }[];
  top_os: { name: string; clicks: number }[];
  top_sources: { name: string; clicks: number }[];
  top_campaigns: { name: string; clicks: number }[];
}

interface ClicksPage {
  clicks: {
    id: string;
    clicked_at: string;
    browser: string;
    os: string;
    device_type: string;
    referrer: string;
  }[];
  page: number;
  page_size: number;
  total: number;
}

const inputCls =
  "w-full rounded-lg border border-transparent bg-zinc-100 px-3.5 py-2.5 text-sm text-zinc-900 shadow-sm transition focus:border-blue-500 focus:bg-white focus:outline-none dark:bg-zinc-800/50 dark:text-white dark:focus:bg-zinc-900";

function daysAgoISO(n: number): string {
  const d = new Date();
  d.setUTCDate(d.getUTCDate() - n);
  return d.toISOString().slice(0, 10);
}

export default function LinkDetailPage() {
  const params = useParams<{ id: string }>();
  const linkId = params.id;
  const router = useRouter();
  const { active } = useActiveWorkspace();
  const tenantId = active?.id ?? null;
  const myRole = active?.role ?? "viewer";
  const canManage = myRole === "owner" || myRole === "admin" || myRole === "member";

  const [tab, setTab] = useState<Tab>("overview");
  const [copied, setCopied] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [saveMsg, setSaveMsg] = useState<string | null>(null);
  const [form, setForm] = useState<{
    title: string; destination_url: string; expires_at: string;
    max_clicks: string; password: string; status: string;
  } | null>(null);

  const linkQuery = useQuery({
    queryKey: ["link", tenantId, linkId],
    queryFn: () => api<LinkOut>(`/api/v1/tenants/${tenantId}/links/${linkId}`),
    enabled: !!tenantId && !!linkId,
  });

  const analyticsQuery = useQuery({
    queryKey: ["linkAnalytics", tenantId, linkId],
    queryFn: () =>
      api<LinkAnalytics>(
        `/api/v1/tenants/${tenantId}/links/${linkId}/analytics?start_date=${daysAgoISO(30)}&end_date=${daysAgoISO(0)}&granularity=day`,
      ),
    enabled: !!tenantId && !!linkId && tab === "analytics",
  });

  const clicksQuery = useQuery({
    queryKey: ["linkClicks", tenantId, linkId],
    queryFn: () =>
      api<ClicksPage>(`/api/v1/tenants/${tenantId}/links/${linkId}/clicks?page_size=12`),
    enabled: !!tenantId && !!linkId && tab === "analytics",
  });

  const queryClient = useQueryClient();

  const updateMutation = useMutation({
    mutationFn: (body: Record<string, unknown>) =>
      api<LinkOut>(`/api/v1/tenants/${tenantId}/links/${linkId}`, {
        method: "PATCH",
        body: JSON.stringify(body),
      }),
    onSuccess: () => {
      setSaveMsg("Saved.");
      queryClient.invalidateQueries({ queryKey: ["link", tenantId, linkId] });
      queryClient.invalidateQueries({ queryKey: ["links", tenantId] });
      setTimeout(() => setSaveMsg(null), 2500);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () =>
      api<{ message: string }>(`/api/v1/tenants/${tenantId}/links/${linkId}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["links", tenantId] });
      router.push("/app/links");
    },
  });

  const link = linkQuery.data;

  function copyUrl() {
    if (!link) return;
    navigator.clipboard.writeText(link.short_url);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  function startEdit() {
    if (!link) return;
    setForm({
      title: link.title,
      destination_url: link.destination_url,
      expires_at: link.expires_at ? link.expires_at.slice(0, 16) : "",
      max_clicks: link.max_clicks != null ? String(link.max_clicks) : "",
      password: "",
      status: link.status === "disabled" ? "disabled" : "active",
    });
  }

  function saveEdit() {
    if (!form) return;
    const body: Record<string, unknown> = {
      title: form.title,
      destination_url: form.destination_url,
      status: form.status,
    };
    body.expires_at = form.expires_at ? new Date(form.expires_at).toISOString() : null;
    body.max_clicks = form.max_clicks ? Number(form.max_clicks) : null;
    if (form.password !== "") body.password = form.password;
    updateMutation.mutate(body);
  }

  const tabs: { id: Tab; label: string; icon: typeof BarChart3 }[] = [
    { id: "overview", label: "Overview", icon: BarChart3 },
    { id: "analytics", label: "Analytics", icon: MousePointerClick },
    { id: "qr", label: "QR Code", icon: QrCode },
    { id: "settings", label: "Settings", icon: Settings2 },
  ];

  const actionError =
    updateMutation.error instanceof ApiError
      ? updateMutation.error.message
      : deleteMutation.error instanceof ApiError
        ? deleteMutation.error.message
        : updateMutation.error || deleteMutation.error
          ? "Something went wrong."
          : null;

  return (
    <div className="space-y-6">
      <FadeIn>
        <button
          type="button"
          onClick={() => router.push("/app/links")}
          className="inline-flex items-center gap-1.5 text-sm text-zinc-500 transition hover:text-zinc-900 dark:hover:text-white"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to links
        </button>
      </FadeIn>

      {linkQuery.isLoading && (
        <div className="space-y-3">
          <div className="h-24 animate-pulse rounded-xl bg-zinc-100 dark:bg-zinc-800/60" />
          <div className="h-64 animate-pulse rounded-xl bg-zinc-100 dark:bg-zinc-800/60" />
        </div>
      )}

      {linkQuery.error && (
        <p className="rounded-lg bg-rose-50 px-4 py-3 text-sm text-rose-600 dark:bg-rose-500/10 dark:text-rose-400">
          Failed to load this link. It may have been deleted, or the backend is unreachable.
        </p>
      )}

      {link && (
        <>
          <FadeIn delay={0.06}>
            <div className="rounded-xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
              <div className="flex flex-wrap items-start justify-between gap-4">
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <h1 className="truncate text-xl font-bold tracking-tight text-zinc-900 dark:text-white">
                      {link.title || "Untitled"}
                    </h1>
                    <span
                      className={`rounded-full px-2.5 py-1 text-[10px] font-semibold uppercase tracking-wide ${
                        STATUS_BADGE[link.status] ?? STATUS_BADGE.disabled
                      }`}
                    >
                      {link.status.replace(/_/g, " ")}
                    </span>
                    {link.password_protected && (
                      <span className="flex items-center gap-1 rounded-full bg-zinc-500/10 px-2 py-1 text-[10px] font-medium text-zinc-500 dark:text-zinc-400">
                        <Lock className="h-3 w-3" /> Protected
                      </span>
                    )}
                  </div>
                  <div className="mt-2 flex flex-wrap items-center gap-2 text-sm">
                    <a
                      href={link.short_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-1 font-mono font-semibold text-blue-600 hover:underline dark:text-blue-400"
                    >
                      /{link.short_code}
                      <ExternalLink className="h-3.5 w-3.5" />
                    </a>
                    <button
                      type="button"
                      onClick={copyUrl}
                      className="inline-flex items-center gap-1 rounded-md px-1.5 py-0.5 text-xs text-zinc-500 transition hover:bg-zinc-100 hover:text-zinc-900 dark:hover:bg-zinc-800 dark:hover:text-white"
                    >
                      {copied ? (
                        <Check className="h-3.5 w-3.5 text-blue-600" />
                      ) : (
                        <Copy className="h-3.5 w-3.5" />
                      )}
                      {copied ? "Copied" : "Copy"}
                    </button>
                  </div>
                </div>
                <div className="flex items-center gap-6 px-2 text-right">
                  <div>
                    <p className="text-3xl font-bold tabular-nums text-zinc-900 dark:text-white">
                      <CountUp to={link.click_count} />
                    </p>
                    <p className="text-xs uppercase tracking-wide text-zinc-400">clicks</p>
                  </div>
                </div>
              </div>

              <div className="mt-5 flex gap-1 border-t border-zinc-100 pt-4 dark:border-zinc-800/60">
                {tabs.map((t) => (
                  <button
                    key={t.id}
                    type="button"
                    onClick={() => setTab(t.id)}
                    className={`inline-flex items-center gap-1.5 rounded-lg px-3.5 py-2 text-sm font-medium transition ${
                      tab === t.id
                        ? "bg-zinc-900 text-white dark:bg-white dark:text-zinc-900"
                        : "text-zinc-500 hover:bg-zinc-100 hover:text-zinc-900 dark:hover:bg-zinc-800 dark:hover:text-white"
                    }`}
                  >
                    <t.icon className="h-4 w-4" />
                    {t.label}
                  </button>
                ))}
              </div>
            </div>
          </FadeIn>

          {actionError && (
            <p className="rounded-lg bg-rose-50 px-4 py-3 text-sm text-rose-600 dark:bg-rose-500/10 dark:text-rose-400">
              {actionError}
            </p>
          )}

          {tab === "overview" && (
            <FadeIn delay={0.1}>
              <div className="grid gap-4 md:grid-cols-2">
                <div className="rounded-xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
                  <p className="text-xs font-semibold uppercase tracking-wider text-zinc-400">
                    Destination
                  </p>
                  <a
                    href={link.destination_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="mt-1 block break-all text-sm font-medium text-blue-600 hover:underline dark:text-blue-400"
                  >
                    {link.destination_url}
                  </a>
                  <dl className="mt-4 grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
                    <div>
                      <dt className="text-xs uppercase tracking-wide text-zinc-400">Created</dt>
                      <dd className="mt-0.5 text-zinc-900 dark:text-white">{formatWhen(link.created_at)}</dd>
                    </div>
                    <div>
                      <dt className="text-xs uppercase tracking-wide text-zinc-400">Expires</dt>
                      <dd className="mt-0.5 text-zinc-900 dark:text-white">
                        {link.expires_at ? formatDate(link.expires_at) : "Never"}
                      </dd>
                    </div>
                    <div>
                      <dt className="text-xs uppercase tracking-wide text-zinc-400">Click limit</dt>
                      <dd className="mt-0.5 text-zinc-900 dark:text-white">
                        {link.max_clicks != null ? fmt(link.max_clicks) : "Unlimited"}
                      </dd>
                    </div>
                    <div>
                      <dt className="text-xs uppercase tracking-wide text-zinc-400">Updated</dt>
                      <dd className="mt-0.5 text-zinc-900 dark:text-white">{formatWhen(link.updated_at)}</dd>
                    </div>
                  </dl>
                </div>

                <div className="rounded-xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
                  <p className="text-xs font-semibold uppercase tracking-wider text-zinc-400">
                    Tags & UTM
                  </p>
                  {link.tags.length > 0 ? (
                    <div className="mt-2 flex flex-wrap gap-1.5">
                      {link.tags.map((tag) => (
                        <span
                          key={tag}
                          className="rounded-full bg-zinc-100 px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400"
                        >
                          {tag}
                        </span>
                      ))}
                    </div>
                  ) : (
                    <p className="mt-2 text-sm text-zinc-400">No tags</p>
                  )}
                  <dl className="mt-3 space-y-1.5 text-sm">
                    {[
                      ["utm_source", link.utm_source],
                      ["utm_medium", link.utm_medium],
                      ["utm_campaign", link.utm_campaign],
                      ["utm_term", link.utm_term],
                      ["utm_content", link.utm_content],
                    ].map(([k, v]) => (
                      <div key={k} className="flex items-baseline justify-between gap-3">
                        <dt className="font-mono text-xs text-zinc-400">{k}</dt>
                        <dd className="truncate text-zinc-900 dark:text-white">{v || "—"}</dd>
                      </div>
                    ))}
                  </dl>
                </div>
              </div>
            </FadeIn>
          )}

          {tab === "analytics" && (
            <FadeIn delay={0.1}>
              <div className="space-y-4">
                {analyticsQuery.isLoading && (
                  <div className="h-64 animate-pulse rounded-xl bg-zinc-100 dark:bg-zinc-800/60" />
                )}
                {analyticsQuery.data && (
                  <>
                    <div className="grid gap-4 sm:grid-cols-3">
                      <div className="rounded-xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
                        <p className="text-2xl font-bold tabular-nums text-zinc-900 dark:text-white">
                          <CountUp to={analyticsQuery.data.total_clicks} />
                        </p>
                        <p className="text-xs uppercase tracking-wide text-zinc-400">Clicks (30 days)</p>
                      </div>
                      <div className="rounded-xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
                        <p className="text-2xl font-bold tabular-nums text-zinc-900 dark:text-white">
                          <CountUp to={analyticsQuery.data.unique_clicks_estimate} />
                        </p>
                        <p className="text-xs uppercase tracking-wide text-zinc-400">Unique visitors (est.)</p>
                      </div>
                      <div className="rounded-xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
                        <p className="text-2xl font-bold tabular-nums text-zinc-900 dark:text-white">
                          {analyticsQuery.data.clicks_over_time.length}
                        </p>
                        <p className="text-xs uppercase tracking-wide text-zinc-400">Days tracked</p>
                      </div>
                    </div>
                    <div className="rounded-xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
                      <ClicksChart data={analyticsQuery.data.clicks_over_time} />
                    </div>
                    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                      <TopList title="Top referrers" items={analyticsQuery.data.top_referrers} />
                      <TopList title="Top devices" items={analyticsQuery.data.top_devices} />
                      <TopList title="Top browsers" items={analyticsQuery.data.top_browsers} />
                      <TopList title="Top operating systems" items={analyticsQuery.data.top_os} />
                      <TopList title="Top UTM sources" items={analyticsQuery.data.top_sources} />
                      <TopList title="Top campaigns" items={analyticsQuery.data.top_campaigns} />
                    </div>
                    <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
                      <p className="border-b border-zinc-100 px-5 py-3 text-sm font-semibold text-zinc-900 dark:border-zinc-800/60 dark:text-white">
                        Recent clicks
                      </p>
                      {clicksQuery.data && clicksQuery.data.clicks.length > 0 ? (
                        <div className="divide-y divide-zinc-100 dark:divide-zinc-800/60">
                          {clicksQuery.data.clicks.map((c) => (
                            <div key={c.id} className="flex flex-wrap items-center gap-x-6 gap-y-1 px-5 py-3 text-sm">
                              <span className="w-40 shrink-0 text-zinc-500 dark:text-zinc-400">
                                {formatWhen(c.clicked_at)}
                              </span>
                              <span className="min-w-20 capitalize text-zinc-900 dark:text-white">{c.browser}</span>
                              <span className="min-w-20 capitalize text-zinc-900 dark:text-white">{c.os}</span>
                              <span className="min-w-16 capitalize text-zinc-900 dark:text-white">{c.device_type}</span>
                              <span className="truncate text-zinc-500 dark:text-zinc-400">{c.referrer || "(direct)"}</span>
                            </div>
                          ))}
                        </div>
                      ) : (
                        <p className="px-5 py-6 text-sm text-zinc-400">No clicks recorded yet.</p>
                      )}
                    </div>
                  </>
                )}
              </div>
            </FadeIn>
          )}

          {tab === "qr" && (
            <FadeIn delay={0.1}>
              <div className="flex flex-col items-center gap-5 rounded-xl border border-zinc-200 bg-white p-8 dark:border-zinc-800 dark:bg-zinc-900">
                <QrCanvas value={link.short_url} size={220} />
                <div className="flex gap-3">
                  <a
                    href={link.short_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-2 rounded-lg bg-zinc-900 px-5 py-2.5 text-sm font-semibold text-white transition hover:bg-zinc-800 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
                  >
                    <ExternalLink className="h-4 w-4" />
                    Open link
                  </a>
                  <button
                    type="button"
                    onClick={copyUrl}
                    className="inline-flex items-center gap-2 rounded-lg border border-zinc-200 px-5 py-2.5 text-sm font-semibold text-zinc-700 transition hover:border-zinc-300 dark:border-zinc-700 dark:text-zinc-300"
                  >
                    {copied ? <Check className="h-4 w-4 text-blue-600" /> : <Copy className="h-4 w-4" />}
                    {copied ? "Copied" : "Copy URL"}
                  </button>
                </div>
                <p className="text-xs text-zinc-400">
                  QR codes are generated client-side and downloadable on the create page.
                </p>
              </div>
            </FadeIn>
          )}

          {tab === "settings" && (
            <FadeIn delay={0.1}>
              <div className="space-y-4">
                {!canManage && (
                  <p className="rounded-lg bg-zinc-100 px-4 py-3 text-sm text-zinc-600 dark:bg-zinc-800/60 dark:text-zinc-300">
                    Your role ({myRole}) is read-only for links.
                  </p>
                )}
                {canManage && !form && (
                  <div className="rounded-xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
                    <div className="space-y-4">
                      {[
                        ["Title", link.title || "—"],
                        ["Destination", link.destination_url],
                        ["Expires", link.expires_at ? formatDate(link.expires_at) : "Never"],
                        ["Click limit", link.max_clicks != null ? fmt(link.max_clicks) : "Unlimited"],
                        ["Status", link.status.replace(/_/g, " ")],
                      ].map(([k, v]) => (
                        <div key={k} className="flex flex-wrap items-baseline justify-between gap-3">
                          <span className="text-xs uppercase tracking-wide text-zinc-400">{k}</span>
                          <span className="max-w-[70%] break-all text-sm text-zinc-900 dark:text-white">{v}</span>
                        </div>
                      ))}
                    </div>
                    <div className="mt-6 flex flex-wrap items-center gap-3">
                      <button
                        type="button"
                        onClick={startEdit}
                        className="inline-flex items-center gap-2 rounded-lg bg-zinc-900 px-5 py-2.5 text-sm font-semibold text-white transition hover:bg-zinc-800 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
                      >
                        <Settings2 className="h-4 w-4" />
                        Edit link
                      </button>
                      <button
                        type="button"
                        onClick={() =>
                          updateMutation.mutate({
                            status: link.status === "disabled" ? "active" : "disabled",
                          })
                        }
                        className="inline-flex items-center gap-2 rounded-lg border border-zinc-200 px-5 py-2.5 text-sm font-semibold text-zinc-700 transition hover:border-zinc-300 dark:border-zinc-700 dark:text-zinc-300"
                      >
                        {link.status === "disabled" ? "Enable link" : "Disable link"}
                      </button>
                    </div>
                  </div>
                )}

                {canManage && form && (
                  <div className="rounded-xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
                    <div className="grid gap-4 sm:grid-cols-2">
                      <label className="block text-sm">
                        <span className="mb-1.5 block font-medium text-zinc-900 dark:text-white">Title</span>
                        <input
                          className={inputCls}
                          value={form.title}
                          onChange={(e) => setForm({ ...form, title: e.target.value })}
                        />
                      </label>
                      <label className="block text-sm">
                        <span className="mb-1.5 block font-medium text-zinc-900 dark:text-white">Status</span>
                        <select
                          className={inputCls}
                          value={form.status}
                          onChange={(e) => setForm({ ...form, status: e.target.value })}
                        >
                          <option value="active">Active</option>
                          <option value="disabled">Disabled</option>
                        </select>
                      </label>
                      <label className="block text-sm sm:col-span-2">
                        <span className="mb-1.5 block font-medium text-zinc-900 dark:text-white">Destination URL</span>
                        <input
                          className={inputCls}
                          value={form.destination_url}
                          onChange={(e) => setForm({ ...form, destination_url: e.target.value })}
                        />
                      </label>
                      <label className="block text-sm">
                        <span className="mb-1.5 block font-medium text-zinc-900 dark:text-white">Expires at</span>
                        <input
                          type="datetime-local"
                          className={inputCls}
                          value={form.expires_at}
                          onChange={(e) => setForm({ ...form, expires_at: e.target.value })}
                        />
                      </label>
                      <label className="block text-sm">
                        <span className="mb-1.5 block font-medium text-zinc-900 dark:text-white">Max clicks</span>
                        <input
                          type="number"
                          min={1}
                          placeholder="Unlimited"
                          className={inputCls}
                          value={form.max_clicks}
                          onChange={(e) => setForm({ ...form, max_clicks: e.target.value })}
                        />
                      </label>
                      <label className="block text-sm sm:col-span-2">
                        <span className="mb-1.5 block font-medium text-zinc-900 dark:text-white">
                          Password (leave blank to keep current)
                        </span>
                        <input
                          type="password"
                          placeholder="New password (optional)"
                          className={inputCls}
                          value={form.password}
                          onChange={(e) => setForm({ ...form, password: e.target.value })}
                        />
                      </label>
                    </div>
                    <div className="mt-6 flex flex-wrap items-center gap-3">
                      <button
                        type="button"
                        onClick={saveEdit}
                        disabled={updateMutation.isPending}
                        className="inline-flex items-center gap-2 rounded-lg bg-zinc-900 px-5 py-2.5 text-sm font-semibold text-white transition hover:bg-zinc-800 disabled:opacity-60 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
                      >
                        <Check className="h-4 w-4" />
                        {updateMutation.isPending ? "Saving…" : "Save changes"}
                      </button>
                      <button
                        type="button"
                        onClick={() => setForm(null)}
                        className="rounded-lg border border-zinc-200 px-5 py-2.5 text-sm font-semibold text-zinc-700 transition hover:border-zinc-300 dark:border-zinc-700 dark:text-zinc-300"
                      >
                        Cancel
                      </button>
                      {saveMsg && <span className="text-sm font-medium text-emerald-600 dark:text-emerald-400">{saveMsg}</span>}
                    </div>
                  </div>
                )}

                {canManage && (
                  <div className="rounded-xl border border-rose-200 bg-rose-50/50 p-6 dark:border-rose-500/20 dark:bg-rose-500/5">
                    <p className="text-sm font-semibold text-rose-600 dark:text-rose-400">Danger zone</p>
                    <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
                      Deleting soft-deletes the link. It stops redirecting immediately; analytics are kept.
                    </p>
                    <div className="mt-4">
                      {confirmDelete ? (
                        <div className="flex items-center gap-3">
                          <button
                            type="button"
                            onClick={() => deleteMutation.mutate()}
                            disabled={deleteMutation.isPending}
                            className="inline-flex items-center gap-2 rounded-lg bg-rose-600 px-5 py-2.5 text-sm font-semibold text-white transition hover:bg-rose-500 disabled:opacity-60"
                          >
                            <Trash2 className="h-4 w-4" />
                            {deleteMutation.isPending ? "Deleting…" : "Yes, delete this link"}
                          </button>
                          <button
                            type="button"
                            onClick={() => setConfirmDelete(false)}
                            className="text-sm text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300"
                          >
                            Cancel
                          </button>
                        </div>
                      ) : (
                        <button
                          type="button"
                          onClick={() => setConfirmDelete(true)}
                          className="inline-flex items-center gap-2 rounded-lg border border-rose-300 px-5 py-2.5 text-sm font-semibold text-rose-600 transition hover:bg-rose-100 dark:border-rose-500/40 dark:text-rose-400 dark:hover:bg-rose-500/10"
                        >
                          <Trash2 className="h-4 w-4" />
                          Delete link
                        </button>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </FadeIn>
          )}
        </>
      )}
    </div>
  );
}
