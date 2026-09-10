const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080"

export class ApiError extends Error {
  constructor (
    public readonly status: number,
    public readonly code: string,
    message: string
  ) {
    super(message)
  }
}

interface Envelope<T> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
    details: string[];
  }
}

export interface RequestOptions {
  method?: string;
  body?: string;
  headers?: Record<string, string>
}

export async function api<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: options.method ?? "GET",
    body: options.body,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...options.headers
    }
  })

  const body = (await res.json() as Envelope<T>)

  if (!res.ok || !body.success || body.data === undefined) {
    throw new ApiError(
      res.status,
      body.error?.code ?? "UNKNOWN",
      body.error?.message ?? "request failed"
    )
  }

  return body.data;
}
