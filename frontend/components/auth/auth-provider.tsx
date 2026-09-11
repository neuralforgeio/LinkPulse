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

// The login now happens in two steps: the password step returns
// { otp_required: true }, and the code step returns the tokens.
interface LoginResponse {
  otp_required?: boolean;
  user?: AuthUser;
  access_token?: string;
  expires_at?: string;
}

type AuthStatus = "loading" | "authenticated" | "guest";

interface AuthContextValue {
  status: AuthStatus;
  user: AuthUser | null;
  tenants: TenantMembership[];
  defaultTenantId: string | null;
  /** Step 1: verify credentials. Returns "otp_required" when a code
      was sent, or "ok" when the session was issued directly. */
  login: (email: string, password: string) => Promise<"otp_required" | "ok">;
  /** Step 2: verify the emailed code and complete the login. */
  verifyOtp: (email: string, code: string) => Promise<void>;
  logout: () => Promise<void>;
  /** Re-fetches /me after profile or workspace changes. */
  reload: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

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

  const loadProfile = useCallback(async () => {
    const me = await api<MeResponse>("/api/v1/auth/me");
    setUser(me.user);
    setTenants(me.tenants);
    setDefaultTenantId(me.default_tenant_id);
  }, []);

  const login = useCallback(
    async (email: string, password: string) => {
      const res = await api<LoginResponse>("/api/v1/auth/login", {
        method: "POST",
        body: JSON.stringify({ email, password }),
      });
      if (res.otp_required) return "otp_required";

      if (res.access_token) {
        setToken(res.access_token, res.expires_at);
        await loadProfile();
        setStatus("authenticated");
      }
      return "ok";
    },
    [loadProfile],
  );

  const verifyOtp = useCallback(
    async (email: string, code: string) => {
      const res = await api<LoginResponse>("/api/v1/auth/login/verify", {
        method: "POST",
        body: JSON.stringify({ email, code }),
      });
      if (!res.access_token) {
        throw new Error("no token issued");
      }
      setToken(res.access_token, res.expires_at);
      await loadProfile();
      setStatus("authenticated");
    },
    [loadProfile],
  );

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
      await loadProfile();
    } catch {
      // Keep the current state on failure — a full reload retries.
    }
  }, [loadProfile]);

  return (
    <AuthContext.Provider
      value={{
        status,
        user,
        tenants,
        defaultTenantId,
        login,
        verifyOtp,
        logout,
        reload,
      }}
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
