import type { AccessLogExtra } from "@/utils/accessLogExtra";
import { accessLogInspectPayloads, parseAccessLogExtra } from "@/utils/accessLogExtra";

/** Ключи ctx:log для эвристики загрузки файлов (типичный набор). */
export const INSPECT_LOG_UPLOAD_FIELDS = [
  {
    key: "upload_method",
    description: "HTTP-метод запроса (POST, PUT, PATCH)",
  },
  {
    key: "upload_content_type",
    description: "Базовый Content-Type без параметров",
  },
  {
    key: "upload_content_length",
    description: "Content-Length (число)",
  },
  {
    key: "upload_filenames",
    description: "Массив имён файлов (JSON-массив в журнале)",
  },
  {
    key: "upload_header_has_filename",
    description: "В Content-Disposition есть filename= (boolean)",
  },
] as const;

export const INSPECT_LOG_GROUP_BY_PREFIX = "inspect_log.";

const INSPECT_LOG_KEY_RE = /^[a-z][a-z0-9_]{0,63}$/;

const CTX_LOG_STRING_RE = /ctx:log\s*\(\s*["']([^"']+)["']/g;
const CTX_LOG_TABLE_RE = /ctx:log\s*\(\s*\{([^}]+)\}/g;
const CTX_LOG_TABLE_KEY_RE = /["']?([a-z][a-z0-9_]*)["']?\s*=/g;

export function isInspectLogFieldKey(key: string): boolean {
  return INSPECT_LOG_KEY_RE.test(key);
}

export function inspectLogGroupByField(key: string): string {
  return `${INSPECT_LOG_GROUP_BY_PREFIX}${key}`;
}

export function parseInspectLogGroupByField(
  name: string,
): string | null {
  if (!name.startsWith(INSPECT_LOG_GROUP_BY_PREFIX)) {
    return null;
  }
  const key = name.slice(INSPECT_LOG_GROUP_BY_PREFIX.length);
  return isInspectLogFieldKey(key) ? key : null;
}

/** Ключи ctx:log из исходников Lua (примеры правил инспекции). */
export function parseCtxLogKeysFromLuaScripts(scripts: string[]): string[] {
  const keys = new Set<string>();
  for (const script of scripts) {
    for (const m of script.matchAll(CTX_LOG_STRING_RE)) {
      const k = m[1]?.trim();
      if (k && isInspectLogFieldKey(k)) {
        keys.add(k);
      }
    }
    for (const block of script.matchAll(CTX_LOG_TABLE_RE)) {
      const inner = block[1] ?? "";
      for (const km of inner.matchAll(CTX_LOG_TABLE_KEY_RE)) {
        const k = km[1]?.trim();
        if (k && isInspectLogFieldKey(k)) {
          keys.add(k);
        }
      }
    }
  }
  return [...keys].sort();
}

/** Все ключи из payload inspect-правил в extra (по фактическим записям). */
export function collectInspectLogKeysFromExtra(
  extra: AccessLogExtra | unknown,
): string[] {
  const parsed = parseAccessLogExtra(extra);
  const keys = new Set<string>();
  for (const { data } of accessLogInspectPayloads(parsed)) {
    for (const k of Object.keys(data)) {
      if (isInspectLogFieldKey(k)) {
        keys.add(k);
      }
    }
  }
  return [...keys].sort();
}

export function collectInspectLogKeysFromExtras(
  extras: unknown[],
): string[] {
  const keys = new Set<string>();
  for (const e of extras) {
    for (const k of collectInspectLogKeysFromExtra(e)) {
      keys.add(k);
    }
  }
  return [...keys].sort();
}

/** Ключи ctx:log только из переданных Lua-скриптов (правила инспекции). */
export function mergedInspectLogFieldKeys(luaScripts: string[]): string[] {
  return parseCtxLogKeysFromLuaScripts(luaScripts);
}

export function formatInspectLogValue(value: unknown): string {
  if (value === null || value === undefined) {
    return "—";
  }
  if (typeof value === "boolean") {
    return value ? "true" : "false";
  }
  if (typeof value === "number") {
    return String(value);
  }
  if (typeof value === "string") {
    return value;
  }
  if (Array.isArray(value)) {
    return value.map((v) => formatInspectLogValue(v)).join(", ");
  }
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

/** Значения inspect ctx:log по ключу (первое непустое среди правил). */
export function getInspectLogFieldFromExtra(
  extra: AccessLogExtra | unknown,
  fieldKey: string,
): unknown {
  if (!isInspectLogFieldKey(fieldKey)) {
    return undefined;
  }
  const parsed = parseAccessLogExtra(extra);
  for (const { data } of accessLogInspectPayloads(parsed)) {
    if (fieldKey in data) {
      return data[fieldKey];
    }
  }
  return undefined;
}

export function formatInspectLogFieldsForPrompt(fieldKeys: string[]): string {
  const metaByKey: Record<string, string> = {};
  for (const f of INSPECT_LOG_UPLOAD_FIELDS) {
    metaByKey[f.key] = f.description;
  }
  const lines: string[] = [
    "Поля extra от ctx:log (под UUID inspect-правила в JSON extra):",
    "  В group_by / group_by_cols: inspect_log.<ключ>",
    "  В filters.inspect_log: { \"<ключ>\": \"подстрока\" } (ILIKE по строковому значению)",
  ];
  if (fieldKeys.length === 0) {
    lines.push(
      "  (в правилах инспекции нет ctx:log — список пуст; см. секцию правил ниже)",
    );
  } else {
    for (const key of fieldKeys) {
      const desc = metaByKey[key];
      const note = desc ? ` — ${desc}` : "";
      lines.push(`  inspect_log.${key}${note}`);
    }
    if (fieldKeys.includes("upload_filenames")) {
      lines.push(
        "  upload_filenames в group_by — JSON-массив как текст; для отдельных имён нужен свой ключ в Lua.",
      );
    }
  }
  return lines.join("\n");
}
