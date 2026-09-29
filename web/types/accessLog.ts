import type { AccessLogExtra } from "@/utils/accessLogExtra";

export type ProxyAccessLogRow = {
  id: string;
  created_at: string;
  source_address: string;
  destination_address: string;
  user: string | null;
  decide_duration_us: number;
  full_url: string;
  action: 0 | 1;
  inspect_rule_id: string | null;
  denied_by: "acl" | "inspect" | "gateway" | null;
  decision_rule_ref?: string;
  extra: AccessLogExtra;
};

export type ProxyAccessLogListResponse = {
  count: number;
  page: number;
  page_size: number;
  results: ProxyAccessLogRow[];
};

export type ListProxyAccessLogParams = {
  page?: number;
  page_size?: number;
  id?: string;
  user?: string;
  source?: string;
  destination?: string;
  url?: string;
  search_only?: boolean;
  from?: string;
  to?: string;
  action?: "0" | "1";
  error_kind?: "any";
  decision_rule_ref?: string;
  inspect_rule_id?: string;
};

export type DeleteProxyAccessLogByPeriodPayload = {
  from: string;
  to: string;
};

export type DeleteProxyAccessLogByPeriodResponse = {
  deleted: number;
};
