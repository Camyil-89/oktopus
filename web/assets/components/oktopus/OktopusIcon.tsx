import { OKTOPUS_TENTACLES } from "@/assets/components/oktopus/tentacles";

const sizeClass = {
  sm: "h-6 w-6",
  md: "h-7 w-7",
  lg: "h-9 w-9",
  xl: "h-24 w-24",
} as const;

export type OktopusIconSize = keyof typeof sizeClass;

type OktopusIconProps = {
  className?: string;
  size?: OktopusIconSize;
  title?: string;
};

export function OktopusIcon({
  className = "",
  size = "md",
  title,
}: OktopusIconProps) {
  const dim = sizeClass[size];
  return (
    <svg
      viewBox="0 0 120 120"
      className={`${dim} shrink-0 text-teal-300 ${className}`.trim()}
      fill="none"
      stroke="currentColor"
      strokeWidth={1.5}
      strokeLinejoin="round"
      strokeLinecap="round"
      role={title ? "img" : "presentation"}
      aria-hidden={title ? undefined : true}
    >
      {title ? <title>{title}</title> : null}
      <circle cx="60" cy="40" r="20" />
      <circle cx="53" cy="38" r="2" fill="currentColor" stroke="none" />
      <circle cx="67" cy="38" r="2" fill="currentColor" stroke="none" />
      {OKTOPUS_TENTACLES.map((t) => (
        <path key={t.path} d={t.path} />
      ))}
      {OKTOPUS_TENTACLES.map((t) => (
        <circle
          key={`dot-${t.path}`}
          cx={t.dotCx}
          cy={t.dotCy}
          r={2.6}
          fill="currentColor"
          stroke="none"
        />
      ))}
    </svg>
  );
}
