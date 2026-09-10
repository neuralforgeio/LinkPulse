import type { LucideIcon } from "lucide-react";

interface StatCardProps {
  label: string;
  value: string;
  icon: LucideIcon;
}

export function StatCard({ label, value, icon: Icon }: StatCardProps) {
  return (
    <div className="rounded-xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
      <div className="flex items-center justify-between">
        <p className="text-sm text-zinc-500 dark:text-zinc-400">{label}</p>
        <Icon className="h-4 w-4 text-zinc-300 dark:text-zinc-600" />
      </div>
      <p className="mt-2 text-3xl font-bold tracking-tight text-zinc-900 dark:text-white">
        {value}
      </p>
    </div>
  );
}
