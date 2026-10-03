"use client";

import { OKTOPUS_TENTACLES } from "@/assets/components/oktopus/tentacles";
import { useTranslation } from "@/contexts/LocaleContext";

const sizeClass = {
  sm: "h-6 w-6",
  md: "h-10 w-10",
  lg: "h-16 w-16",
  xl: "h-24 w-24",
} as const;

export type OktopusLoadingSize = keyof typeof sizeClass;

type OktopusLoadingProps = {
  className?: string;
  size?: OktopusLoadingSize;
  /** Подпись для screen readers; для встроенных индикаторов antd — `decorative` */
  title?: string;
  decorative?: boolean;
};

export function OktopusLoading({
  className = "",
  size = "lg",
  title,
  decorative = false,
}: OktopusLoadingProps) {
  const { t } = useTranslation();
  const ariaTitle = title ?? t("loading.title");
  const dim = sizeClass[size];
  return (
    <svg
      viewBox="0 0 120 120"
      className={`oktopus-loading ${dim} shrink-0 text-teal-300 ${className}`.trim()}
      fill="none"
      stroke="currentColor"
      strokeWidth={1.5}
      strokeLinejoin="round"
      strokeLinecap="round"
      role={decorative ? "presentation" : "img"}
      aria-hidden={decorative ? true : undefined}
      aria-label={decorative ? undefined : ariaTitle}
    >
      {!decorative ? <title>{ariaTitle}</title> : null}
      <circle cx="60" cy="40" r="20" />
      <circle cx="53" cy="38" r="2" fill="currentColor" stroke="none" />
      <circle cx="67" cy="38" r="2" fill="currentColor" stroke="none" />
      {OKTOPUS_TENTACLES.map((tentacle, i) => (
        <g key={tentacle.path} className={`tent t${i + 1}`}>
          <path className="base" d={tentacle.path} />
          <path
            className="flow"
            d={tentacle.path}
            pathLength={18}
            strokeDasharray="0.1 18"
            strokeDashoffset={0}
          />
          <circle
            cx={tentacle.dotCx}
            cy={tentacle.dotCy}
            r={2.6}
            fill="currentColor"
            stroke="none"
          />
        </g>
      ))}
    </svg>
  );
}
