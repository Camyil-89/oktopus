"use client";

import { scrollableModalProps } from "@/assets/modals/modalConfig";
import { useTranslation } from "@/contexts/LocaleContext";
import type { RulesSectionUnsaved } from "@/types/rulesUnsaved";
import { Button, Modal, Space } from "antd";
import { useRouter, usePathname } from "next/navigation";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";

type GuardHandlers = Pick<RulesSectionUnsaved, "discard" | "save">;

type ManageNavigationGuardContextValue = {
  registerGuard: (handlers: GuardHandlers | null) => void;
  confirmLeave: (run: () => void) => void;
};

const ManageNavigationGuardContext =
  createContext<ManageNavigationGuardContextValue | null>(null);

export function ManageNavigationGuardProvider({
  children,
}: {
  children: ReactNode;
}) {
  const router = useRouter();
  const pathname = usePathname();
  const { t } = useTranslation();
  const handlersRef = useRef<GuardHandlers | null>(null);
  const [guardActive, setGuardActive] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const pendingRunRef = useRef<(() => void) | null>(null);

  const registerGuard = useCallback((handlers: GuardHandlers | null) => {
    handlersRef.current = handlers;
    setGuardActive(handlers !== null);
  }, []);

  const confirmLeave = useCallback((run: () => void) => {
    if (!handlersRef.current) {
      run();
      return;
    }
    pendingRunRef.current = run;
    setModalOpen(true);
  }, []);

  const closeModal = () => {
    setModalOpen(false);
    pendingRunRef.current = null;
  };

  const proceedAfterDiscard = () => {
    const run = pendingRunRef.current;
    handlersRef.current?.discard();
    handlersRef.current = null;
    setGuardActive(false);
    closeModal();
    run?.();
  };

  const proceedAfterSave = async () => {
    const handlers = handlersRef.current;
    const run = pendingRunRef.current;
    if (!handlers || !run) {
      closeModal();
      return;
    }
    setSaving(true);
    try {
      const ok = await handlers.save();
      if (!ok) return;
      handlersRef.current = null;
      setGuardActive(false);
      closeModal();
      run();
    } finally {
      setSaving(false);
    }
  };

  useEffect(() => {
    if (!guardActive) return;
    const onBeforeUnload = (e: BeforeUnloadEvent) => {
      e.preventDefault();
      e.returnValue = "";
    };
    window.addEventListener("beforeunload", onBeforeUnload);
    return () => window.removeEventListener("beforeunload", onBeforeUnload);
  }, [guardActive]);

  useEffect(() => {
    if (!guardActive) return;
    const onClick = (e: MouseEvent) => {
      if (e.defaultPrevented) return;
      if (e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) {
        return;
      }
      const el = (e.target as Element | null)?.closest("a[href]");
      if (!el) return;
      const href = el.getAttribute("href");
      if (!href || href.startsWith("#")) return;
      if (el.getAttribute("target") === "_blank") return;
      let url: URL;
      try {
        url = new URL(href, window.location.href);
      } catch {
        return;
      }
      if (url.origin !== window.location.origin) return;
      const samePath =
        url.pathname === pathname && url.search === window.location.search;
      if (samePath) return;

      e.preventDefault();
      e.stopPropagation();
      confirmLeave(() => {
        router.push(href);
      });
    };
    document.addEventListener("click", onClick, true);
    return () => document.removeEventListener("click", onClick, true);
  }, [guardActive, confirmLeave, pathname, router]);

  const value = useMemo(
    () => ({ registerGuard, confirmLeave }),
    [registerGuard, confirmLeave],
  );

  return (
    <ManageNavigationGuardContext.Provider value={value}>
      {children}
      <Modal
        title={t("rules.unsavedTitle")}
        open={modalOpen}
        onCancel={closeModal}
        footer={
          <Space wrap>
            <Button onClick={closeModal}>{t("common.cancel")}</Button>
            <Button danger onClick={proceedAfterDiscard}>
              {t("rules.unsavedDiscard")}
            </Button>
            <Button
              type="primary"
              loading={saving}
              onClick={() => void proceedAfterSave()}
            >
              {t("common.save")}
            </Button>
          </Space>
        }
        {...scrollableModalProps}
      >
        {t("rules.unsavedMessage")}
      </Modal>
    </ManageNavigationGuardContext.Provider>
  );
}

export function useManageNavigationGuard() {
  const ctx = useContext(ManageNavigationGuardContext);
  if (!ctx) {
    throw new Error(
      "useManageNavigationGuard must be used within ManageNavigationGuardProvider",
    );
  }
  return ctx;
}
