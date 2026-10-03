import type {
  Completion,
  CompletionContext,
  CompletionResult,
} from "@codemirror/autocomplete";
import type { EditorView } from "@codemirror/view";

export type ACLListMeta = {
  name: string;
  list_type: string;
};

export type ACLEntry = {
  name: string;
  kind: string;
  source: "config" | "list";
  preview: string;
};

const ACL_TYPE_HELP: Record<string, { detail: string; info: string }> = {
  src: {
    detail: "src",
    info: "Клиентский IP: адрес, CIDR (10.0.0.0/8) или диапазон.\nПример: acl nets src 192.168.0.0/16",
  },
  dstdomain: {
    detail: "dstdomain",
    info: "Домен/SNI: example.com, .suffix, *.wild.domain\nПример: acl blocked dstdomain .evil.test",
  },
  port: {
    detail: "port",
    info: "Порт или диапазон: 443, 80-90\nПример: acl safe_ports port 80 443",
  },
  url_regex: {
    detail: "url_regex",
    info: "Regex по host+path (HTTP).\nПример: acl bad url_regex (?i)blocked",
  },
  ldap_group: {
    detail: "ldap_group",
    info: "Группы LDAP пользователя (OR).\nПример: acl admins ldap_group IT-Admins",
  },
  all: {
    detail: "all",
    info: "Всегда совпадает.\nПример: acl all all",
  },
};

const DIRECTIVE_HELP: Record<string, string> = {
  acl: "acl <имя> <тип> <значения…>\nТипы: src, dstdomain, port, url_regex, ldap_group, all",
  http_access:
    "http_access allow|deny [!]<acl> …\nПорядок строк важен; первое совпадение решает.",
  ssl_verify:
    "ssl_verify require|skip [!]<acl> …\nПроверка TLS прокси→сайт (MITM). Первое совпадение решает; без правил — require.",
  delay_pools:
    "delay_pools N\nЧисло delay pool (1..N). Нужно для delay_class, delay_parameters, delay_access.",
  delay_class:
    "delay_class <pool> <1|2|3>\n1 — на IP клиента, 2 — общий bucket, 3 — /24 или /64.",
  delay_parameters:
    "delay_parameters <pool> <aggregate> <individual>\naggregate/individual: restore/max, none или -1/-1 (без лимита).\nПример: delay_parameters 1 none 1048576/2097152",
  delay_access:
    "delay_access <pool> allow|deny [!]<acl> …\nПервое совпадение: allow — применить pool, deny — без ограничения.",
  delay_initial_bucket_size:
    "delay_initial_bucket_size <pool> <bytes>\nСтартовый размер bucket.",
  allow: "Разрешить, если все acl на строке совпали.",
  deny: "Запретить, если все acl на строке совпали.",
  require: "Проверять сертификат origin (по умолчанию).",
  skip: "Не проверять сертификат origin (InsecureSkipVerify).",
};

const LINE_DIRECTIVES: Completion[] = [
  mkOption("acl", "keyword", "директива", DIRECTIVE_HELP.acl, "acl "),
  mkOption(
    "http_access",
    "keyword",
    "директива",
    DIRECTIVE_HELP.http_access,
    "http_access ",
  ),
  mkOption(
    "ssl_verify",
    "keyword",
    "директива",
    DIRECTIVE_HELP.ssl_verify,
    "ssl_verify ",
  ),
  mkOption(
    "delay_pools",
    "keyword",
    "директива",
    DIRECTIVE_HELP.delay_pools,
    "delay_pools ",
  ),
  mkOption(
    "delay_class",
    "keyword",
    "директива",
    DIRECTIVE_HELP.delay_class,
    "delay_class ",
  ),
  mkOption(
    "delay_parameters",
    "keyword",
    "директива",
    DIRECTIVE_HELP.delay_parameters,
    "delay_parameters ",
  ),
  mkOption(
    "delay_access",
    "keyword",
    "директива",
    DIRECTIVE_HELP.delay_access,
    "delay_access ",
  ),
  mkOption(
    "delay_initial_bucket_size",
    "keyword",
    "директива",
    DIRECTIVE_HELP.delay_initial_bucket_size,
    "delay_initial_bucket_size ",
  ),
];

function splitFields(line: string): string[] {
  const out: string[] = [];
  let cur = "";
  let inQuote = false;
  for (let i = 0; i < line.length; i++) {
    const c = line[i];
    if (c === '"') {
      inQuote = !inQuote;
      continue;
    }
    if (!inQuote && /\s/.test(c)) {
      if (cur) {
        out.push(cur);
        cur = "";
      }
      continue;
    }
    cur += c;
  }
  if (cur) out.push(cur);
  return out;
}

