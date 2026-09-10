"use client";

import { createContext, useContext, useEffect, useState } from "react";
import { useAuth, type TenantMembership } from "@/components/auth/auth-provider";

interface WorkspaceContextValue {
  active: TenantMembership | null;
  tenants: TenantMembership[];
  setActive: (tenantId: string) => void;
}

const WorkspaceContext = createContext<WorkspaceContextValue | null>(null);

function storageKey(userId: string): string {
  return `linkpulse.workspace.${userId}`;
}

/**
 * The workspace the dashboard currently operates on. The choice is
 * persisted per user in localStorage (a preference, not a secret) and
 * falls back to the account's default workspace. Every tenant-scoped
 * page (members now, links and analytics later) reads from here.
 */
export function ActiveWorkspaceProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const { status, user, tenants, defaultTenantId } = useAuth();
  const [activeId, setActiveId] = useState<string | null>(null);

  useEffect(() => {
    if (status !== "authenticated" || !user) return;

    const saved = window.localStorage.getItem(storageKey(user.id));
    const valid = saved && tenants.some((t) => t.id === saved) ? saved : null;
    const fallback = defaultTenantId ?? tenants[0]?.id ?? null;
    const chosen = valid ?? fallback;

    setActiveId(chosen);
    if (chosen) {
      window.localStorage.setItem(storageKey(user.id), chosen);
    }
  }, [status, user, tenants, defaultTenantId]);

  function setActive(tenantId: string) {
    setActiveId(tenantId);
    if (user) {
      window.localStorage.setItem(storageKey(user.id), tenantId);
    }
  }

  const active = tenants.find((t) => t.id === activeId) ?? null;

  return (
    <WorkspaceContext.Provider value={{ active, tenants, setActive }}>
      {children}
    </WorkspaceContext.Provider>
  );
}

export function useActiveWorkspace(): WorkspaceContextValue {
  const ctx = useContext(WorkspaceContext);
  if (!ctx) {
    throw new Error(
      "useActiveWorkspace must be used inside <ActiveWorkspaceProvider>",
    );
  }
  return ctx;
}
