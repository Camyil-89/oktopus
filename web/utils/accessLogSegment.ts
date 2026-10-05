import type { MessageKey } from "@/i18n/translate";

/** Вкладки журнала (query segment=). */
export type AccessLogSegment = "traffic" | "attacks" | "errors";

export const ACCESS_LOG_SEGMENTS: AccessLogSegment[] = [
  "traffic",
  "attacks",
  "errors",
];

export const ACCESS_LOG_SEGMENT_LABEL_KEYS: Record<
  AccessLogSegment,
  MessageKey
> = {
  traffic: "accessLog.segment.traffic",
  attacks: "accessLog.segment.attacks",
  errors: "accessLog.segment.errors",
};

/** Известные kind в extra.policy_anomaly (дублируйте kind из internal/proxy/observe/policyanomaly/*.go). */
export const ACCESS_LOG_ATTACK_KINDS: {
  id: string;
  labelKey: MessageKey;
}[] = [
  {
    id: "host_sni_mismatch",
    labelKey: "accessLog.policyAnomaly.host_sni_mismatch",
  },
  {
    id: "url_host_mismatch",
    labelKey: "accessLog.policyAnomaly.url_host_mismatch",
  },
  {
    id: "connect_port_mismatch",
    labelKey: "accessLog.policyAnomaly.connect_port_mismatch",
  },
  {
    id: "connect_literal_ip",
    labelKey: "accessLog.policyAnomaly.connect_literal_ip",
  },
  {
    id: "dst_resolve_private",
    labelKey: "accessLog.policyAnomaly.dst_resolve_private",
  },
];

export function parseAccessLogSegment(raw: unknown): AccessLogSegment {
  if (raw === "attacks" || raw === "errors" || raw === "traffic") {
    return raw;
  }
  return "traffic";
}
