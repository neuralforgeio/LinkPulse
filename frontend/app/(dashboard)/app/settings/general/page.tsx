"use client";

import { useState, type FormEvent } from "react";
import { useMutation } from "@tanstack/react-query";
import { Check } from "lucide-react";
import { api, ApiError } from "@/lib/api/client";
import { useAuth } from "@/components/auth/auth-provider";
import { useActiveWorkspace } from "@/components/auth/workspace-context";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { FadeIn } from "@/components/motion/fade";

interface TenantDetail {
  id: string;
  name: string;
  slug: string;
}

export default function GeneralPage() {
  const { active } = useActiveWorkspace();
  const { reload } = useAuth();
  const tenantId = active?.id ?? null;
  const myRole = active?.role ?? "viewer";
  const canManage = myRole === "owner" || myRole === "admin";

  const [name, setName] = useState(active?.name ?? "");
  const [slug, setSlug] = useState(active?.slug ?? "");
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const saveMutation = useMutation({
    mutationFn: () =>
      api<TenantDetail>(`/api/v1/tenants/${tenantId}`, {
        method: "PATCH",
        body: JSON.stringify({
          name: name.trim(),
          slug: slug.trim().toLowerCase(),
        }),
      }),
    onSuccess: () => {
      setSaved(true);
      setError(null);
      reload(); // sidebar + workspace switcher update immediately
      setTimeout(() => setSaved(false), 2500);
    },
    onError: (err) =>
      setError(err instanceof ApiError ? err.message : "Something went wrong."),
  });

  if (!canManage) {
    return (
      <div className="rounded-xl border border-zinc-200 bg-white p-8 text-center dark:border-zinc-800 dark:bg-zinc-900">
        <h2 className="text-lg font-semibold text-zinc-900 dark:text-white">
          Restricted
        </h2>
        <p className="mt-2 text-sm text-zinc-500 dark:text-zinc-400">
          Only owners and admins can change workspace settings for{" "}
          {active?.name}.
        </p>
      </div>
    );
  }

  function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setSaved(false);

    if (name.trim().length < 2) {
      setError("Workspace name must be at least 2 characters.");
      return;
    }
    const s = slug.trim().toLowerCase();
    if (!/^[a-z0-9]+(-[a-z0-9]+)*$/.test(s) || s.length < 3 || s.length > 48) {
      setError("Slug must be 3-48 lowercase letters, digits, and dashes.");
      return;
    }

    setError(null);
    saveMutation.mutate();
  }

  return (
    <div className="max-w-xl">
      <FadeIn>
        <h2 className="text-lg font-semibold tracking-tight text-zinc-900 dark:text-white">
          General
        </h2>
        <p className="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
          Workspace identity for {active?.name}.
        </p>
      </FadeIn>

      <FadeIn delay={0.08}>
        <form
          onSubmit={onSubmit}
          noValidate
          className="mt-6 space-y-5 rounded-xl border border-zinc-200 bg-white p-6 sm:p-8 dark:border-zinc-800 dark:bg-zinc-900"
        >
          <Input
            label="Workspace name"
            placeholder="Marketing Squad"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <Input
            label="Slug"
            placeholder="marketing-squad"
            hint="Lowercase letters, digits, dashes. Must be globally unique."
            value={slug}
            onChange={(e) => setSlug(e.target.value.toLowerCase())}
          />

          {error && (
            <p className="rounded-lg bg-rose-50 px-4 py-3 text-sm text-rose-600 dark:bg-rose-500/10 dark:text-rose-400">
              {error}
            </p>
          )}
          {saved && (
            <p className="flex items-center gap-2 rounded-lg bg-emerald-500/10 px-4 py-3 text-sm text-emerald-600 dark:text-emerald-400">
              <Check className="h-4 w-4" />
              Workspace saved — the sidebar updated too.
            </p>
          )}

          <Button type="submit" loading={saveMutation.isPending}>
            Save changes
          </Button>
        </form>
      </FadeIn>
    </div>
  );
}
