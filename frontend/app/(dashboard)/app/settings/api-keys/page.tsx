"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, Copy, KeyRound, Plus, X } from "lucide-react";
import { api, ApiError } from "@/lib/api/client";
import { useActiveWorkspace } from "@/components/auth/workspace-context";
import { Button } from "@/components/ui/button";
import { FadeIn } from "@/components/motion/fade";

interface ApiKeyOut {
  id: string;
  name: string;
  key_prefix: string;
  scopes: string[];
  last_used_at: string | null;
  revoked_at: string | null;
  created_at: string;
}

interface CreateKeyResult {
  id: string;
  name: string;
  key: string; // shown exactly once
  key_prefix: string;
  scopes: string[];
  created_at: string;
}

const ALL_SCOPES = ["links:read", "links:write", "analytics:read"];

function formatDate(iso: string | null): string {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export default function ApiKeysPage() {
  const { active } = useActiveWorkspace();
  const tenantId = active?.id ?? null;
  const myRole = active?.role ?? "viewer";
  const canManage = myRole === "owner" || myRole === "admin";

  const queryClient = useQueryClient();

  const {
    data: keys,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["apikeys", tenantId],
    queryFn: () => api<ApiKeyOut[]>(`/api/v1/tenants/${tenantId}/api-keys`),
    enabled: !!tenantId && canManage,
  });

  const [name, setName] = useState("");
  const [scopes, setScopes] = useState<string[]>(["links:read", "links:write"]);
  const [created, setCreated] = useState<CreateKeyResult | null>(null);
  const [copied, setCopied] = useState(false);
  const [confirmingId, setConfirmingId] = useState<string | null>(null);

  const createMutation = useMutation({
    mutationFn: () =>
      api<CreateKeyResult>(`/api/v1/tenants/${tenantId}/api-keys`, {
        method: "POST",
        body: JSON.stringify({ name: name.trim(), scopes }),
      }),
    onSuccess: (data) => {
      setCreated(data);
      setCopied(false);
      setName("");
      queryClient.invalidateQueries({ queryKey: ["apikeys", tenantId] });
    },
  });

  const revokeMutation = useMutation({
    mutationFn: (keyId: string) =>
      api<{ message: string }>(
        `/api/v1/tenants/${tenantId}/api-keys/${keyId}`,
        {
          method: "DELETE",
        },
      ),
    onSuccess: () => {
      setConfirmingId(null);
      queryClient.invalidateQueries({ queryKey: ["apikeys", tenantId] });
    },
  });

  async function copyKey() {
    if (!created) return;
    await navigator.clipboard.writeText(created.key);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  function toggleScope(scope: string) {
    setScopes((prev) =>
      prev.includes(scope) ? prev.filter((s) => s !== scope) : [...prev, scope],
    );
  }

  const actionError = createMutation.error ?? revokeMutation.error;
  const actionErrorMessage =
    actionError instanceof ApiError
      ? actionError.message
      : actionError
        ? "Something went wrong."
        : null;

  if (!canManage) {
    return (
      <div className="rounded-xl border border-zinc-200 bg-white p-8 text-center dark:border-zinc-800 dark:bg-zinc-900">
        <h2 className="text-lg font-semibold text-zinc-900 dark:text-white">
          Restricted
        </h2>
        <p className="mt-2 text-sm text-zinc-500 dark:text-zinc-400">
          Only owners and admins can manage API keys for {active?.name}.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <FadeIn>
        <h2 className="text-lg font-semibold tracking-tight text-zinc-900 dark:text-white">
          API Keys
        </h2>
        <p className="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
          Let your tools create and manage links via the public REST API.
        </p>
      </FadeIn>

      {actionErrorMessage && (
        <p className="rounded-lg bg-rose-50 px-4 py-3 text-sm text-rose-600 dark:bg-rose-500/10 dark:text-rose-400">
          {actionErrorMessage}
        </p>
      )}

      {/* Create + one-time key reveal */}
      <FadeIn delay={0.08}>
        <div className="rounded-xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
          <div className="flex flex-wrap items-end justify-between gap-4">
            <div className="min-w-[14rem] flex-1">
              <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300">
                Key name
              </label>
              <input
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Mobile App Integration"
                className="mt-1.5 w-full rounded-lg border border-transparent bg-zinc-100 px-4 py-2.5 text-sm text-zinc-900 shadow-sm transition placeholder:text-zinc-400 focus:border-blue-500 focus:bg-white focus:outline-none dark:bg-zinc-800/50 dark:text-white dark:focus:bg-zinc-900"
              />
            </div>
            <Button
              onClick={() => createMutation.mutate()}
              loading={createMutation.isPending}
              disabled={scopes.length === 0}
            >
              <Plus className="h-4 w-4" />
              Create key
            </Button>
          </div>

          <div className="mt-4 flex flex-wrap gap-2">
            {ALL_SCOPES.map((sc) => (
              <button
                key={sc}
                type="button"
                onClick={() => toggleScope(sc)}
                className={`rounded-full px-3.5 py-1.5 font-mono text-xs transition ${
                  scopes.includes(sc)
                    ? "bg-blue-600/10 text-blue-700 ring-1 ring-blue-600/30 dark:text-blue-300"
                    : "border border-zinc-200 text-zinc-400 hover:text-zinc-600 dark:border-zinc-800"
                }`}
              >
                {sc}
              </button>
            ))}
          </div>

          {created && (
            <div className="mt-5 rounded-lg bg-blue-600/10 p-5">
              <p className="text-xs font-medium uppercase tracking-wider text-blue-700 dark:text-blue-300">
                Copy this key now — it will never be shown again
              </p>
              <div className="mt-2 flex flex-wrap items-center gap-3">
                <code className="break-all rounded-lg bg-white px-3 py-2 font-mono text-sm font-semibold text-zinc-900 dark:bg-zinc-900 dark:text-white">
                  {created.key}
                </code>
                <button
                  type="button"
                  onClick={copyKey}
                  className="inline-flex items-center gap-2 rounded-lg bg-white px-4 py-2 text-sm font-semibold text-zinc-900 shadow-sm transition hover:bg-zinc-50 dark:bg-zinc-900 dark:text-white dark:hover:bg-zinc-800"
                >
                  {copied ? (
                    <>
                      <Check className="h-4 w-4 text-blue-600" />
                      Copied!
                    </>
                  ) : (
                    <>
                      <Copy className="h-4 w-4" />
                      Copy
                    </>
                  )}
                </button>
              </div>
            </div>
          )}
        </div>
      </FadeIn>

      {/* Key list */}
      <FadeIn delay={0.16}>
        <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
          {isLoading && (
            <div className="space-y-3 p-5">
              {[0, 1].map((i) => (
                <div
                  key={i}
                  className="h-14 animate-pulse rounded-lg bg-zinc-100 dark:bg-zinc-800/60"
                />
              ))}
            </div>
          )}

          {error && (
            <p className="p-6 text-sm text-rose-600 dark:text-rose-400">
              Failed to load API keys. Is the backend running?
            </p>
          )}

          {!isLoading && keys && keys.length === 0 && (
            <div className="flex flex-col items-center gap-3 px-6 py-12 text-center">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-blue-600/10 text-blue-600 dark:text-blue-400">
                <KeyRound className="h-6 w-6" />
              </div>
              <p className="font-semibold text-zinc-900 dark:text-white">
                No API keys yet
              </p>
              <p className="text-sm text-zinc-500 dark:text-zinc-400">
                Create one above to start using the public API.
              </p>
            </div>
          )}

          {keys?.map((k) => (
            <div
              key={k.id}
              className="flex items-center gap-4 border-b border-zinc-100 px-5 py-4 last:border-0 dark:border-zinc-800/60"
            >
              <div className="min-w-0 flex-1">
                <p className="truncate font-medium text-zinc-900 dark:text-white">
                  {k.name}
                </p>
                <p className="truncate font-mono text-xs text-zinc-500 dark:text-zinc-400">
                  {k.key_prefix}••••••••
                </p>
              </div>

              <div className="hidden shrink-0 flex-wrap gap-1.5 sm:flex">
                {k.scopes.map((sc) => (
                  <span
                    key={sc}
                    className="rounded-full bg-zinc-100 px-2 py-0.5 font-mono text-[10px] text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400"
                  >
                    {sc}
                  </span>
                ))}
              </div>

              <p className="hidden shrink-0 text-xs text-zinc-400 lg:block">
                used {formatDate(k.last_used_at)}
              </p>

              {k.revoked_at ? (
                <span className="shrink-0 rounded-full bg-rose-500/10 px-2.5 py-1 text-[10px] font-semibold uppercase tracking-wide text-rose-600 dark:text-rose-400">
                  revoked
                </span>
              ) : (
                <div className="shrink-0">
                  {confirmingId === k.id ? (
                    <div className="flex items-center gap-2">
                      <button
                        type="button"
                        onClick={() => revokeMutation.mutate(k.id)}
                        className="rounded-full bg-rose-600 px-3.5 py-1.5 text-xs font-semibold text-white transition hover:bg-rose-500"
                      >
                        Revoke
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
                      onClick={() => setConfirmingId(k.id)}
                      title="Revoke key"
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
    </div>
  );
}
