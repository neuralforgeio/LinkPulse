import type { NameCount } from "./types";

interface TopListProps {
  title: string;
  items: NameCount[];
}

/** A "top X" ranking with proportional bars. */
export function TopList({ title, items }: TopListProps) {
  const max = Math.max(...items.map((i) => i.clicks), 1);

  return (
    <div className="rounded-2xl border border-slate-200 bg-white p-5 dark:border-slate-800 dark:bg-slate-900">
      <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
        {title}
      </h3>
      {items.length === 0 ? (
        <p className="mt-3 text-sm text-slate-400 dark:text-slate-500">
          No data yet.
        </p>
      ) : (
        <ul className="mt-3 space-y-2.5">
          {items.map((item) => (
            <li key={item.name}>
              <div className="flex items-center justify-between text-sm">
                <span className="truncate text-slate-600 dark:text-slate-300">
                  {item.name}
                </span>
                <span className="ml-2 shrink-0 font-medium text-slate-900 dark:text-white">
                  {item.clicks.toLocaleString("en-US")}
                </span>
              </div>
              <div className="mt-1 h-1.5 rounded-full bg-slate-100 dark:bg-slate-800">
                <div
                  className="h-full rounded-full bg-gradient-to-r from-emerald-500 to-teal-400"
                  style={{
                    width: `${Math.max(4, (item.clicks / max) * 100)}%`,
                  }}
                />
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
