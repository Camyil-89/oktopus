type UnauthorizedHandler = () => void;

let handler: UnauthorizedHandler | null = null;
let redirecting = false;

export function setUnauthorizedHandler(next: UnauthorizedHandler | null) {
  handler = next;
  redirecting = false;
}

/** Вызывается из fetchApi при 401 (кроме POST /api/auth/login). */
export function notifyUnauthorized(apiPath: string) {
  if (apiPath.includes("/api/auth/login")) {
    return;
  }
  if (typeof window === "undefined") {
    return;
  }
  if (window.location.pathname.startsWith("/login")) {
    return;
  }
  if (redirecting) {
    return;
  }
  redirecting = true;
  handler?.();
}
