import type { NameCount } from "./types";

interface TopListProps {
  title: string;
  items: NameCount[];
}

/** A "top X" ranking with proportional bars. */
export function TopList({ title, items }: TopListProps) {
  const max = Math.max(...items.map((i) => i.clicks), 1);

  return (
    <div className="rounded-xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
      <h3 className="text-sm font-semibold text-zinc-900 dark:text-white">
        {title}
      </h3>
      {items.length === 0 ? (
        <p className="mt-3 text-sm text-zinc-400 dark:text-zinc-500">
          No data yet.
        </p>
      ) : (
        <ul className="mt-3 space-y-2.5">
          {items.map((item) => (
            <li key={item.name}>
              <div className="flex items-center justify-between text-sm">
                <span className="truncate text-zinc-600 dark:text-zinc-300">
                  {item.name}
                </span>
                <span className="ml-2 shrink-0 font-medium text-zinc-900 dark:text-white">
                  {item.clicks.toLocaleString("en-US")}
                </span>
              </div>
              <div className="mt-1 h-1.5 rounded-full bg-zinc-100 dark:bg-zinc-800">
                <div
                  className="h-full rounded-full bg-blue-600"
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
