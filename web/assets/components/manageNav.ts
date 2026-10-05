import type { TranslateFn } from "@/i18n/translate";

export const MANAGE_NAV_PATHS = [
  "/manage",
  "/manage/access-log",
  "/manage/reports",
  "/manage/users",
] as const;

export type ManageNavPath = (typeof MANAGE_NAV_PATHS)[number];

const NAV_TITLE_KEYS: Record<ManageNavPath, Parameters<TranslateFn>[0]> = {
  "/manage": "nav.home",
  "/manage/users": "nav.users",
  "/manage/access-log": "nav.accessLog",
  "/manage/reports": "nav.accessLogReports",
};

export function manageNavLabel(path: ManageNavPath, t: TranslateFn): string {
  return t(NAV_TITLE_KEYS[path]);
}

export function managePageTitle(pathname: string, t: TranslateFn): string {
  if (pathname === "/manage/users" || pathname.startsWith("/manage/users/")) {
    return t("nav.users");
  }
  const inst = pathname.match(/^\/manage\/instances\/([^/]+)(\/(.*))?$/);
  if (inst) {
    const sub = inst[2] ?? "";
    if (sub.startsWith("/rules")) return t("nav.proxyRules");
    if (sub.startsWith("/settings")) return t("nav.proxySettings");
    if (sub.startsWith("/access-log")) return t("nav.accessLog");
    return t("nav.instanceStats");
  }
  if (
    pathname === "/manage/reports" ||
    pathname.startsWith("/manage/reports/")
  ) {
    return t("nav.accessLogReports");
  }
  if (pathname === "/manage/access-log") {
    return t("nav.accessLog");
  }
  return t("nav.home");
}
