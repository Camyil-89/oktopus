"use client";

import {
  ApiOutlined,
  BarChartOutlined,
  HomeOutlined,
  SafetyCertificateOutlined,
  TeamOutlined,
  UnorderedListOutlined,
} from "@ant-design/icons";
import Link from "next/link";
import { usePathname } from "next/navigation";
import type { AuthUser } from "@/types/auth";
import { OktopusIcon } from "@/assets/components/oktopus/OktopusIcon";
import { MANAGE_PAGE_TITLES } from "@/assets/components/manageNav";

const navItems = [
  { href: "/manage", icon: HomeOutlined, label: "Главная" },
  {
    href: "/manage/rules",
    icon: SafetyCertificateOutlined,
    label: "Правила ACL",
  },
  {
    href: "/manage/access-log",
    icon: UnorderedListOutlined,
    label: "Журнал доступа",
  },
  {
    href: "/manage/access-log/reports",
    icon: BarChartOutlined,
    label: "Отчёты журнала",
  },
  {
    href: "/manage/proxy",
    icon: ApiOutlined,
    label: MANAGE_PAGE_TITLES["/manage/proxy"],
  },
  { href: "/manage/users", icon: TeamOutlined, label: "Пользователи" },
] as const;

function selectedHref(pathname: string): string {
  if (pathname === "/manage/users" || pathname.startsWith("/manage/users/")) {
    return "/manage/users";
  }
  if (pathname === "/manage/rules" || pathname.startsWith("/manage/rules/")) {
    return "/manage/rules";
  }
  if (pathname === "/manage/proxy" || pathname.startsWith("/manage/proxy/")) {
    return "/manage/proxy";
  }
  if (
    pathname === "/manage/access-log/reports" ||
    pathname.startsWith("/manage/access-log/reports/")
  ) {
    return "/manage/access-log/reports";
  }
  if (pathname === "/manage/access-log") {
    return "/manage/access-log";
  }
  return "/manage";
}

type ManageSidebarProps = {
  user: AuthUser | null;
  onLogout: () => void;
};

export function ManageSidebar({ user, onLogout }: ManageSidebarProps) {
  const pathname = usePathname();
  const active = selectedHref(pathname);

  return (
    <aside className="flex w-[236px] shrink-0 flex-col border-r border-white/5 bg-ink/50">
      <div className="flex h-14 items-center gap-2.5 border-b border-white/5 px-5">
        <span className="grid h-7 w-7 place-items-center rounded-md border border-teal-400/25 bg-teal-400/10">
          <OktopusIcon size="sm" title="Oktopus" />
        </span>
        <span className="text-sm font-medium tracking-tight text-zinc-200">
          Oktopus
        </span>
      </div>

      <nav className="flex flex-1 flex-col gap-0.5 px-3 py-4">
        {navItems.map(({ href, icon: Icon, label }) => {
          const isActive = active === href;
          return (
            <Link
              key={href}
              href={href}
              className={`nav-item ${isActive ? "nav-item-active" : ""}`}
            >
              <Icon
                className={`nav-item-icon text-[17px] ${isActive ? "text-teal-400" : ""}`}
              />
              {label}
            </Link>
          );
        })}
      </nav>

      <div className="border-t border-white/5 p-3">
        <button
          type="button"
          onClick={onLogout}
          className="flex w-full cursor-pointer items-center gap-2.5 rounded-lg px-2 py-1.5 transition hover:bg-white/[0.03]"
        >
          <div className="grid h-7 w-7 place-items-center rounded-full border border-white/8 bg-white/5 font-mono text-[11px] text-zinc-400">
            {(user?.username ?? "?").slice(0, 1).toUpperCase()}
          </div>
          <div className="min-w-0 text-left">
            <p className="truncate text-[12.5px] leading-tight text-zinc-300">
              {user?.username ?? "—"}
            </p>
            <p className="truncate font-mono text-[10.5px] text-zinc-500">
              панель управления
            </p>
          </div>
          <LogoutIcon className="ml-auto h-4 w-4 text-zinc-500" />
        </button>
      </div>
    </aside>
  );
}

function LogoutIcon({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      fill="none"
      stroke="currentColor"
      strokeWidth={1.6}
      viewBox="0 0 24 24"
      aria-hidden
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M15.75 9V5.25A2.25 2.25 0 0 0 13.5 3h-6a2.25 2.25 0 0 0-2.25 2.25v13.5A2.25 2.25 0 0 0 7.5 21h6a2.25 2.25 0 0 0 2.25-2.25V15m3 0 3-3m0 0-3-3m3 3H9"
      />
    </svg>
  );
}