export function parseACLEntries(
  doc: string,
  lists: ACLListMeta[],
): ACLEntry[] {
  const out: ACLEntry[] = [];
  const seen = new Set<string>();
  for (const raw of doc.split("\n")) {
    const line = raw.trim().replace(/^\ufeff/, "");
    if (!line || line.startsWith("#")) continue;
    const f = splitFields(line);
    if (f[0]?.toLowerCase() !== "acl" || !f[1]) continue;
    const name = f[1];
    const kind = (f[2] ?? "").toLowerCase();
    const preview = f.slice(3).join(" ").slice(0, 80);
    if (!seen.has(name)) {
      seen.add(name);
      out.push({ name, kind, source: "config", preview });
    }
  }
  for (const l of lists) {
    const name = l.name.trim();
    if (!name || seen.has(name)) continue;
    seen.add(name);
    const kind = l.list_type === "sni" ? "dstdomain" : l.list_type;
    out.push({
      name,
      kind,
      preview: `list (${kind})`,
      source: "list",
    });
  }
  return out;
}

function infoPanel(text: string): HTMLElement {
  const el = document.createElement("div");
  el.className = "oktopus-acl-completion-info";
  el.style.whiteSpace = "pre-wrap";
  el.style.fontSize = "12px";
  el.style.lineHeight = "1.45";
  el.style.maxWidth = "360px";
  el.textContent = text;
  return el;
}

function insertReplace(
  view: EditorView,
  from: number,
  to: number,
  text: string,
) {
  view.dispatch({ changes: { from, to, insert: text } });
}

function mkOption(
  label: string,
  type: string,
  detail: string,
  infoText: string,
  insertText: string,
): Completion {
  return {
    label,
    type,
    detail,
    info: () => infoPanel(infoText),
    apply(view, _completion, from, to) {
      insertReplace(view, from, to, insertText);
    },
  };
}

function tokenMatch(context: CompletionContext) {
  return context.matchBefore(/!?[\w.-]*/);
}

function completeFrom(
  context: CompletionContext,
  options: Completion[],
  from?: number,
): CompletionResult | null {
  const word = tokenMatch(context);
  const start = from ?? word?.from ?? context.pos;
  const query = (word?.text ?? "").replace(/^!/, "").toLowerCase();
  const filtered = query
    ? options.filter((o) => o.label.toLowerCase().startsWith(query))
    : options;
  if (filtered.length === 0) return null;
  if (
    !context.explicit &&
    word &&
    word.from === word.to &&
    query.length === 0
  ) {
    return null;
  }
  return {
    from: start,
    options: filtered,
    validFor: /^!?[\w.-]*$/,
  };
}

function isHttpAccessToken(token: string): boolean {
  const t = token.toLowerCase();
  return t === "http_access" || "http_access".startsWith(t);
}

function isSSLVerifyToken(token: string): boolean {
  const t = token.toLowerCase();
  return t === "ssl_verify" || "ssl_verify".startsWith(t);
}

function isPolicyDirectiveToken(token: string): boolean {
  return isHttpAccessToken(token) || isSSLVerifyToken(token);
}

function directiveOptionsForPrefix(prefix: string): Completion[] {
  const p = prefix.toLowerCase();
  return LINE_DIRECTIVES.filter((o) => o.label.toLowerCase().startsWith(p));
}

export type SquidCompletionEnv = {
  listMetas: ACLListMeta[];
};

let completionEnv: SquidCompletionEnv = { listMetas: [] };

export function setSquidCompletionEnv(env: SquidCompletionEnv) {
  completionEnv = env;
}

