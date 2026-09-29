export const MANAGE_PAGE_TITLES: Record<string, string> = {
  "/manage": "Главная",
  "/manage/users": "Пользователи",
  "/manage/proxy": "Настройки прокси",
  "/manage/rules": "Правила прокси",
  "/manage/access-log": "Журнал доступа",
  "/manage/access-log/reports": "Отчёты журнала",
};

export function managePageTitle(pathname: string): string {
  if (pathname === "/manage/users" || pathname.startsWith("/manage/users/")) {
    return MANAGE_PAGE_TITLES["/manage/users"];
  }
  if (pathname === "/manage/rules" || pathname.startsWith("/manage/rules/")) {
    return MANAGE_PAGE_TITLES["/manage/rules"];
  }
  if (pathname === "/manage/proxy" || pathname.startsWith("/manage/proxy/")) {
    return MANAGE_PAGE_TITLES["/manage/proxy"];
  }
  if (
    pathname === "/manage/access-log/reports" ||
    pathname.startsWith("/manage/access-log/reports/")
  ) {
    return MANAGE_PAGE_TITLES["/manage/access-log/reports"];
  }
  if (pathname === "/manage/access-log") {
    return MANAGE_PAGE_TITLES["/manage/access-log"];
  }
  return MANAGE_PAGE_TITLES["/manage"];
}
