/**
 * The single entry point for every HTTP call. Components never call fetch
 * directly — keeping it here means auth headers, error shape, and one day a
 * desktop transport are changed in one file rather than forty.
 */

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

export async function apiGet<T>(path: string, signal?: AbortSignal): Promise<T> {
  const response = await fetch(`${BASE}${path}`, {
    signal,
    headers: { Accept: "application/json" },
  });

  if (!response.ok) {
    const body: unknown = await response.json().catch(() => null);
    if (isErrorBody(body)) {
      throw new ApiError(response.status, body.error);
    }
    throw new ApiError(response.status, {
      code: "internal_error",
      message: `Server returned ${String(response.status)}`,
    });
  }

  return (await response.json()) as T;
}
