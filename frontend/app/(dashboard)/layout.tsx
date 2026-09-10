"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import {
  BarChart3,
  Building2,
  Check,
  ChevronDown,
  LayoutDashboard,
  Link2,
  LogOut,
  Settings,
} from "lucide-react";
import { Logo } from "@/components/brand/logo";
import { ThemeToggle } from "@/components/theme-toggle";
import { useAuth } from "@/components/auth/auth-provider";
import {
  ActiveWorkspaceProvider,
  useActiveWorkspace,
} from "@/components/auth/workspace-context";

const NAV = [
  {
    href: "/app/overview",
    label: "Overview",
    icon: LayoutDashboard,
    enabled: true,
  },
  { href: "/app/links", label: "Links", icon: Link2, enabled: true },
  {
    href: "/app/analytics",
    label: "Analytics",
    icon: BarChart3,
    enabled: true,
  },
  {
    href: "/app/settings/members",
    label: "Settings",
    icon: Settings,
    enabled: true,
  },
];

/** Dropdown to switch the active workspace. */
function WorkspaceSwitcher() {
  const { active, tenants, setActive } = useActiveWorkspace();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function onDocClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", onDocClick);
    return () => document.removeEventListener("mousedown", onDocClick);
  }, []);

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="inline-flex items-center gap-2 rounded-lg border border-zinc-200 bg-white px-3.5 py-1.5 text-sm font-medium text-zinc-700 transition hover:border-zinc-300 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-200 dark:hover:border-zinc-600"
      >
        <Building2 className="h-4 w-4 text-blue-600 dark:text-blue-400" />
        <span className="max-w-[10rem] truncate">
          {active?.name ?? "Workspace"}
        </span>
        <ChevronDown className="h-3.5 w-3.5 text-zinc-400" />
      </button>

      {open && (
        <div className="absolute left-0 z-10 mt-2 w-64 overflow-hidden rounded-xl border border-zinc-200 bg-white p-1.5 shadow-xl dark:border-zinc-800 dark:bg-zinc-900">
          {tenants.map((t) => (
            <button
              key={t.id}
              type="button"
              onClick={() => {
                setActive(t.id);
                setOpen(false);
              }}
              className={`flex w-full items-center justify-between rounded-lg px-3 py-2.5 text-left text-sm transition ${
                t.id === active?.id
                  ? "bg-zinc-100 text-zinc-900 dark:bg-zinc-800 dark:text-white"
                  : "text-zinc-600 hover:bg-zinc-100 dark:text-zinc-300 dark:hover:bg-zinc-800/60"
              }`}
            >
              <span className="min-w-0">
                <span className="block truncate font-medium">{t.name}</span>
                <span className="block text-xs text-zinc-400">{t.role}</span>
              </span>
              {t.id === active?.id && (
                <Check className="h-4 w-4 shrink-0 text-blue-600 dark:text-blue-400" />
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <ActiveWorkspaceProvider>
      <DashboardShell>{children}</DashboardShell>
    </ActiveWorkspaceProvider>
  );
}

function DashboardShell({ children }: { children: React.ReactNode }) {
  const { status, user, logout } = useAuth();
  const { active } = useActiveWorkspace();
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    if (status === "guest") {
      router.replace("/login");
    }
  }, [status, router]);

  if (status !== "authenticated") {
    return (
      <div className="flex min-h-dvh items-center justify-center bg-zinc-50 dark:bg-zinc-950">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-zinc-300 border-t-blue-600 dark:border-zinc-700" />
      </div>
    );
  }

  return (
    <div className="flex min-h-dvh bg-zinc-50 dark:bg-zinc-950">
      {/* Sidebar */}
      <aside className="hidden w-60 shrink-0 flex-col border-r border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900 lg:flex">
        <div className="p-5">
          <Logo />
        </div>

        <nav className="flex-1 space-y-1 px-3">
          {NAV.map((item) => {
            const isActive = pathname.startsWith(item.href);
            const base = `flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition ${
              isActive
                ? "bg-zinc-100 text-zinc-900 dark:bg-zinc-800 dark:text-white"
                : "text-zinc-500 hover:bg-zinc-100 hover:text-zinc-900 dark:text-zinc-400 dark:hover:bg-zinc-800/60 dark:hover:text-white"
            }`;
            const iconCls = `h-4 w-4 ${isActive ? "text-blue-600 dark:text-blue-400" : ""}`;

            if (!item.enabled) {
              return (
                <span
                  key={item.href}
                  className={`${base} cursor-not-allowed opacity-50`}
                  aria-disabled="true"
                >
                  <item.icon className={iconCls} />
                  {item.label}
                  <span className="ml-auto rounded-full bg-zinc-100 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-zinc-500 dark:bg-zinc-800">
                    Soon
                  </span>
                </span>
              );
            }

            return (
              <Link key={item.href} href={item.href} className={base}>
                <item.icon className={iconCls} />
                {item.label}
              </Link>
            );
          })}
        </nav>

        {/* User card */}
        <div className="border-t border-zinc-200 p-4 dark:border-zinc-800">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-blue-600 to-indigo-600 text-sm font-bold text-white">
              {(user?.name ?? "?").charAt(0).toUpperCase()}
            </div>
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-semibold text-zinc-900 dark:text-white">
                {user?.name}
              </p>
              <p className="truncate text-xs text-zinc-500 dark:text-zinc-400">
                {active?.name ?? "Workspace"}
              </p>
            </div>
            <button
              type="button"
              onClick={() => logout()}
              title="Log out"
              className="flex h-8 w-8 items-center justify-center rounded-lg text-zinc-400 transition hover:bg-rose-50 hover:text-rose-600 dark:hover:bg-rose-500/10 dark:hover:text-rose-400"
            >
              <LogOut className="h-4 w-4" />
            </button>
          </div>
        </div>
      </aside>

      {/* Main area */}
      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex items-center justify-between gap-3 border-b border-zinc-200 bg-white px-4 py-3 dark:border-zinc-800 dark:bg-zinc-900 sm:px-6">
          <div className="flex items-center gap-3">
            <div className="lg:hidden">
              <Logo />
            </div>
            <WorkspaceSwitcher />
          </div>
          <div className="flex items-center gap-3">
            <ThemeToggle />
            <button
              type="button"
              onClick={() => logout()}
              className="flex h-9 w-9 items-center justify-center rounded-lg border border-zinc-200 text-zinc-500 transition hover:border-rose-300 hover:text-rose-600 dark:border-zinc-800 dark:text-zinc-400 lg:hidden"
              title="Log out"
            >
              <LogOut className="h-4 w-4" />
            </button>
          </div>
        </header>

        <main className="flex-1 p-4 sm:p-6 lg:p-8">{children}</main>
      </div>
    </div>
  );
}
