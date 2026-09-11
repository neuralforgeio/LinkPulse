"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";
import { api, refreshAccessToken } from "@/lib/api/client";
import { clearToken, setToken } from "@/lib/auth/token-store";

export interface AuthUser {
  id: string;
  name: string;
  email: string;
}

export interface TenantMembership {
  id: string;
  name: string;
  slug: string;
  role: string;
}

interface MeResponse {
  user: AuthUser;
  tenants: TenantMembership[];
  default_tenant_id: string | null;
}

interface LoginResponse {
  user: AuthUser;
  access_token: string;
  token_type: string;
  expires_at: string;
}

type AuthStatus = "loading" | "authenticated" | "guest";

interface AuthContextValue {
  status: AuthStatus;
  user: AuthUser | null;
  tenants: TenantMembership[];
  defaultTenantId: string | null;
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  /** Re-fetches /me — call after profile or workspace changes so the
      sidebar and switcher reflect them immediately. */
  reload: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

/**
 * Owns the client-side session: the in-memory access token (held by
 * token-store) plus the user profile. On mount it recovers the session
 * from the HttpOnly refresh cookie via a silent refresh.
 */
export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>("loading");
  const [user, setUser] = useState<AuthUser | null>(null);
  const [tenants, setTenants] = useState<TenantMembership[]>([]);
  const [defaultTenantId, setDefaultTenantId] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    (async () => {
      const token = await refreshAccessToken();
      if (!token) {
        if (!cancelled) setStatus("guest");
        return;
      }
      try {
        const me = await api<MeResponse>("/api/v1/auth/me");
        if (cancelled) return;
        setUser(me.user);
        setTenants(me.tenants);
        setDefaultTenantId(me.default_tenant_id);
        setStatus("authenticated");
      } catch {
        if (!cancelled) setStatus("guest");
      }
    })();

    return () => {
      cancelled = true;
    };
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    const res = await api<LoginResponse>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    });
    setToken(res.access_token, res.expires_at);

    const me = await api<MeResponse>("/api/v1/auth/me");
    setUser(me.user);
    setTenants(me.tenants);
    setDefaultTenantId(me.default_tenant_id);
    setStatus("authenticated");
  }, []);

  const logout = useCallback(async () => {
    try {
      await api("/api/v1/auth/logout", { method: "POST" });
    } catch {
      // Even if the call fails, drop the local session.
    }
    clearToken();
    setUser(null);
    setTenants([]);
    setDefaultTenantId(null);
    setStatus("guest");
  }, []);

  const reload = useCallback(async () => {
    try {
      const me = await api<MeResponse>("/api/v1/auth/me");
      setUser(me.user);
      setTenants(me.tenants);
      setDefaultTenantId(me.default_tenant_id);
    } catch {
      // Keep the current state on failure — a full reload retries.
    }
  }, []);

  return (
    <AuthContext.Provider
      value={{ status, user, tenants, defaultTenantId, login, logout, reload }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used inside <AuthProvider>");
  }
  return ctx;
}
