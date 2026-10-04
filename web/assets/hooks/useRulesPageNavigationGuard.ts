"use client";

import { useManageNavigationGuard } from "@/contexts/ManageNavigationGuardContext";
import type { RulesSectionUnsaved } from "@/types/rulesUnsaved";
import { useEffect } from "react";

export function useRulesPageNavigationGuard(
  acl: RulesSectionUnsaved | null,
  inspect: RulesSectionUnsaved | null,
) {
  const { registerGuard } = useManageNavigationGuard();
  const dirty = Boolean(acl?.dirty || inspect?.dirty);

  useEffect(() => {
    if (!dirty) {
      registerGuard(null);
      return;
    }
    registerGuard({
      discard: () => {
        if (acl?.dirty) acl.discard();
        if (inspect?.dirty) inspect.discard();
      },
      save: async () => {
        let ok = true;
        if (acl?.dirty) ok = (await acl.save()) && ok;
        if (inspect?.dirty) ok = (await inspect.save()) && ok;
        return ok;
      },
    });
    return () => registerGuard(null);
  }, [acl, inspect, dirty, registerGuard]);
}