export function squidPolicyCompletions(
  context: CompletionContext,
): CompletionResult | null {
  const doc = context.state.doc.toString();
  const catalog = parseACLEntries(doc, completionEnv.listMetas);
  const line = context.state.doc.lineAt(context.pos);
  const lineText = line.text;
  const before = lineText.slice(0, context.pos - line.from);
  const trimmed = before.trimStart();
  if (trimmed.startsWith("#")) return null;

  const fields = splitFields(trimmed);
  const kw = fields[0]?.toLowerCase() ?? "";

  if (fields.length === 0 || before.trim() === "") {
    return completeFrom(context, LINE_DIRECTIVES, line.from);
  }

  if (fields.length === 1 && kw !== "acl" && !isPolicyDirectiveToken(kw)) {
    const dirs = directiveOptionsForPrefix(kw);
    if (dirs.length > 0) {
      return completeFrom(context, dirs);
    }
  }

  if (fields.length === 1 && isHttpAccessToken(kw) && kw !== "http_access") {
    return completeFrom(context, [
      mkOption(
        "http_access",
        "keyword",
        "директива",
        DIRECTIVE_HELP.http_access,
        "http_access ",
      ),
    ]);
  }

  if (kw === "acl") {
    if (fields.length === 1) {
      return completeFrom(context, [
        mkOption("acl", "keyword", "", DIRECTIVE_HELP.acl, "acl "),
      ]);
    }
    if (fields.length === 2) {
      return completeFrom(
        context,
        Object.entries(ACL_TYPE_HELP).map(([k, h]) =>
          mkOption(k, "type", h.detail, h.info, `${k} `),
        ),
      );
    }
    if (fields.length === 3) {
      const t = fields[2].toLowerCase();
      const h = ACL_TYPE_HELP[t];
      if (h) {
        return completeFrom(context, [
          mkOption(t, "type", h.detail, h.info, `${t} `),
        ]);
      }
    }
    return null;
  }

  if (isSSLVerifyToken(kw) || kw === "ssl_verify") {
    if (fields.length === 1) {
      return completeFrom(context, [
        mkOption(
          "ssl_verify",
          "keyword",
          "",
          DIRECTIVE_HELP.ssl_verify,
          "ssl_verify ",
        ),
      ]);
    }
    if (fields.length === 2) {
      return completeFrom(context, [
        mkOption(
          "require",
          "keyword",
          "действие",
          DIRECTIVE_HELP.require,
          "require ",
        ),
        mkOption("skip", "keyword", "действие", DIRECTIVE_HELP.skip, "skip "),
      ]);
    }
    const aclOptions: Completion[] = catalog.map((e) => {
      const h = ACL_TYPE_HELP[e.kind] ?? ACL_TYPE_HELP.src;
      const src =
        e.source === "list"
          ? `List «${e.name}» (${e.kind})`
          : `ACL «${e.name}» (${e.kind})`;
      const info = `${src}\n${e.preview ? `Значения: ${e.preview}\n` : ""}${h.info}`;
      const name = e.name;
      return {
        label: name,
        type: "variable",
        detail: e.kind,
        info: () => infoPanel(info),
        apply(view: EditorView, _c: Completion, from: number, to: number) {
          insertReplace(view, from, to, `${name} `);
        },
      };
    });
    aclOptions.push(
      mkOption("all", "variable", "all", ACL_TYPE_HELP.all.info, "all "),
    );
    aclOptions.push(
      mkOption("!", "keyword", "отрицание", "Инверсия следующего acl: !name", "!"),
    );
    return completeFrom(context, aclOptions);
  }

  if (isHttpAccessToken(kw) || kw === "http_access") {
    if (fields.length === 1) {
      return completeFrom(context, [
        mkOption(
          "http_access",
          "keyword",
          "",
          DIRECTIVE_HELP.http_access,
          "http_access ",
        ),
      ]);
    }
    if (fields.length === 2) {
      return completeFrom(context, [
        mkOption("allow", "keyword", "действие", DIRECTIVE_HELP.allow, "allow "),
        mkOption("deny", "keyword", "действие", DIRECTIVE_HELP.deny, "deny "),
      ]);
    }
    const aclOptions: Completion[] = catalog.map((e) => {
      const h = ACL_TYPE_HELP[e.kind] ?? ACL_TYPE_HELP.src;
      const src =
        e.source === "list"
          ? `List «${e.name}» (${e.kind})`
          : `ACL «${e.name}» (${e.kind})`;
      const info = `${src}\n${e.preview ? `Значения: ${e.preview}\n` : ""}${h.info}`;
      const name = e.name;
      return {
        label: name,
        type: "variable",
        detail: e.kind,
        info: () => infoPanel(info),
        apply(view: EditorView, _c: Completion, from: number, to: number) {
          insertReplace(view, from, to, `${name} `);
        },
      };
    });
    aclOptions.push(
      mkOption("all", "variable", "all", ACL_TYPE_HELP.all.info, "all "),
    );
    aclOptions.push(
      mkOption("!", "keyword", "отрицание", "Инверсия следующего acl: !name", "!"),
    );
    return completeFrom(context, aclOptions);
  }

  return null;
}
