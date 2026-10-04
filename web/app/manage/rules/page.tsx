"use client";

import { ProxyACLSquidCard } from "@/assets/components/ProxyACLSquidCard";
import { ProxyInspectRulesCard } from "@/assets/components/ProxyInspectRulesCard";
import { useRulesPageNavigationGuard } from "@/assets/hooks/useRulesPageNavigationGuard";
import { useTranslation } from "@/contexts/LocaleContext";
import type { RulesSectionUnsaved } from "@/types/rulesUnsaved";
import { Tabs } from "antd";
import { useCallback, useState } from "react";

export default function ProxyRulesPage() {
  const { t } = useTranslation();
  const [aclUnsaved, setAclUnsaved] = useState<RulesSectionUnsaved | null>(
    null,
  );
  const [inspectUnsaved, setInspectUnsaved] =
    useState<RulesSectionUnsaved | null>(null);

  const onAclUnsaved = useCallback((state: RulesSectionUnsaved) => {
    setAclUnsaved(state);
  }, []);
  const onInspectUnsaved = useCallback((state: RulesSectionUnsaved) => {
    setInspectUnsaved(state);
  }, []);

  useRulesPageNavigationGuard(aclUnsaved, inspectUnsaved);

  return (
    <Tabs
      defaultActiveKey="acl"
      items={[
        {
          key: "acl",
          label: t("rules.tabAcl"),
          children: (
            <div className="flex w-full flex-col gap-4 pt-2">
              <ProxyACLSquidCard onUnsavedChange={onAclUnsaved} />
            </div>
          ),
        },
        {
          key: "inspect",
          label: t("rules.tabInspect"),
          children: (
            <div className="flex w-full flex-col gap-4 pt-2">
              <ProxyInspectRulesCard onUnsavedChange={onInspectUnsaved} />
            </div>
          ),
        },
      ]}
    />
  );
}
