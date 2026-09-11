"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Building2,
  KeyRound,
  Link2,
  LogIn,
  MailPlus,
  ScrollText,
  Users,
} from "lucide-react";
import { api } from "@/lib/api/client";
import { useActiveWorkspace } from "@/components/auth/workspace-context";
import { FadeIn } from "@/components/motion/fade";

const PAGE_SIZE = 20;

interface LogEntry {
  id: string;
  action: string;
  resource_type: string;
  resource_id: string;
  user_id: string | null;
  user_name: string;
  metadata: Record<string, unknown>;
  created_at: string;
}

interface AuditResponse {
  entries: LogEntry[];
  page: number;
  page_size: number;
  total: number;
}

const ACTION_FILTERS = [
  { value: "", label: "All actions" },
  { value: "auth.login", label: "auth.login" },
  { value: "auth.logout", label: "auth.logout" },
  { value: "auth.register", label: "auth.register" },
  { value: "link.create", label: "link.create" },
  { value: "link.update", label: "link.update" },
  { value: "link.delete", label: "link.delete" },
  { value: "member.role_update", label: "member.role_update" },
  { value: "member.remove", label: "member.remove" },
  { value: "invitation.create", label: "invitation.create" },
  { value: "invitation.accept", label: "invitation.accept" },
  { value: "api_key.create", label: "api_key.create" },
  { value: "api_key.revoke", label: "api_key.revoke" },
  { value: "tenant.create", label: "tenant.create" },
  { value: "tenant.update", label: "tenant.update" },
];

// Look up by resource type: icon + badge color per event family.
const TYPE_STYLE: Record<string, { icon: typeof Link2; badge: string }> = {
  link: {
    icon: Link2,
    badge: "bg-blue-600/10 text-blue-700 dark:text-blue-300",
  },
  member: {
    icon: Users,
    badge: "bg-violet-500/10 text-violet-600 dark:text-violet-400",
  },
  invitation: {
    icon: MailPlus,
    badge: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
  },
  api_key: {
    icon: KeyRound,
    badge: "bg-amber-500/10 text-amber-600 dark:text-amber-400",
  },
  tenant: {
    icon: Building2,
    badge: "bg-sky-500/10 text-sky-600 dark:text-sky-400",
  },
  user: {
    icon: LogIn,
    badge: "bg-zinc-500/10 text-zinc-500 dark:text-zinc-400",
  },
};

const FALLBACK_STYLE = {
  icon: ScrollText,
  badge: "bg-zinc-500/10 text-zinc-500 dark:text-zinc-400",
};

function formatWhen(iso: string): string {
  return new Date(iso).toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export default function AuditPage() {
  const { active } = useActiveWorkspace();
  const tenantId = active?.id ?? null;

  const [action, setAction] = useState("");
  const [page, setPage] = useState(1);

  const { data, isLoading, error } = useQuery({
    queryKey: ["audit", tenantId, action, page],
    queryFn: () => {
      const params = new URLSearchParams({
        page: String(page),
        page_size: String(PAGE_SIZE),
      });
      if (action) params.set("action", action);
      return api<AuditResponse>(
        `/api/v1/tenants/${tenantId}/audit-logs?${params}`,
      );
    },
    enabled: !!tenantId,
  });

  const entries = data?.entries ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <div className="space-y-6">
      <FadeIn>
        <h2 className="text-lg font-semibold tracking-tight text-zinc-900 dark:text-white">
          Audit Log
        </h2>
        <p className="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
          Every important action in {active?.name} — who, what, when.
          Append-only: entries can never be edited or deleted.
        </p>
      </FadeIn>

      <FadeIn delay={0.08}>
        <div className="flex flex-wrap items-center justify-between gap-3">
          <select
            value={action}
            onChange={(e) => {
              setAction(e.target.value);
              setPage(1);
            }}
            className="rounded-lg border border-transparent bg-zinc-100 px-3.5 py-2.5 text-sm text-zinc-900 shadow-sm transition focus:border-blue-500 focus:bg-white focus:outline-none dark:bg-zinc-800/50 dark:text-white dark:focus:bg-zinc-900"
          >
            {ACTION_FILTERS.map((f) => (
              <option key={f.value} value={f.value}>
                {f.label}
              </option>
            ))}
          </select>
          <p className="text-sm text-zinc-500 dark:text-zinc-400">
            {total} event{total === 1 ? "" : "s"}
          </p>
        </div>
      </FadeIn>

      <FadeIn delay={0.16}>
        <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
          {isLoading && (
            <div className="space-y-3 p-5">
              {[0, 1, 2, 3, 4].map((i) => (
                <div
                  key={i}
                  className="h-12 animate-pulse rounded-lg bg-zinc-100 dark:bg-zinc-800/60"
                />
              ))}
            </div>
          )}

          {error && (
            <p className="p-6 text-sm text-rose-600 dark:text-rose-400">
              Failed to load the audit log. Is the backend running?
            </p>
          )}

          {!isLoading && entries.length === 0 && (
            <div className="flex flex-col items-center gap-3 px-6 py-12 text-center">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-zinc-500/10 text-zinc-500 dark:text-zinc-400">
                <ScrollText className="h-6 w-6" />
              </div>
              <p className="font-semibold text-zinc-900 dark:text-white">
                No activity yet
              </p>
              <p className="text-sm text-zinc-500 dark:text-zinc-400">
                Actions in this workspace will appear here.
              </p>
            </div>
          )}

          {entries.map((e) => {
            const style = TYPE_STYLE[e.resource_type] ?? FALLBACK_STYLE;
            const Icon = style.icon;
            return (
              <div
                key={e.id}
                className="flex items-center gap-4 border-b border-zinc-100 px-5 py-3.5 last:border-0 dark:border-zinc-800/60"
              >
                <div
                  className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-lg ${style.badge}`}
                >
                  <Icon className="h-4 w-4" />
                </div>

                <div className="min-w-0 flex-1">
                  <p className="flex flex-wrap items-center gap-2 text-sm">
                    <span
                      className={`rounded-full px-2.5 py-0.5 font-mono text-xs font-semibold ${style.badge}`}
                    >
                      {e.action}
                    </span>
                    <span className="text-zinc-500 dark:text-zinc-400">
                      by{" "}
                      <span className="font-medium text-zinc-700 dark:text-zinc-200">
                        {e.user_name || "system"}
                      </span>
                    </span>
                  </p>
                  {e.resource_id && (
                    <p className="mt-0.5 truncate font-mono text-xs text-zinc-400">
                      {e.resource_type}#{e.resource_id.slice(0, 18)}
                    </p>
                  )}
                </div>

                <p className="shrink-0 text-xs text-zinc-400">
                  {formatWhen(e.created_at)}
                </p>
              </div>
            );
          })}
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
