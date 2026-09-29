/** Компактный размер в двоичных единицах (КиБ, МиБ). */
export function formatBytes(n: number): string {
  if (n <= 0) {
    return "0 Б";
  }
  const units = ["Б", "КиБ", "МиБ", "ГиБ", "ТиБ"];
  let v = n;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i++;
  }
  const digits = v >= 100 ? 0 : v >= 10 ? 1 : 2;
  return `${v.toFixed(digits)} ${units[i]}`;
}
