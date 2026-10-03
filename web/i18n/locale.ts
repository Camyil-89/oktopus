import { isApiErrorCode } from "./errorCodes";
import type { MessageKey, TranslateFn } from "./translate";

export type Locale = "ru" | "en";

export const DEFAULT_LOCALE: Locale = "ru";

export const LOCALE_STORAGE_KEY = "oktopus_locale";

export function localeTag(locale: Locale): "ru-RU" | "en-US" {
  return locale === "en" ? "en-US" : "ru-RU";
}

export function parseStoredLocale(value: string | null): Locale {
  if (value === "en" || value === "ru") {
    return value;
  }
  return DEFAULT_LOCALE;
}

export function resolveApiErrorMessage(raw: string, t: TranslateFn): string {
  const trimmed = raw.trim();
  if (!trimmed) {
    return raw;
  }
  if (!isApiErrorCode(trimmed)) {
    return raw;
  }
  const key = `errors.${trimmed}` as MessageKey;
  const translated = t(key);
  if (translated === key) {
    return raw;
  }
  return translated;
}
