import type { TranslateFn } from "@/i18n/translate";

export const MANAGE_NAV_PATHS = [
  "/manage",
  "/manage/rules",
  "/manage/access-log",
  "/manage/access-log/reports",
  "/manage/proxy",
  "/manage/users",
] as const;

export type ManageNavPath = (typeof MANAGE_NAV_PATHS)[number];

const NAV_TITLE_KEYS: Record<ManageNavPath, Parameters<TranslateFn>[0]> = {
  "/manage": "nav.home",
  "/manage/users": "nav.users",
  "/manage/proxy": "nav.proxySettings",
  "/manage/rules": "nav.proxyRules",
  "/manage/access-log": "nav.accessLog",
  "/manage/access-log/reports": "nav.accessLogReports",
};

export function manageNavLabel(path: ManageNavPath, t: TranslateFn): string {
  return t(NAV_TITLE_KEYS[path]);
}

export function managePageTitle(pathname: string, t: TranslateFn): string {
  if (pathname === "/manage/users" || pathname.startsWith("/manage/users/")) {
    return t("nav.users");
  }
  if (pathname === "/manage/rules" || pathname.startsWith("/manage/rules/")) {
    return t("nav.proxyRules");
  }
  if (pathname === "/manage/proxy" || pathname.startsWith("/manage/proxy/")) {
    return t("nav.proxySettings");
  }
  if (
    pathname === "/manage/access-log/reports" ||
    pathname.startsWith("/manage/access-log/reports/")
  ) {
    return t("nav.accessLogReports");
  }
  if (pathname === "/manage/access-log") {
    return t("nav.accessLog");
  }
  return t("nav.home");
}
