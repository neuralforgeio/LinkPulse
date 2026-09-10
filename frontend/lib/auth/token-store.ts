// In-memory access token store. Deliberately NOT persisted: an XSS bug
// cannot steal what is never written to disk. After a page reload the
// session is recovered by silently refreshing — the refresh token lives
// in an HttpOnly cookie the browser keeps for us.

let accessToken: string | null = null;
let expiresAtMs = 0;

export function setToken(token: string, expiresAtISO?: string): void {
  accessToken = token;
  expiresAtMs = expiresAtISO ? Date.parse(expiresAtISO) : 0;
}

export function getToken(): string | null {
  return accessToken;
}

/** True when no token exists or it expires within 30s (clock-skew margin). */
export function isTokenExpired(): boolean {
  if (!accessToken) return true;
  if (!expiresAtMs) return false;
  return Date.now() >= expiresAtMs - 30_000;
}

export function clearToken(): void {
  accessToken = null;
  expiresAtMs = 0;
}
