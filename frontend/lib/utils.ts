// utils.ts — shared formatting helpers.

/** formatDate renders an ISO timestamp as "MMM D, YYYY" (en-US). */
export function formatDate(iso: string | null | undefined): string {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

/** formatWhen renders an ISO timestamp as "MMM D, YYYY, H:MM AM/PM". */
export function formatWhen(iso: string | null | undefined): string {
  if (!iso) return "—";
  return new Date(iso).toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

/** fmt renders a count with en-US thousands separators. */
export function fmt(n: number | null | undefined): string {
  return (n ?? 0).toLocaleString("en-US");
}

/** STATUS_BADGE maps link statuses to Tailwind badge classes. */
export const STATUS_BADGE: Record<string, string> = {
  active: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
  expired: "bg-amber-500/10 text-amber-600 dark:text-amber-400",
  disabled: "bg-zinc-500/10 text-zinc-600 dark:text-zinc-400",
  max_clicks_reached: "bg-rose-500/10 text-rose-600 dark:text-rose-400",
};
