import { OktopusLoading } from "@/assets/components/oktopus/OktopusLoading";

/** Индикатор для antd (Spin, Table, Card, Button, Modal и т.д.) */
export function OktopusAntdIndicator() {
  return (
    <span className="oktopus-antd-indicator inline-flex items-center justify-center leading-none">
      <OktopusLoading size="sm" decorative />
    </span>
  );
}
