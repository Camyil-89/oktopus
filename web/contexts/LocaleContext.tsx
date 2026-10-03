"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import {
  DEFAULT_LOCALE,
  LOCALE_STORAGE_KEY,
  localeTag,
  type Locale,
  parseStoredLocale,
  resolveApiErrorMessage,
} from "@/i18n/locale";
import { createT, type MessageKey, type TranslateFn } from "@/i18n/translate";

type LocaleContextValue = {
  locale: Locale;
  localeTag: "ru-RU" | "en-US";
  setLocale: (locale: Locale) => void;
  t: TranslateFn;
  apiErrorMessage: (raw: string) => string;
};

const LocaleContext = createContext<LocaleContextValue | null>(null);

function readStoredLocale(): Locale {
  if (typeof window === "undefined") {
    return DEFAULT_LOCALE;
  }
  return parseStoredLocale(localStorage.getItem(LOCALE_STORAGE_KEY));
}

export function LocaleProvider({ children }: { children: React.ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(DEFAULT_LOCALE);

  useEffect(() => {
    setLocaleState(readStoredLocale());
  }, []);

  const setLocale = useCallback((next: Locale) => {
    setLocaleState(next);
    localStorage.setItem(LOCALE_STORAGE_KEY, next);
  }, []);

  const t = useMemo(() => createT(locale), [locale]);

  const apiErrorMessage = useCallback(
    (raw: string) => resolveApiErrorMessage(raw, t),
    [t],
  );

  const tag = useMemo(() => localeTag(locale), [locale]);

  const value = useMemo(
    () => ({
      locale,
      localeTag: tag,
      setLocale,
      t,
      apiErrorMessage,
    }),
    [locale, tag, setLocale, t, apiErrorMessage],
  );

  return (
    <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>
  );
}

export function useLocale() {
  const ctx = useContext(LocaleContext);
  if (!ctx) {
    throw new Error("useLocale must be used within LocaleProvider");
  }
  return ctx;
}

export function useTranslation() {
  const ctx = useLocale();
  return ctx;
}

export function useApiErrorMessage() {
  const { apiErrorMessage } = useLocale();
  return useCallback(
    (e: unknown, fallback: string) => {
      if (e && typeof e === "object" && "message" in e) {
        const msg = String((e as { message: string }).message);
        if (msg) {
          return apiErrorMessage(msg);
        }
      }
      return fallback;
    },
    [apiErrorMessage],
  );
}

export type { MessageKey };
