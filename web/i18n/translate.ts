import type { Locale } from "./locale";
import { en } from "./locales/en";
import { ru } from "./locales/ru";

export type MessageKey = keyof typeof ru;

const catalogs: Record<Locale, Record<MessageKey, string>> = {
  ru,
  en,
};

export type TranslateFn = (
  key: MessageKey,
  params?: Record<string, string | number>,
) => string;

export function createT(locale: Locale): TranslateFn {
  const table = catalogs[locale];
  const fallback = catalogs.ru;
  return (key, params) => {
    let text = table[key] ?? fallback[key] ?? key;
    if (params) {
      for (const [name, value] of Object.entries(params)) {
        text = text.replaceAll(`{${name}}`, String(value));
      }
    }
    return text;
  };
}
