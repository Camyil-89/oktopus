import type { Locale } from "@/i18n/locale";
import type { TranslateFn } from "@/i18n/translate";

export type ByteUnitLabels = {
  byte: string;
  kib: string;
  mib: string;
  gib: string;
  tib: string;
};

export function byteUnitLabelsFromT(t: TranslateFn): ByteUnitLabels {
  return {
    byte: t("units.byte"),
    kib: t("units.kib"),
    mib: t("units.mib"),
    gib: t("units.gib"),
    tib: t("units.tib"),
  };
}

/** Компактный размер в двоичных единицах (КиБ, МиБ). */
export function formatBytes(
  n: number,
  units: ByteUnitLabels = {
    byte: "Б",
    kib: "КиБ",
    mib: "МиБ",
    gib: "ГиБ",
    tib: "ТиБ",
  },
): string {
  if (n <= 0) {
    return `0 ${units.byte}`;
  }
  const unitList = [units.byte, units.kib, units.mib, units.gib, units.tib];
  let v = n;
  let i = 0;
  while (v >= 1024 && i < unitList.length - 1) {
    v /= 1024;
    i++;
  }
  const digits = v >= 100 ? 0 : v >= 10 ? 1 : 2;
  return `${v.toFixed(digits)} ${unitList[i]}`;
}

export function formatBytesForLocale(n: number, locale: Locale, t: TranslateFn) {
  return formatBytes(n, byteUnitLabelsFromT(t));
}
