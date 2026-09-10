import Link from "next/link";

const TABS = [
  { label: "Members", href: "/app/settings/members", enabled: true },
  { label: "General", href: null, enabled: false },
  { label: "Profile", href: null, enabled: false },
  { label: "API Keys", href: null, enabled: false },
];

export default function SettingsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div>
      <h1 className="text-2xl font-bold tracking-tight text-zinc-900 dark:text-white">
        Settings
      </h1>
      <nav className="mt-4 flex gap-2 overflow-x-auto pb-1">
        {TABS.map((tab) =>
          tab.enabled ? (
            <Link
              key={tab.label}
              href={tab.href!}
              className="shrink-0 rounded-lg bg-zinc-900 px-4 py-1.5 text-sm font-medium text-white transition hover:bg-zinc-800 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
            >
              {tab.label}
            </Link>
          ) : (
            <span
              key={tab.label}
              className="flex shrink-0 cursor-not-allowed items-center gap-1.5 rounded-lg border border-zinc-200 px-4 py-1.5 text-sm font-medium text-zinc-400 opacity-60 dark:border-zinc-800 dark:text-zinc-500"
            >
              {tab.label}
              <span className="rounded-full bg-zinc-100 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide dark:bg-zinc-800">
                Soon
              </span>
            </span>
          ),
        )}
      </nav>
      <div className="mt-6">{children}</div>
    </div>
  );
}
