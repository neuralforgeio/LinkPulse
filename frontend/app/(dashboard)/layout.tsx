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
  { href: "/app/links", label: "Links", icon: Link2, enabled: false },
  {
    href: "/app/analytics",
    label: "Analytics",
    icon: BarChart3,
    enabled: false,
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

  // Close on any click outside the dropdown.
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
        className="inline-flex items-center gap-2 rounded-full border border-slate-200 bg-white px-3.5 py-1.5 text-sm font-medium text-slate-700 transition hover:border-slate-300 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-slate-600"
      >
        <Building2 className="h-4 w-4 text-emerald-500" />
        <span className="max-w-[10rem] truncate">
          {active?.name ?? "Workspace"}
        </span>
        <ChevronDown className="h-3.5 w-3.5 text-slate-400" />
      </button>

      {open && (
        <div className="absolute left-0 z-10 mt-2 w-64 overflow-hidden rounded-2xl border border-slate-200 bg-white p-1.5 shadow-xl dark:border-slate-800 dark:bg-slate-900">
          {tenants.map((t) => (
            <button
              key={t.id}
              type="button"
              onClick={() => {
                setActive(t.id);
                setOpen(false);
              }}
              className={`flex w-full items-center justify-between rounded-xl px-3 py-2.5 text-left text-sm transition ${
                t.id === active?.id
                  ? "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300"
                  : "text-slate-600 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800/60"
              }`}
            >
              <span className="min-w-0">
                <span className="block truncate font-medium">{t.name}</span>
                <span className="block text-xs text-slate-400">{t.role}</span>
              </span>
              {t.id === active?.id && <Check className="h-4 w-4 shrink-0" />}
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
  // The provider must wrap the shell so the shell itself (sidebar,
  // header) can read the active workspace.
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

  // Guard: no session, no dashboard.
  useEffect(() => {
    if (status === "guest") {
      router.replace("/login");
    }
  }, [status, router]);

  if (status !== "authenticated") {
    return (
      <div className="flex min-h-dvh items-center justify-center bg-slate-50 dark:bg-slate-950">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-slate-300 border-t-emerald-500 dark:border-slate-700" />
      </div>
    );
  }

  return (
    <div className="flex min-h-dvh bg-slate-50 dark:bg-slate-950">
      {/* Sidebar */}
      <aside className="hidden w-64 shrink-0 flex-col border-r border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900 lg:flex">
        <div className="p-5">
          <Logo />
        </div>

        <nav className="flex-1 space-y-1 px-3">
          {NAV.map((item) => {
            const activeItem = pathname.startsWith(item.href);
            const base = `flex items-center gap-3 rounded-xl px-3.5 py-2.5 text-sm font-medium transition ${
              activeItem
                ? "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300"
                : "text-slate-500 hover:bg-slate-100 hover:text-slate-900 dark:text-slate-400 dark:hover:bg-slate-800/60 dark:hover:text-white"
            }`;

            if (!item.enabled) {
              return (
                <span
                  key={item.href}
                  className={`${base} cursor-not-allowed opacity-50`}
                  aria-disabled="true"
                >
                  <item.icon className="h-4 w-4" />
                  {item.label}
                  <span className="ml-auto rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-slate-500 dark:bg-slate-800">
                    Soon
                  </span>
                </span>
              );
            }

            return (
              <Link key={item.href} href={item.href} className={base}>
                <item.icon className="h-4 w-4" />
                {item.label}
              </Link>
            );
          })}
        </nav>

        {/* User card */}
        <div className="border-t border-slate-200 p-4 dark:border-slate-800">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-emerald-400 to-teal-500 text-sm font-bold text-slate-950">
              {(user?.name ?? "?").charAt(0).toUpperCase()}
            </div>
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-semibold text-slate-900 dark:text-white">
                {user?.name}
              </p>
              <p className="truncate text-xs text-slate-500 dark:text-slate-400">
                {active?.name ?? "Workspace"}
              </p>
            </div>
            <button
              type="button"
              onClick={() => logout()}
              title="Log out"
              className="flex h-8 w-8 items-center justify-center rounded-lg text-slate-400 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-500/10 dark:hover:text-red-400"
            >
              <LogOut className="h-4 w-4" />
            </button>
          </div>
        </div>
      </aside>

      {/* Main area */}
      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex items-center justify-between gap-3 border-b border-slate-200 bg-white px-4 py-3 dark:border-slate-800 dark:bg-slate-900 sm:px-6">
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
              className="flex h-9 w-9 items-center justify-center rounded-full border border-slate-200 text-slate-500 transition hover:text-red-600 dark:border-slate-800 dark:text-slate-400 lg:hidden"
              title="Log out"
            >
              <LogOut className="h-4 w-4" />
            </button>
          </div>
        </header>

        <main className="flex-1 p-6 lg:p-8">{children}</main>
      </div>
    </div>
  );
}
