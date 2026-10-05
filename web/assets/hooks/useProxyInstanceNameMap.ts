"use client";

import { listProxyInstances } from "@/api/proxy";
import { useEffect, useState } from "react";

export function useProxyInstanceNameMap(): ReadonlyMap<string, string> {
  const [names, setNames] = useState<Map<string, string>>(() => new Map());

  useEffect(() => {
    void listProxyInstances()
      .then((list) => {
        const m = new Map<string, string>();
        for (const i of list) {
          m.set(i.id, i.name);
        }
        setNames(m);
      })
      .catch(() => setNames(new Map()));
  }, []);

  return names;
}
