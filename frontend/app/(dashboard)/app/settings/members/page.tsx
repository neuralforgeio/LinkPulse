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
  owner: "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
  admin: "bg-teal-500/10 text-teal-700 dark:text-teal-300",
  member: "bg-sky-500/10 text-sky-700 dark:text-sky-300",
  viewer: "bg-slate-500/10 text-slate-600 dark:text-slate-400",
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
        { method: "DELETE" },
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

  // Owner appears in the dropdown only for owners (least privilege,
  // mirroring the backend rule).
  const roleOptions = ["owner", "admin", "member", "viewer"].filter(
    (r) => r !== "owner" || canGrantOwner,
  );

  return (
    <div className="space-y-6">
      <FadeIn>
        <h2 className="text-lg font-semibold tracking-tight text-slate-900 dark:text-white">
          Members
        </h2>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
          Manage who has access to{" "}
          <span className="font-medium text-slate-700 dark:text-slate-200">
            {active?.name}
          </span>
          . Your role: <span className="font-medium">{myRole}</span>.
        </p>
      </FadeIn>

      {actionErrorMessage && (
        <p className="rounded-xl bg-red-50 px-4 py-3 text-sm text-red-600 dark:bg-red-500/10 dark:text-red-400">
          {actionErrorMessage}
        </p>
      )}

      {canManage && (
        <FadeIn delay={0.08}>
          <div className="rounded-2xl border border-slate-200 bg-white p-6 dark:border-slate-800 dark:bg-slate-900">
            <div className="flex flex-wrap items-center justify-between gap-4">
              <div>
                <h3 className="font-semibold text-slate-900 dark:text-white">
                  Invite a member
                </h3>
                <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
                  One-time code, valid for 72 hours.
                </p>
              </div>
              <div className="flex items-center gap-3">
                <select
                  value={inviteRole}
                  onChange={(e) => setInviteRole(e.target.value)}
                  className="rounded-xl border border-transparent bg-slate-100 px-3.5 py-2.5 text-sm text-slate-900 shadow-sm transition focus:border-emerald-500 focus:bg-white focus:outline-none dark:bg-slate-900 dark:text-white dark:focus:bg-slate-950"
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
              <div className="mt-5 rounded-2xl bg-emerald-500/10 p-5">
                <p className="text-xs font-medium uppercase tracking-wider text-emerald-700 dark:text-emerald-300">
                  Invitation created — joins as {invite.role}
                </p>
                <p className="mt-1 font-mono text-2xl font-bold tracking-wider text-slate-900 dark:text-white">
                  {invite.invite_code}
                </p>
                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                  Expires {new Date(invite.expires_at).toLocaleString("en-US")}
                </p>
                <button
                  type="button"
                  onClick={copyInviteLink}
                  className="mt-3 inline-flex items-center gap-2 rounded-full bg-white px-4 py-2 text-sm font-semibold text-slate-900 shadow-sm transition hover:bg-slate-100 dark:bg-slate-950 dark:text-white dark:hover:bg-slate-900"
                >
                  {copied ? (
                    <>
                      <Check className="h-4 w-4 text-emerald-500" />
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

      {/* Roster */}
      <FadeIn delay={0.16}>
        <div className="overflow-hidden rounded-2xl border border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900">
          {isLoading && (
            <div className="space-y-3 p-5">
              {[0, 1, 2].map((i) => (
                <div
                  key={i}
                  className="h-14 animate-pulse rounded-xl bg-slate-100 dark:bg-slate-800/60"
                />
              ))}
            </div>
          )}

          {error && (
            <p className="p-6 text-sm text-red-600 dark:text-red-400">
              Failed to load members. Is the backend running?
            </p>
          )}

          {members?.map((m) => {
            const isSelf = m.user_id === user?.id;
            return (
              <div
                key={m.user_id}
                className="flex items-center gap-4 border-b border-slate-100 px-5 py-4 last:border-0 dark:border-slate-800/60"
              >
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-emerald-400 to-teal-500 text-sm font-bold text-slate-950">
                  {m.name.charAt(0).toUpperCase()}
                </div>

                <div className="min-w-0 flex-1">
                  <p className="flex items-center gap-2 truncate font-medium text-slate-900 dark:text-white">
                    {m.name}
                    {isSelf && (
                      <span className="rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-slate-500 dark:bg-slate-800">
                        You
                      </span>
                    )}
                  </p>
                  <p className="truncate text-sm text-slate-500 dark:text-slate-400">
                    {m.email}
                  </p>
                </div>

                <p className="hidden shrink-0 text-xs text-slate-400 sm:block">
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
                    className="shrink-0 rounded-lg border border-transparent bg-slate-100 px-2.5 py-1.5 text-sm font-medium text-slate-700 shadow-sm transition focus:border-emerald-500 focus:outline-none dark:bg-slate-900 dark:text-slate-200"
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
                          className="rounded-full bg-red-500 px-3.5 py-1.5 text-xs font-semibold text-white transition hover:bg-red-400"
                        >
                          Confirm remove
                        </button>
                        <button
                          type="button"
                          onClick={() => setConfirmingId(null)}
                          className="text-xs text-slate-500 hover:text-slate-700 dark:hover:text-slate-300"
                        >
                          Cancel
                        </button>
                      </div>
                    ) : (
                      <button
                        type="button"
                        onClick={() => setConfirmingId(m.user_id)}
                        title="Remove member"
                        className="flex h-8 w-8 items-center justify-center rounded-lg text-slate-400 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-500/10 dark:hover:text-red-400"
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
        <p className="text-sm text-slate-500 dark:text-slate-400">
          Only owners and admins can invite or manage members.
        </p>
      )}
    </div>
  );
}
