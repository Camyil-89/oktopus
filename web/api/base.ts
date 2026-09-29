import { notifyUnauthorized } from "./unauthorized";

const API_BASE =
  process.env.NEXT_PUBLIC_API_URL?.replace(/\/$/, "") ??
  "http://localhost:8000";

export { API_BASE };
export class ApiError extends Error {
  status: number;
  diagnostics?: unknown;

  constructor(status: number, message: string, diagnostics?: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.diagnostics = diagnostics;
  }
}

type HttpMethod = "GET" | "POST" | "PATCH" | "PUT" | "DELETE";

type ErrorPayload = { error?: string; diagnostics?: unknown };

function apiErrorFromPayload(
  payload: ErrorPayload | Record<string, unknown>,
  statusText: string,
): ApiError {
  const msg =
    typeof payload.error === "string" ? payload.error : statusText;
  return new ApiError(
    0,
    msg,
    "diagnostics" in payload ? payload.diagnostics : undefined,
  );
}

export async function fetchApi<T>(
  path: string,
  method: HttpMethod,
  body?: unknown,
): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method,
    credentials: "include",
    headers: body ? { "Content-Type": "application/json" } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  });

  if (res.status === 204) {
    return undefined as T;
  }

  const text = await res.text();
  let payload: ErrorPayload | T = {};
  if (text) {
    try {
      payload = JSON.parse(text) as ErrorPayload | T;
    } catch {
      payload = { error: text };
    }
  }

  if (!res.ok) {
    if (res.status === 401) {
      notifyUnauthorized(path);
    }
    const err = apiErrorFromPayload(
      payload as ErrorPayload,
      res.statusText,
    );
    err.status = res.status;
    throw err;
  }

  return payload as T;
}

export async function fetchApiForm<T>(
  path: string,
  method: "POST" | "PUT" | "PATCH",
  form: FormData,
): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method,
    credentials: "include",
    body: form,
  });

  if (res.status === 204) {
    return undefined as T;
  }

  const text = await res.text();
  let payload: ErrorPayload | T = {};
  if (text) {
    try {
      payload = JSON.parse(text) as ErrorPayload | T;
    } catch {
      payload = { error: text };
    }
  }

  if (!res.ok) {
    if (res.status === 401) {
      notifyUnauthorized(path);
    }
    const err = apiErrorFromPayload(
      payload as ErrorPayload,
      res.statusText,
    );
    err.status = res.status;
    throw err;
  }

  return payload as T;
}
