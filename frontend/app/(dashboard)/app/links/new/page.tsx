"use client";

import { useState, type FormEvent } from "react";
import Link from "next/link";
import QRCode from "qrcode";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { motion } from "framer-motion";
import { BarChart3, Check, Copy, Download } from "lucide-react";
import { api, ApiError } from "@/lib/api/client";
import { useActiveWorkspace } from "@/components/auth/workspace-context";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { FadeIn } from "@/components/motion/fade";
import { QrCode } from "@/components/links/qr-code";

interface LinkOut {
  id: string;
  short_code: string;
  short_url: string;
  destination_url: string;
  title: string;
  status: string;
  tags: string[];
}

interface CreateBody {
  destination_url: string;
  title?: string;
  custom_code?: string;
  expires_at?: string;
  max_clicks?: number;
  password?: string;
  tags?: string[];
  utm?: {
    source: string;
    medium: string;
    campaign: string;
    term: string;
    content: string;
  };
}

const EMPTY_UTM = {
  source: "",
  medium: "",
  campaign: "",
  term: "",
  content: "",
};

export default function NewLinkPage() {
  const { active } = useActiveWorkspace();
  const tenantId = active?.id ?? null;
  const myRole = active?.role ?? "viewer";
  const canManage =
    myRole === "owner" || myRole === "admin" || myRole === "member";

  const queryClient = useQueryClient();

  const [destination, setDestination] = useState("");
  const [title, setTitle] = useState("");
  const [customCode, setCustomCode] = useState("");
  const [expiresAt, setExpiresAt] = useState("");
  const [maxClicks, setMaxClicks] = useState("");
  const [password, setPassword] = useState("");
  const [tagsInput, setTagsInput] = useState("");
  const [showUtm, setShowUtm] = useState(false);
  const [utm, setUtm] = useState(EMPTY_UTM);

  const [errors, setErrors] = useState<string[]>([]);
  const [shakeKey, setShakeKey] = useState(0);
  const [created, setCreated] = useState<LinkOut | null>(null);
  const [copied, setCopied] = useState(false);

  const createMutation = useMutation({
    mutationFn: (body: CreateBody) =>
      api<LinkOut>(`/api/v1/tenants/${tenantId}/links`, {
        method: "POST",
        body: JSON.stringify(body),
      }),
    onSuccess: (data) => {
      setCreated(data);
      setCopied(false);
      queryClient.invalidateQueries({ queryKey: ["links", tenantId] });
    },
    onError: (err) => {
      if (err instanceof ApiError) {
        setErrors([err.message]);
      } else {
        setErrors([
          "Cannot reach the server. Is the backend running on :8080?",
        ]);
      }
      setShakeKey((k) => k + 1);
    },
  });

  // Viewers are read-only — the backend would reject them anyway, but a
  // clear message beats a form that fights back.
  if (!canManage) {
    return (
      <div className="mx-auto max-w-md rounded-2xl border border-slate-200 bg-white p-8 text-center dark:border-slate-800 dark:bg-slate-900">
        <h1 className="text-lg font-semibold text-slate-900 dark:text-white">
          Read-only access
        </h1>
        <p className="mt-2 text-sm text-slate-500 dark:text-slate-400">
          Your role in {active?.name} is{" "}
          <span className="font-medium">{myRole}</span> — only members and above
          can create links.
        </p>
        <Link
          href="/app/links"
          className="mt-5 inline-block text-sm font-medium text-emerald-600 hover:text-emerald-500 dark:text-emerald-400"
        >
          Back to links
        </Link>
      </div>
    );
  }

  async function copyShortUrl() {
    if (!created) return;
    await navigator.clipboard.writeText(created.short_url);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  async function downloadQr() {
    if (!created) return;
    const dataUrl = await QRCode.toDataURL(created.short_url, {
      width: 512,
      margin: 2,
    });
    const a = document.createElement("a");
    a.href = dataUrl;
    a.download = `linkpulse-${created.short_code}.png`;
    a.click();
  }

  // Mirrors the backend rules (PRD 9.4.1) for instant feedback.
  function validate(): string[] {
    const problems: string[] = [];
    const dest = destination.trim();
    if (!dest) {
      problems.push("Destination URL is required.");
    } else if (!/^https?:\/\//.test(dest)) {
      problems.push("Destination URL must start with http:// or https://.");
    }
    if (customCode.trim() && !/^[a-zA-Z0-9_-]{3,32}$/.test(customCode.trim())) {
      problems.push(
        "Custom alias must be 3-32 characters: letters, digits, dash, underscore.",
      );
    }
    if (password && password.length < 4) {
      problems.push("Link password must be at least 4 characters.");
    }
    if (
      maxClicks.trim() &&
      (isNaN(Number(maxClicks)) || Number(maxClicks) <= 0)
    ) {
      problems.push("Max clicks must be a positive number.");
    }
    if (expiresAt && new Date(expiresAt).getTime() <= Date.now()) {
      problems.push("Expiry must be in the future.");
    }
    return problems;
  }

  function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const problems = validate();
    if (problems.length > 0) {
      setErrors(problems);
      setShakeKey((k) => k + 1);
      return;
    }

    const body: CreateBody = {
      destination_url: destination.trim(),
      title: title.trim(),
    };
    if (customCode.trim()) body.custom_code = customCode.trim();
    if (expiresAt) body.expires_at = new Date(expiresAt).toISOString();
    if (maxClicks.trim()) body.max_clicks = Number(maxClicks.trim());
    if (password) body.password = password;
    const tags = tagsInput
      .split(",")
      .map((t) => t.trim().toLowerCase())
      .filter(Boolean);
    if (tags.length > 0) body.tags = tags;
    if (utm.source || utm.medium || utm.campaign || utm.term || utm.content) {
      body.utm = utm;
    }

    setErrors([]);
    createMutation.mutate(body);
  }

  function resetForm() {
    setDestination("");
    setTitle("");
    setCustomCode("");
    setExpiresAt("");
    setMaxClicks("");
    setPassword("");
    setTagsInput("");
    setShowUtm(false);
    setUtm(EMPTY_UTM);
    setErrors([]);
    setCreated(null);
  }

  if (created) {
    return (
      <div className="mx-auto max-w-md">
        <motion.div
          initial={{ opacity: 0, scale: 0.92, y: 24 }}
          animate={{ opacity: 1, scale: 1, y: 0 }}
          transition={{ type: "spring", stiffness: 260, damping: 22 }}
          className="rounded-3xl border border-slate-200 bg-white p-8 text-center shadow-xl dark:border-slate-800 dark:bg-slate-900"
        >
          <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-emerald-100 text-emerald-600 dark:bg-emerald-500/15 dark:text-emerald-400">
            <svg
              viewBox="0 0 24 24"
              className="h-8 w-8"
              fill="none"
              stroke="currentColor"
              strokeWidth={2.5}
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <motion.path
                d="M5 13l4 4L19 7"
                initial={{ pathLength: 0 }}
                animate={{ pathLength: 1 }}
                transition={{ duration: 0.45, delay: 0.25 }}
              />
            </svg>
          </div>
          <h1 className="mt-5 text-2xl font-bold tracking-tight text-slate-900 dark:text-white">
            Link created
          </h1>
          <p className="mt-2 text-sm text-slate-500 dark:text-slate-400">
            {created.title || "Your link"} is live. Share it anywhere — every
            click gets counted.
          </p>

          <div className="mt-5 rounded-2xl bg-slate-100 p-4 dark:bg-slate-950/60">
            <p className="break-all font-mono text-lg font-bold text-emerald-600 dark:text-emerald-400">
              {created.short_url}
            </p>
            <button
              type="button"
              onClick={copyShortUrl}
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
                  Copy URL
                </>
              )}
            </button>
          </div>

          <div className="mt-5 flex flex-col items-center gap-3">
            <QrCode value={created.short_url} size={180} />
            <button
              type="button"
              onClick={downloadQr}
              className="inline-flex items-center gap-2 text-sm font-medium text-emerald-600 transition hover:text-emerald-500 dark:text-emerald-400"
            >
              <Download className="h-4 w-4" />
              Download QR (PNG)
            </button>
          </div>

          <div className="mt-6 flex gap-3">
            <Button onClick={resetForm} className="flex-1">
              Create another
            </Button>
            <Link
              href="/app/links"
              className="flex flex-1 items-center justify-center rounded-full border border-slate-200 px-4 py-2.5 text-sm font-semibold text-slate-600 transition hover:border-slate-300 dark:border-slate-700 dark:text-slate-300"
            >
              View all links
            </Link>
          </div>
        </motion.div>
      </div>
    );
  }

  return (
    <div className="max-w-2xl">
      <FadeIn>
        <div className="flex items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-slate-900 dark:text-white">
              Create a link
            </h1>
            <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
              Shorten a URL for {active?.name}.
            </p>
          </div>
          <Link
            href="/app/links"
            className="text-sm font-medium text-slate-500 transition hover:text-slate-900 dark:text-slate-400 dark:hover:text-white"
          >
            ← Back to links
          </Link>
        </div>
      </FadeIn>

      <FadeIn delay={0.08}>
        <form
          onSubmit={onSubmit}
          noValidate
          className="mt-6 space-y-5 rounded-3xl border border-slate-200 bg-white p-6 sm:p-8 dark:border-slate-800 dark:bg-slate-900"
        >
          <Input
            label="Destination URL"
            type="url"
            placeholder="https://example.com/very/long/path"
            value={destination}
            onChange={(e) => setDestination(e.target.value)}
          />

          <div className="grid gap-5 sm:grid-cols-2">
            <Input
              label="Title"
              placeholder="Campaign Page"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
            />
            <Input
              label="Custom alias"
              placeholder="promo2026"
              hint="3-32 characters — leave empty for a random code."
              value={customCode}
              onChange={(e) => setCustomCode(e.target.value)}
            />
          </div>

          {/* UTM builder */}
          <div>
            <button
              type="button"
              onClick={() => setShowUtm((v) => !v)}
              className="inline-flex items-center gap-2 text-sm font-medium text-emerald-600 transition hover:text-emerald-500 dark:text-emerald-400"
            >
              <BarChart3 className="h-4 w-4" />
              {showUtm ? "Hide UTM tracking" : "Add UTM tracking"}
            </button>
            {showUtm && (
              <div className="mt-4 grid gap-4 rounded-2xl border border-dashed border-slate-300 p-4 dark:border-slate-700 sm:grid-cols-2">
                <Input
                  label="utm_source"
                  placeholder="instagram"
                  value={utm.source}
                  onChange={(e) => setUtm({ ...utm, source: e.target.value })}
                />
                <Input
                  label="utm_medium"
                  placeholder="social"
                  value={utm.medium}
                  onChange={(e) => setUtm({ ...utm, medium: e.target.value })}
                />
                <Input
                  label="utm_campaign"
                  placeholder="september-promo"
                  value={utm.campaign}
                  onChange={(e) => setUtm({ ...utm, campaign: e.target.value })}
                />
                <Input
                  label="utm_term"
                  placeholder="optional"
                  value={utm.term}
                  onChange={(e) => setUtm({ ...utm, term: e.target.value })}
                />
                <Input
                  label="utm_content"
                  placeholder="bio-link"
                  value={utm.content}
                  onChange={(e) => setUtm({ ...utm, content: e.target.value })}
                />
                <p className="self-end text-xs text-slate-500 dark:text-slate-400">
                  Merged into the destination URL — explicit values overwrite
                  existing ones.
                </p>
              </div>
            )}
          </div>

          <div className="grid gap-5 sm:grid-cols-3">
            <Input
              label="Expires at"
              type="datetime-local"
              value={expiresAt}
              onChange={(e) => setExpiresAt(e.target.value)}
            />
            <Input
              label="Max clicks"
              type="number"
              placeholder="1000"
              hint="Leave empty for unlimited."
              value={maxClicks}
              onChange={(e) => setMaxClicks(e.target.value)}
            />
            <Input
              label="Password"
              type="password"
              placeholder="&bull;&bull;&bull;&bull;"
              hint="Optional protection."
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>

          <Input
            label="Tags"
            placeholder="campaign, instagram"
            hint="Comma-separated, up to 10."
            value={tagsInput}
            onChange={(e) => setTagsInput(e.target.value)}
          />

          {errors.length > 0 && (
            <motion.div
              key={shakeKey}
              initial={{ opacity: 0 }}
              animate={{ opacity: 1, x: [0, -10, 10, -6, 6, -2, 0] }}
              transition={{ duration: 0.4 }}
            >
              <div className="rounded-xl bg-red-50 px-4 py-3 text-sm text-red-600 dark:bg-red-500/10 dark:text-red-400">
                <ul className="list-disc space-y-1 pl-4">
                  {errors.map((problem) => (
                    <li key={problem}>{problem}</li>
                  ))}
                </ul>
              </div>
            </motion.div>
          )}

          <Button
            type="submit"
            loading={createMutation.isPending}
            className="w-full py-3"
          >
            Create link
          </Button>
        </form>
      </FadeIn>
    </div>
  );
}
