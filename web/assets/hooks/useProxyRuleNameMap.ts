"use client";

import { listProxyInspectRules } from "@/api/proxy";
import { useEffect, useState } from "react";

export type ProxyRuleKind = "inspect";

export function useProxyRuleNameMap(): {
  ruleNames: ReadonlyMap<string, string>;
  ruleKinds: ReadonlyMap<string, ProxyRuleKind>;
  loading: boolean;
} {
  const [ruleNames, setRuleNames] = useState<Map<string, string>>(
    () => new Map(),
  );
  const [ruleKinds, setRuleKinds] = useState<Map<string, ProxyRuleKind>>(
    () => new Map(),
  );
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    void listProxyInspectRules()
      .then((inspect) => {
        if (cancelled) {
          return;
        }
        const names = new Map<string, string>();
        const kinds = new Map<string, ProxyRuleKind>();
        for (const r of inspect) {
          names.set(r.id, r.name);
          kinds.set(r.id, "inspect");
        }
        setRuleNames(names);
        setRuleKinds(kinds);
      })
      .catch(() => {
        if (!cancelled) {
          setRuleNames(new Map());
          setRuleKinds(new Map());
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return { ruleNames, ruleKinds, loading };
}
