"use client";

import { LogoutOutlined } from "@ant-design/icons";
import { Button } from "antd";
import { usePathname } from "next/navigation";
import { useRouter } from "next/navigation";
import { useAuth } from "@/contexts/AuthContext";
import { managePageTitle } from "@/assets/components/manageNav";
import { ManageSidebar } from "@/assets/components/ManageSidebar";

export function ManageShell({ children }: { children: React.ReactNode }) {
  const { user, logout } = useAuth();
  const router = useRouter();
  const pathname = usePathname();
  const pageTitle = managePageTitle(pathname);

  const handleLogout = async () => {
    await logout();
    router.replace("/login");
  };

  return (
    <div className="flex h-dvh max-h-dvh overflow-hidden bg-ink text-zinc-200">
      <ManageSidebar onLogout={handleLogout} user={user} />
      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex h-14 shrink-0 flex-row items-center gap-4 border-b border-white/5 bg-ink/40 px-6 backdrop-blur">
          <h1 className="text-[14px] font-medium tracking-tight text-zinc-50">
            {pageTitle}
          </h1>
        </header>
        <main className="grid-bg min-h-0 flex-1 overflow-y-auto">
          <div className="manage-main-inner mx-auto max-w-[1200px] px-6 py-7">
            {children}
          </div>
        </main>
      </div>
    </div>
  );
}
