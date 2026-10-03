"use client";

import { GlobalOutlined } from "@ant-design/icons";
import { Select } from "antd";
import { useLocale } from "@/contexts/LocaleContext";
import type { Locale } from "@/i18n/locale";

const OPTIONS: { value: Locale; label: string }[] = [
  { value: "ru", label: "Русский" },
  { value: "en", label: "English" },
];

export function LanguageSelect() {
  const { locale, setLocale } = useLocale();

  return (
    <Select<Locale>
      aria-label="Language"
      className="min-w-[120px]"
      value={locale}
      onChange={setLocale}
      options={OPTIONS}
      suffixIcon={<GlobalOutlined />}
      size="small"
    />
  );
}
