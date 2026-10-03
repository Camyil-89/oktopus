"use client";

import { ProxyACLSquidCard } from "@/assets/components/ProxyACLSquidCard";
import { ProxyInspectRulesCard } from "@/assets/components/ProxyInspectRulesCard";
import { useTranslation } from "@/contexts/LocaleContext";
import { Tabs } from "antd";

export default function ProxyRulesPage() {
  const { t } = useTranslation();
  return (
    <Tabs
      defaultActiveKey="acl"
      items={[
        {
          key: "acl",
          label: t("rules.tabAcl"),
          children: (
            <div className="flex w-full flex-col gap-4 pt-2">
              <ProxyACLSquidCard />
            </div>
          ),
        },
        {
          key: "inspect",
          label: t("rules.tabInspect"),
          children: (
            <div className="flex w-full flex-col gap-4 pt-2">
              <ProxyInspectRulesCard />
            </div>
          ),
        },
      ]}
    />
  );
}
