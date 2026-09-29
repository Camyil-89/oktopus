"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { OktopusLoading } from "@/assets/components/oktopus/OktopusLoading";
import { useAuth } from "@/contexts/AuthContext";

export function AuthGate({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!loading && !user) {
      router.replace("/login");
    }
  }, [loading, user, router]);

  if (loading || !user) {
    return (
      <div className="grid-bg flex flex-1 items-center justify-center bg-ink p-8">
        <OktopusLoading size="xl" />
      </div>
    );
  }

  return children;
}
