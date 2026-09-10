"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, Copy, Link2, Lock, Plus, Search, X } from "lucide-react";
import { api, ApiError } from "@/lib/api/client";
import { useActiveWorkspace } from "@/components/auth/workspace-context";
import { FadeIn } from "@/components/motion/fade";

const PAGE_SIZE = 10;

interface LinkOut {
  id: string;
  short_code: string;
  short_url: string;
  destination_url: string;
  title: string;
  status: string;
  password_protected: boolean;
  click_count: number;
  tags: string[];
  created_at: string;
}

interface LinksResponse {
  links: LinkOut[];
  page: number;
  page_size: number;
  total: number;
}

// Semantic status colors — green ONLY as a status, never as the theme.
const STATUS_BADGE: Record<string, string> = {
  active: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
  expired: "bg-amber-500/10 text-amber-600 dark:text-amber-400",
  disabled: "bg-zinc-500/10 text-zinc-500 dark:text-zinc-400",
  max_clicks_reached: "bg-rose-500/10 text-rose-600 dark:text-rose-400",
};

const STATUS_OPTIONS = [
  { value: "", label: "All statuses" },
  { value: "active", label: "Active" },
  { value: "expired", label: "Expired" },
  { value: "disabled", label: "Disabled" },
  { value: "max_clicks_reached", label: "Limit reached" },
  { value: "password_protected", label: "Password protected" },
];

const SORT_OPTIONS = [
  { value: "created_at:desc", label: "Newest first" },
  { value: "created_at:asc", label: "Oldest first" },
  { value: "click_count:desc", label: "Most clicks" },
  { value: "title:asc", label: "Title A–Z" },
];

