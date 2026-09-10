import {
  clearToken,
  getToken,
  isTokenExpired,
  setToken,
} from "@/lib/auth/token-store";

const API_BASE =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
  ) {
    super(message);
  }
}

interface Envelope<T> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
    details: string[];
  };
}

export interface RequestOptions {
  method?: string;
  body?: string;
  headers?: Record<string, string>;
}

// Single-flight refresh: when several requests hit 401 at once, they all
// share ONE refresh call. This is not an optimization — it is required.
// Our refresh tokens rotate, so two parallel refreshes would make the
// second one replay a revoked token, triggering the backend's reuse
// detection and revoking EVERY session of the user.
let refreshInFlight: Promise<string | null> | null = null;

/**
 * Refreshes the access token via the HttpOnly cookie.
 * Returns the new token, or null when there is no valid session.
 */
export async function refreshAccessToken(): Promise<string | null> {
  if (refreshInFlight) return refreshInFlight;

  refreshInFlight = (async () => {
    try {
      const res = await fetch(`${API_BASE}/api/v1/auth/refresh`, {
        method: "POST",
        credentials: "include",
      });
      const body = (await res.json()) as Envelope<{
        access_token: string;
        expires_at: string;
      }>;
      if (!res.ok || !body.success || !body.data) {
        clearToken();
        return null;
      }
      setToken(body.data.access_token, body.data.expires_at);
      return body.data.access_token;
    } catch {
      // Network error — keep whatever token we already have.
      return null;
    }
  })();

  const token = await refreshInFlight;
  refreshInFlight = null;
  return token;
}

async function request(
  path: string,
  options: RequestOptions,
  token: string | null,
): Promise<Response> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...options.headers,
  };
  if (token) headers.Authorization = `Bearer ${token}`;

  return fetch(`${API_BASE}${path}`, {
    method: options.method ?? "GET",
    body: options.body,
    credentials: "include",
    headers,
  });
}

export async function api<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  // Pre-emptive refresh when the token is (almost) expired.
  let token = getToken();
  if (token && isTokenExpired()) {
    token = await refreshAccessToken();
  }

  let res = await request(path, options, token);

  // One silent refresh + retry on auth failure. Auth endpoints are
  // excluded: their 401s are real answers ("invalid credentials").
  if (res.status === 401 && !path.startsWith("/api/v1/auth/")) {
    const fresh = await refreshAccessToken();
    if (fresh) {
      res = await request(path, options, fresh);
    }
  }

  const body = (await res.json()) as Envelope<T>;

  if (!res.ok || !body.success || body.data === undefined) {
    throw new ApiError(
      res.status,
      body.error?.code ?? "UNKNOWN",
      body.error?.message ?? "request failed",
    );
  }
  return body.data;
}
