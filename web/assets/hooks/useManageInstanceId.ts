"use client";

import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";

/** Static export pre-renders only this segment; nginx serves that shell for every instance. */
const STATIC_EXPORT_INSTANCE_ID = "_";

function instanceIdFromPath(pathname: string): string {
  const segment = pathname.match(/^\/manage\/instances\/([^/]+)/)?.[1];
  if (!segment || segment === STATIC_EXPORT_INSTANCE_ID) {
    return "";
  }
  try {
    return decodeURIComponent(segment);
  } catch {
    return segment;
  }
}

/**
 * Instance id from the browser URL.
 * `useParams().id` stays `_` on the static shell nginx rewrites to, so API calls must not use it.
 * Returns "" until the real id is known (first paint of the export shell).
 */
export function useManageInstanceId(): string {
  const pathname = usePathname() || "";
  const fromRouter = instanceIdFromPath(pathname);
  const [id, setId] = useState(fromRouter);

  useEffect(() => {
    const live = instanceIdFromPath(window.location.pathname);
    setId(live || fromRouter);
  }, [fromRouter, pathname]);

  return fromRouter || id;
}
