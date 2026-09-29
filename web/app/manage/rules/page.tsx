"use client";

import { ProxyACLSquidCard } from "@/assets/components/ProxyACLSquidCard";
import { ProxyInspectRulesCard } from "@/assets/components/ProxyInspectRulesCard";
import { Tabs } from "antd";

export default function ProxyRulesPage() {
  return (
    <Tabs
      defaultActiveKey="acl"
      items={[
        {
          key: "acl",
          label: "ACL",
          children: (
            <div className="flex w-full flex-col gap-4 pt-2">
              <ProxyACLSquidCard />
            </div>
          ),
        },
        {
          key: "inspect",
          label: "Инспекция исходящих",
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
