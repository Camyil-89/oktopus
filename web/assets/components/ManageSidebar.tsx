"use client";

import {
  BarChartOutlined,
  HomeOutlined,
  LineChartOutlined,
  SafetyCertificateOutlined,
  SettingOutlined,
  TeamOutlined,
  UnorderedListOutlined,
} from "@ant-design/icons";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";
import type { AuthUser } from "@/types/auth";
import * as proxyApi from "@/api/proxy";
import type { ProxyInstance } from "@/types/proxy";
import { OktopusIcon } from "@/assets/components/oktopus/OktopusIcon";
import { manageNavLabel } from "@/assets/components/manageNav";
import { AppVersion } from "@/assets/components/AppVersion";
import { PROXY_INSTANCES_CHANGED_EVENT } from "@/assets/modals/CreateProxyInstanceModal";
import { useTranslation } from "@/contexts/LocaleContext";

type ManageSidebarProps = {
  user: AuthUser | null;
  onLogout: () => void;
};

export function ManageSidebar({ user, onLogout }: ManageSidebarProps) {
  const pathname = usePathname();
  const { t } = useTranslation();
  const [instances, setInstances] = useState<ProxyInstance[]>([]);

  useEffect(() => {
    const reload = () => {
      void proxyApi.listProxyInstances().then(setInstances).catch(() => setInstances([]));
    };
    reload();
    window.addEventListener(PROXY_INSTANCES_CHANGED_EVENT, reload);
    return () => window.removeEventListener(PROXY_INSTANCES_CHANGED_EVENT, reload);
  }, []);

  const globalItems = [
    { href: "/manage", icon: HomeOutlined, label: manageNavLabel("/manage", t) },
    { href: "/manage/access-log", icon: UnorderedListOutlined, label: manageNavLabel("/manage/access-log", t) },
    { href: "/manage/reports", icon: BarChartOutlined, label: manageNavLabel("/manage/reports", t) },
    { href: "/manage/users", icon: TeamOutlined, label: manageNavLabel("/manage/users", t) },
  ] as const;

  return (
    <aside className="flex w-[236px] shrink-0 flex-col border-r border-white/5 bg-ink/50">
      <div className="flex h-14 items-center gap-2.5 border-b border-white/5 px-5">
        <span className="grid h-7 w-7 place-items-center rounded-md border border-teal-400/25 bg-teal-400/10">
          <OktopusIcon size="sm" title="Oktopus" />
        </span>
        <span className="text-sm font-medium tracking-tight text-zinc-200">Oktopus</span>
      </div>

      <nav className="flex flex-1 flex-col gap-0.5 overflow-y-auto px-3 py-4">
        {globalItems.map(({ href, icon: Icon, label }) => {
          const isActive = pathname === href || (href !== "/manage" && pathname.startsWith(href));
          return (
            <Link key={href} href={href} className={`nav-item ${isActive ? "nav-item-active" : ""}`}>
              <Icon className={`nav-item-icon text-[17px] ${isActive ? "text-teal-400" : ""}`} />
              {label}
            </Link>
          );
        })}

        {instances.map((inst) => {
          const base = `/manage/instances/${inst.id}`;
          const inInstance = pathname === base || pathname.startsWith(`${base}/`);
          const subNavActive = (href: string) =>
            href === base
              ? pathname === base
              : pathname === href || pathname.startsWith(`${href}/`);
          return (
            <div key={inst.id} className="mt-3 flex flex-col gap-0.5">
              <Link
                href={base}
                className={`nav-item text-[13px] font-medium ${inInstance ? "nav-item-active" : ""}`}
              >
                {inst.name}
              </Link>
              {inInstance ? (
                <div className="ml-3 flex flex-col gap-0.5 border-l border-white/10 pl-2">
                  {[
                    { href: base, icon: LineChartOutlined, label: t("nav.instanceStats") },
                    { href: `${base}/rules`, icon: SafetyCertificateOutlined, label: t("nav.proxyRules") },
                    { href: `${base}/settings`, icon: SettingOutlined, label: t("nav.proxySettings") },
                    { href: `${base}/access-log`, icon: UnorderedListOutlined, label: t("nav.accessLog") },
                  ].map(({ href, icon: Icon, label }) => (
                    <Link
                      key={href}
                      href={href}
                      className={`nav-item py-1.5 text-[12px] ${subNavActive(href) ? "nav-item-active" : ""}`}
                    >
                      <Icon className="nav-item-icon text-[15px]" />
                      {label}
                    </Link>
                  ))}
                </div>
              ) : null}
            </div>
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
            <p className="truncate text-[12.5px] leading-tight text-zinc-300">{user?.username ?? "—"}</p>
            <p className="truncate font-mono text-[10.5px] text-zinc-500">{t("nav.controlPanel")}</p>
          </div>
        </button>
        <AppVersion className="mt-2 px-2 text-center font-mono text-[10px] tabular-nums text-zinc-600" />
      </div>
    </aside>
  );
}
