"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, Copy, UserPlus, X } from "lucide-react";
import { api, ApiError } from "@/lib/api/client";
import { useAuth } from "@/components/auth/auth-provider";
import { useActiveWorkspace } from "@/components/auth/workspace-context";
import { Button } from "@/components/ui/button";
import { FadeIn } from "@/components/motion/fade";

interface MemberOut {
  user_id: string;
  name: string;
  email: string;
  role: string;
  joined_at: string;
}

interface InviteResult {
  invite_code: string;
  invite_url: string;
  role: string;
  expires_at: string;
}

const INVITE_ROLES = ["member", "viewer", "admin"];

const ROLE_BADGE: Record<string, string> = {
  owner: "bg-blue-600/10 text-blue-700 dark:text-blue-300",
  admin: "bg-violet-500/10 text-violet-600 dark:text-violet-400",
  member: "bg-sky-500/10 text-sky-600 dark:text-sky-400",
  viewer: "bg-zinc-500/10 text-zinc-500 dark:text-zinc-400",
};

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export default function MembersPage() {
  const { user } = useAuth();
  const { active } = useActiveWorkspace();

  const tenantId = active?.id ?? null;
  const myRole = active?.role ?? "viewer";
  const canManage = myRole === "owner" || myRole === "admin";
  const canGrantOwner = myRole === "owner";

  const queryClient = useQueryClient();

  const {
    data: members,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["members", tenantId],
    queryFn: () => api<MemberOut[]>(`/api/v1/tenants/${tenantId}/members`),
    enabled: !!tenantId,
  });

  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: ["members", tenantId] });

  const roleMutation = useMutation({
    mutationFn: (vars: { userId: string; role: string }) =>
      api<MemberOut>(`/api/v1/tenants/${tenantId}/members/${vars.userId}`, {
        method: "PATCH",
        body: JSON.stringify({ role: vars.role }),
      }),
    onSuccess: invalidate,
  });

  const removeMutation = useMutation({
    mutationFn: (userId: string) =>
      api<{ message: string }>(
        `/api/v1/tenants/${tenantId}/members/${userId}`,
        {
          method: "DELETE",
        },
      ),
    onSuccess: invalidate,
  });

  const [inviteRole, setInviteRole] = useState("member");
  const [invite, setInvite] = useState<InviteResult | null>(null);
  const [copied, setCopied] = useState(false);
  const [confirmingId, setConfirmingId] = useState<string | null>(null);

  const inviteMutation = useMutation({
    mutationFn: () =>
      api<InviteResult>(`/api/v1/tenants/${tenantId}/invitations`, {
        method: "POST",
        body: JSON.stringify({ role: inviteRole }),
      }),
    onSuccess: (data) => {
      setInvite(data);
      setCopied(false);
    },
  });

  async function copyInviteLink() {
    if (!invite) return;
    await navigator.clipboard.writeText(
      `${window.location.origin}${invite.invite_url}`,
    );
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  const actionError =
    roleMutation.error ?? removeMutation.error ?? inviteMutation.error;
  const actionErrorMessage =
    actionError instanceof ApiError
      ? actionError.message
      : actionError
        ? "Something went wrong."
        : null;

  const roleOptions = ["owner", "admin", "member", "viewer"].filter(
    (r) => r !== "owner" || canGrantOwner,
  );

  return (
    <div className="space-y-6">
      <FadeIn>
        <h2 className="text-lg font-semibold tracking-tight text-zinc-900 dark:text-white">
          Members
        </h2>
        <p className="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
          Manage who has access to{" "}
          <span className="font-medium text-zinc-700 dark:text-zinc-200">
            {active?.name}
          </span>
          . Your role: <span className="font-medium">{myRole}</span>.
        </p>
      </FadeIn>

      {actionErrorMessage && (
        <p className="rounded-lg bg-rose-50 px-4 py-3 text-sm text-rose-600 dark:bg-rose-500/10 dark:text-rose-400">
          {actionErrorMessage}
        </p>
      )}

      {canManage && (
        <FadeIn delay={0.08}>
          <div className="rounded-xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
            <div className="flex flex-wrap items-center justify-between gap-4">
              <div>
                <h3 className="font-semibold text-zinc-900 dark:text-white">
                  Invite a member
                </h3>
                <p className="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
                  One-time code, valid for 72 hours.
                </p>
              </div>
              <div className="flex items-center gap-3">
                <select
                  value={inviteRole}
                  onChange={(e) => setInviteRole(e.target.value)}
                  className="rounded-lg border border-transparent bg-zinc-100 px-3.5 py-2.5 text-sm text-zinc-900 shadow-sm transition focus:border-blue-500 focus:bg-white focus:outline-none dark:bg-zinc-800/50 dark:text-white dark:focus:bg-zinc-900"
                >
                  {INVITE_ROLES.map((role) => (
                    <option key={role} value={role}>
                      {role}
                    </option>
                  ))}
                </select>
                <Button
                  onClick={() => inviteMutation.mutate()}
                  loading={inviteMutation.isPending}
                >
                  <UserPlus className="h-4 w-4" />
                  Create invitation
                </Button>
              </div>
            </div>

            {invite && (
              <div className="mt-5 rounded-lg bg-blue-600/10 p-5">
                <p className="text-xs font-medium uppercase tracking-wider text-blue-700 dark:text-blue-300">
                  Invitation created — joins as {invite.role}
                </p>
                <p className="mt-1 font-mono text-2xl font-bold tracking-wider text-zinc-900 dark:text-white">
                  {invite.invite_code}
                </p>
                <p className="mt-1 text-xs text-zinc-500">
                  Expires {new Date(invite.expires_at).toLocaleString("en-US")}
                </p>
                <button
                  type="button"
                  onClick={copyInviteLink}
                  className="mt-3 inline-flex items-center gap-2 rounded-lg bg-white px-4 py-2 text-sm font-semibold text-zinc-900 shadow-sm transition hover:bg-zinc-50 dark:bg-zinc-900 dark:text-white dark:hover:bg-zinc-800"
                >
                  {copied ? (
                    <>
                      <Check className="h-4 w-4 text-blue-600" />
                      Copied!
                    </>
                  ) : (
                    <>
                      <Copy className="h-4 w-4" />
                      Copy invite link
                    </>
                  )}
                </button>
              </div>
            )}
          </div>
        </FadeIn>
      )}

      <FadeIn delay={0.16}>
        <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
          {isLoading && (
            <div className="space-y-3 p-5">
              {[0, 1, 2].map((i) => (
                <div
                  key={i}
                  className="h-14 animate-pulse rounded-lg bg-zinc-100 dark:bg-zinc-800/60"
                />
              ))}
            </div>
          )}

          {error && (
            <p className="p-6 text-sm text-rose-600 dark:text-rose-400">
              Failed to load members. Is the backend running?
            </p>
          )}

          {members?.map((m) => {
            const isSelf = m.user_id === user?.id;
            return (
              <div
                key={m.user_id}
                className="flex items-center gap-4 border-b border-zinc-100 px-5 py-4 last:border-0 dark:border-zinc-800/60"
              >
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-blue-600 to-indigo-600 text-sm font-bold text-white">
                  {m.name.charAt(0).toUpperCase()}
                </div>

                <div className="min-w-0 flex-1">
                  <p className="flex items-center gap-2 truncate font-medium text-zinc-900 dark:text-white">
                    {m.name}
                    {isSelf && (
                      <span className="rounded-full bg-zinc-100 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-zinc-500 dark:bg-zinc-800">
                        You
                      </span>
                    )}
                  </p>
                  <p className="truncate text-sm text-zinc-500 dark:text-zinc-400">
                    {m.email}
                  </p>
                </div>

                <p className="hidden shrink-0 text-xs text-zinc-400 sm:block">
                  joined {formatDate(m.joined_at)}
                </p>

                {canManage && !isSelf ? (
                  <select
                    value={m.role}
                    disabled={roleMutation.isPending}
                    onChange={(e) =>
                      roleMutation.mutate({
                        userId: m.user_id,
                        role: e.target.value,
                      })
                    }
                    className="shrink-0 rounded-lg border border-transparent bg-zinc-100 px-2.5 py-1.5 text-sm font-medium text-zinc-700 shadow-sm transition focus:border-blue-500 focus:outline-none dark:bg-zinc-800/50 dark:text-zinc-200"
                  >
                    {roleOptions.map((role) => (
                      <option key={role} value={role}>
                        {role}
                      </option>
                    ))}
                  </select>
                ) : (
                  <span
                    className={`shrink-0 rounded-full px-3 py-1 text-xs font-semibold uppercase tracking-wide ${
                      ROLE_BADGE[m.role] ?? ROLE_BADGE.viewer
                    }`}
                  >
                    {m.role}
                  </span>
                )}

                {canManage && !isSelf && (
                  <div className="shrink-0">
                    {confirmingId === m.user_id ? (
                      <div className="flex items-center gap-2">
                        <button
                          type="button"
                          onClick={() => {
                            removeMutation.mutate(m.user_id);
                            setConfirmingId(null);
                          }}
                          className="rounded-full bg-rose-600 px-3.5 py-1.5 text-xs font-semibold text-white transition hover:bg-rose-500"
                        >
                          Confirm remove
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
                        onClick={() => setConfirmingId(m.user_id)}
                        title="Remove member"
                        className="flex h-8 w-8 items-center justify-center rounded-lg text-zinc-400 transition hover:bg-rose-50 hover:text-rose-600 dark:hover:bg-rose-500/10 dark:hover:text-rose-400"
                      >
                        <X className="h-4 w-4" />
                      </button>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </FadeIn>

      {!canManage && (
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          Only owners and admins can invite or manage members.
        </p>
      )}
    </div>
  );
}
