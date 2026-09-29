import type { ReactNode } from "react";
import { OktopusLoading } from "@/assets/components/oktopus/OktopusLoading";
import type { OktopusLoadingSize } from "@/assets/components/oktopus/OktopusLoading";

type SkeletonLoaderProps = {
  children: ReactNode;
  className?: string;
  loaderSize?: OktopusLoadingSize;
  showLoader?: boolean;
};

export function SkeletonLoader({
  children,
  className = "",
  loaderSize = "md",
  showLoader = true,
}: SkeletonLoaderProps) {
  return (
    <div className={`relative ${className}`.trim()}>
      {children}
      {showLoader ? (
        <div
          className="pointer-events-none absolute inset-0 flex items-center justify-center"
          aria-hidden
        >
          <OktopusLoading size={loaderSize} />
        </div>
      ) : null}
    </div>
  );
}