const selectCls =
  "rounded-lg border border-transparent bg-zinc-100 px-3.5 py-2.5 text-sm text-zinc-900 shadow-sm transition focus:border-blue-500 focus:bg-white focus:outline-none dark:bg-zinc-800/50 dark:text-white dark:focus:bg-zinc-900";

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export default function LinksPage() {
  const { active } = useActiveWorkspace();
  const tenantId = active?.id ?? null;
  const myRole = active?.role ?? "viewer";
  const canManage =
    myRole === "owner" || myRole === "admin" || myRole === "member";

  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [status, setStatus] = useState("");
  const [sort, setSort] = useState("created_at:desc");
  const [page, setPage] = useState(1);
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [confirmingId, setConfirmingId] = useState<string | null>(null);

  const queryClient = useQueryClient();

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(search), 350);
    return () => clearTimeout(timer);
  }, [search]);

  const [sortCol, sortOrd] = sort.split(":");

  const { data, isLoading, error } = useQuery({
    queryKey: ["links", tenantId, debouncedSearch, status, sort, page],
    queryFn: () => {
      const params = new URLSearchParams({
        page: String(page),
        page_size: String(PAGE_SIZE),
        sort_by: sortCol,
        sort_order: sortOrd,
      });
      if (debouncedSearch) params.set("search", debouncedSearch);
      if (status) params.set("status", status);
      return api<LinksResponse>(`/api/v1/tenants/${tenantId}/links?${params}`);
    },
    enabled: !!tenantId,
  });

  const deleteMutation = useMutation({
    mutationFn: (linkId: string) =>
      api<{ message: string }>(`/api/v1/tenants/${tenantId}/links/${linkId}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      setConfirmingId(null);
      queryClient.invalidateQueries({ queryKey: ["links", tenantId] });
    },
  });

  async function copyShortUrl(link: LinkOut) {
    await navigator.clipboard.writeText(link.short_url);
    setCopiedId(link.id);
    setTimeout(() => setCopiedId(null), 2000);
  }

  const links = data?.links ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  const actionError =
    deleteMutation.error instanceof ApiError
      ? deleteMutation.error.message
      : deleteMutation.error
        ? "Something went wrong."
        : null;

  return (
    <div className="space-y-6">
      <FadeIn>
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-zinc-900 dark:text-white">
              Links
            </h1>
            <p className="mt-1 text-zinc-500 dark:text-zinc-400">
              {total} link{total === 1 ? "" : "s"} in {active?.name}.
            </p>
          </div>
          {canManage && (
            <Link
              href="/app/links/new"
              className="inline-flex items-center gap-2 rounded-lg bg-zinc-900 px-5 py-2.5 text-sm font-semibold text-white shadow-sm transition hover:bg-zinc-800 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
            >
              <Plus className="h-4 w-4" />
              New link
            </Link>
          )}
        </div>
      </FadeIn>

      <FadeIn delay={0.08}>
        <div className="flex flex-wrap items-center gap-3">
          <div className="relative min-w-[14rem] flex-1">
            <Search className="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
            <input
              value={search}
              onChange={(e) => {
                setSearch(e.target.value);
                setPage(1);
              }}
              placeholder="Search title, code, or destination…"
              className="w-full rounded-lg border border-transparent bg-zinc-100 py-2.5 pl-10 pr-4 text-sm text-zinc-900 shadow-sm transition placeholder:text-zinc-400 focus:border-blue-500 focus:bg-white focus:outline-none focus:ring-4 focus:ring-blue-500/10 dark:bg-zinc-800/50 dark:text-white dark:focus:bg-zinc-900"
            />
          </div>
          <select
            value={status}
            onChange={(e) => {
              setStatus(e.target.value);
              setPage(1);
            }}
            className={selectCls}
          >
            {STATUS_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
          <select
            value={sort}
            onChange={(e) => setSort(e.target.value)}
            className={selectCls}
          >
            {SORT_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
        </div>
      </FadeIn>

      {actionError && (
        <p className="rounded-lg bg-rose-50 px-4 py-3 text-sm text-rose-600 dark:bg-rose-500/10 dark:text-rose-400">
          {actionError}
        </p>
      )}

      <FadeIn delay={0.16}>
        <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
          {isLoading && (
            <div className="space-y-3 p-5">
              {[0, 1, 2, 3].map((i) => (
                <div
                  key={i}
                  className="h-16 animate-pulse rounded-lg bg-zinc-100 dark:bg-zinc-800/60"
                />
              ))}
            </div>
          )}

          {error && (
            <p className="p-6 text-sm text-rose-600 dark:text-rose-400">
              Failed to load links. Is the backend running?
            </p>
          )}

          {!isLoading && !error && links.length === 0 && (
            <div className="flex flex-col items-center gap-3 px-6 py-16 text-center">
              <div className="flex h-14 w-14 items-center justify-center rounded-xl bg-blue-600/10 text-blue-600 dark:text-blue-400">
                <Link2 className="h-7 w-7" />
              </div>
              <div>
                <p className="font-semibold text-zinc-900 dark:text-white">
                  {search || status
                    ? "No links match your filters"
                    : "No links yet"}
                </p>
                <p className="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
                  {search || status
                    ? "Try a different search or status."
                    : "Create your first short link to get started."}
                </p>
              </div>
              {canManage && !search && !status && (
                <Link
                  href="/app/links/new"
                  className="mt-1 inline-flex items-center gap-2 rounded-lg bg-zinc-900 px-5 py-2.5 text-sm font-semibold text-white shadow-sm transition hover:bg-zinc-800 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
                >
                  <Plus className="h-4 w-4" />
                  Create your first link
                </Link>
              )}
            </div>
          )}

          {links.map((link) => (
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
                    onClick={() => copyShortUrl(link)}
                    title="Copy short URL"
                    className="shrink-0 text-zinc-400 transition hover:text-blue-600 dark:hover:text-blue-400"
                  >
                    {copiedId === link.id ? (
                      <Check className="h-4 w-4 text-blue-600" />
                    ) : (
                      <Copy className="h-4 w-4" />
                    )}
                  </button>
                </div>
                <p className="mt-0.5 text-xs text-zinc-400">
                  {link.click_count.toLocaleString("en-US")} click
                  {link.click_count === 1 ? "" : "s"}
                </p>
              </div>

              <div className="min-w-0 flex-1">
                <p className="truncate font-medium text-zinc-900 dark:text-white">
                  {link.title || "Untitled"}
                </p>
                <p className="truncate text-sm text-zinc-500 dark:text-zinc-400">
                  {link.destination_url}
                </p>
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
                  <span
                    className="flex items-center rounded-full bg-zinc-500/10 px-2 py-1 text-zinc-500 dark:text-zinc-400"
                    title="Password protected"
                  >
                    <Lock className="h-3 w-3" />
                  </span>
                )}
              </div>

              <p className="hidden shrink-0 text-xs text-zinc-400 lg:block">
                {formatDate(link.created_at)}
              </p>

              {canManage && (
                <div className="shrink-0">
                  {confirmingId === link.id ? (
                    <div className="flex items-center gap-2">
                      <button
                        type="button"
                        onClick={() => deleteMutation.mutate(link.id)}
                        className="rounded-full bg-rose-600 px-3.5 py-1.5 text-xs font-semibold text-white transition hover:bg-rose-500"
                      >
                        Delete
                      </button>
                      <button
                        type="button"
                        onClick={() => setConfirmingId(null)}
                        className="text-xs text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300"
                      >
                        Cancel
                      </button>
                    </div>
                  ) : (
                    <button
                      type="button"
                      onClick={() => setConfirmingId(link.id)}
                      title="Delete link"
                      className="flex h-8 w-8 items-center justify-center rounded-lg text-zinc-400 transition hover:bg-rose-50 hover:text-rose-600 dark:hover:bg-rose-500/10 dark:hover:text-rose-400"
                    >
                      <X className="h-4 w-4" />
                    </button>
                  )}
                </div>
              )}
            </div>
          ))}
        </div>
      </FadeIn>

      {total > PAGE_SIZE && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-zinc-500 dark:text-zinc-400">
            Page {page} of {totalPages}
          </p>
          <div className="flex gap-2">
            <button
              type="button"
              disabled={page <= 1}
              onClick={() => setPage((p) => p - 1)}
              className="rounded-lg border border-zinc-200 px-4 py-2 text-sm font-medium text-zinc-600 transition hover:border-zinc-300 disabled:opacity-40 dark:border-zinc-800 dark:text-zinc-300"
            >
              Previous
            </button>
            <button
              type="button"
              disabled={page >= totalPages}
              onClick={() => setPage((p) => p + 1)}
              className="rounded-lg border border-zinc-200 px-4 py-2 text-sm font-medium text-zinc-600 transition hover:border-zinc-300 disabled:opacity-40 dark:border-zinc-800 dark:text-zinc-300"
            >
              Next
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
