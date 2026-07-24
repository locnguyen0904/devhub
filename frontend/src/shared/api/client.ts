/**
 * The single entry point for every HTTP call. Components never call fetch
 * directly — keeping it here means the access token, auth header, error shape,
 * and one day a desktop transport are changed in one file rather than forty.
 */
import type { Session } from "@/shared/types";

/** Error body returned by the API. Mirrors httpx.ErrorBody on the Go side. */
export interface ApiErrorBody {
  code: string;
  message: string;
  fields?: Record<string, string>;
}

/** Thrown for any non-2xx response. Callers switch on `code`, never on message. */
export class ApiError extends Error {
  readonly code: string;
  readonly status: number;
  readonly fields?: Record<string, string>;

  constructor(status: number, body: ApiErrorBody) {
    super(body.message);
    this.name = "ApiError";
    this.status = status;
    this.code = body.code;
    this.fields = body.fields;
  }
}

// Relative path: frontend and API share an origin in both dev and production,
// so there is no base URL to configure. See docs/01-architecture.md §9.
const BASE = "/api/v1";

// The access token lives only in this module's memory — never localStorage,
// which any injected script could read. It is lost on reload and rebuilt from
// the refresh cookie by refreshSession().
let accessToken: string | null = null;

export function getAccessToken(): string | null {
  return accessToken;
}

export function setAccessToken(token: string | null): void {
  accessToken = token;
}

// Coalesces concurrent refreshes: several requests hitting 401 at once must not
// each rotate the refresh token, or all but the first would present a token the
// server just revoked and the whole session would be killed.
let inFlightRefresh: Promise<Session> | null = null;

function isErrorBody(value: unknown): value is { error: ApiErrorBody } {
  if (typeof value !== "object" || value === null || !("error" in value)) {
    return false;
  }
  const { error } = value;
  return (
    typeof error === "object" &&
    error !== null &&
    "code" in error &&
    "message" in error
  );
}

async function toApiError(response: Response): Promise<ApiError> {
  const body: unknown = await response.json().catch(() => null);
  if (isErrorBody(body)) {
    return new ApiError(response.status, body.error);
  }
  return new ApiError(response.status, {
    code: "internal_error",
    message: `Server returned ${String(response.status)}`,
  });
}

/**
 * Exchanges the refresh cookie for a new access token. Concurrent callers share
 * one in-flight request. On success the new token is stored for later calls.
 */
export function refreshSession(): Promise<Session> {
  inFlightRefresh ??= (async () => {
    try {
      const response = await fetch(`${BASE}/auth/refresh`, {
        method: "POST",
        credentials: "same-origin",
        headers: { Accept: "application/json" },
      });
      if (!response.ok) {
        accessToken = null;
        throw await toApiError(response);
      }
      const session = (await response.json()) as Session;
      accessToken = session.access_token;
      return session;
    } finally {
      inFlightRefresh = null;
    }
  })();
  return inFlightRefresh;
}

interface RequestOptions {
  method?: string;
  body?: unknown;
  signal?: AbortSignal;
  // Set on the retry after a refresh, so a genuinely-unauthorised request does
  // not loop refreshing forever.
  retried?: boolean;
}

async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const headers: Record<string, string> = { Accept: "application/json" };
  if (accessToken) headers.Authorization = `Bearer ${accessToken}`;
  if (opts.body !== undefined) headers["Content-Type"] = "application/json";

  const response = await fetch(`${BASE}${path}`, {
    method: opts.method ?? "GET",
    credentials: "same-origin",
    headers,
    body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
    signal: opts.signal,
  });

  // A 401 on an authenticated call usually means the access token expired.
  // Refresh once and retry; if that also fails, surface the error.
  if (response.status === 401 && !opts.retried && accessToken !== null) {
    try {
      await refreshSession();
    } catch {
      throw await toApiError(response);
    }
    return request<T>(path, { ...opts, retried: true });
  }

  if (!response.ok) throw await toApiError(response);

  // Some endpoints (logout) answer with an empty body; parsing "" as JSON
  // throws, so treat an empty response as no value.
  const text = await response.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

export function apiGet<T>(path: string, signal?: AbortSignal): Promise<T> {
  return request<T>(path, { signal });
}

export function apiPost<T>(path: string, body?: unknown): Promise<T> {
  return request<T>(path, { method: "POST", body });
}
