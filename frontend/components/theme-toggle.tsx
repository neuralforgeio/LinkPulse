"use client";

import { useEffect, useState } from "react";
import { useTheme } from "next-themes";
import { Monitor, Moon, Sun } from "lucide-react";

const ORDER = ["light", "dark", "system"] as const;

/** Cycles light → dark → system on click. */
export function ThemeToggle() {
  const [mounted, setMounted] = useState(false);
  const { theme, setTheme } = useTheme();

  // next-themes resolves the theme only on the client; render a stable
  // placeholder until mounted to avoid hydration mismatch.
  useEffect(() => setMounted(true), []);

  if (!mounted) {
    return <div className="h-9 w-9" aria-hidden="true" />;
  }

  const current = (ORDER as readonly string[]).includes(theme ?? "")
    ? (theme as (typeof ORDER)[number])
    : "system";
  const next = ORDER[(ORDER.indexOf(current) + 1) % ORDER.length];

  return (
    <button
      type="button"
      onClick={() => setTheme(next)}
      title={`Theme: ${current} — click for ${next}`}
      className="flex h-9 w-9 items-center justify-center rounded-full border border-slate-200 text-slate-500 transition hover:border-slate-300 hover:text-slate-900 dark:border-slate-800 dark:text-slate-400 dark:hover:border-slate-600 dark:hover:text-white"
    >
      {current === "dark" ? (
        <Moon className="h-4 w-4" />
      ) : current === "light" ? (
        <Sun className="h-4 w-4" />
      ) : (
        <Monitor className="h-4 w-4" />
      )}
    </button>
  );
}
