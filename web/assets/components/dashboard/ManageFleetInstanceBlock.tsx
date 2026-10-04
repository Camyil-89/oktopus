"use client";

import { InstanceRuntimeOverview } from "@/assets/components/dashboard/InstanceRuntimeOverview";
import type { ProxyInstance, ProxyInstanceRuntimeStatus } from "@/types/proxy";
import { instanceRuntimeFingerprint } from "@/utils/runtimeStatusFingerprint";
import Link from "next/link";
import { memo } from "react";

type Props = {
  meta: ProxyInstance;
  runtime: ProxyInstanceRuntimeStatus;
  loading: boolean;
};

function ManageFleetInstanceBlockInner({ meta, runtime, loading }: Props) {
  return (
    <div className="flex flex-col gap-3">
      <Link
        href={`/manage/instances/${meta.id}`}
        className="text-[13px] font-medium text-teal-300 hover:text-teal-200"
      >
        {meta.name}
      </Link>
      <InstanceRuntimeOverview inst={runtime} loading={loading} showCharts />
    </div>
  );
}

function propsEqual(prev: Props, next: Props): boolean {
  if (prev.loading !== next.loading) {
    return false;
  }
  if (prev.meta.id !== next.meta.id) {
    return false;
  }
  if (
    prev.meta.name !== next.meta.name ||
    prev.meta.listen !== next.meta.listen
  ) {
    return false;
  }
  return (
    instanceRuntimeFingerprint(prev.runtime) ===
    instanceRuntimeFingerprint(next.runtime)
  );
}

export const ManageFleetInstanceBlock = memo(
  ManageFleetInstanceBlockInner,
  propsEqual,
);
