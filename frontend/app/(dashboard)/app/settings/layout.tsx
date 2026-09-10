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
      <h1 className="text-2xl font-bold tracking-tight text-slate-900 dark:text-white">
        Settings
      </h1>
      <nav className="mt-4 flex gap-2 overflow-x-auto pb-1">
        {TABS.map((tab) =>
          tab.enabled ? (
            <Link
              key={tab.label}
              href={tab.href!}
              className="shrink-0 rounded-full bg-emerald-500/10 px-4 py-1.5 text-sm font-medium text-emerald-700 transition hover:bg-emerald-500/20 dark:text-emerald-300"
            >
              {tab.label}
            </Link>
          ) : (
            <span
              key={tab.label}
              className="flex shrink-0 cursor-not-allowed items-center gap-1.5 rounded-full border border-slate-200 px-4 py-1.5 text-sm font-medium text-slate-400 opacity-60 dark:border-slate-800 dark:text-slate-500"
            >
              {tab.label}
              <span className="rounded-full bg-slate-100 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide dark:bg-slate-800">
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
