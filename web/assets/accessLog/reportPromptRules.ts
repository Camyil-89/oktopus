import { getProxyACLPolicy, listProxyInspectRules } from "@/api/proxy";
import { INSPECT_LUA_API_REFERENCE } from "@/assets/inspect/luaExamples";
import type { ProxyInspectRule } from "@/types/inspect";
import type { ProxyACLPolicy } from "@/types/proxy";

function inspectActionLabel(action: 0 | 1): string {
  return action === 0
    ? "deny_on_match (0) — при совпадении Lua запретить запрос"
    : "allow_on_match (1) — при совпадении отметить и проверять следующие правила";
}

export async function loadProxyRulesForReportPrompt(): Promise<{
  policy: ProxyACLPolicy;
  inspect: ProxyInspectRule[];
}> {
  const [policy, inspect] = await Promise.all([
    getProxyACLPolicy(),
    listProxyInspectRules(),
  ]);
  return {
    policy,
    inspect: [...inspect].sort((a, b) => a.sort_order - b.sort_order),
  };
}

export function formatAclPolicySection(policy: ProxyACLPolicy): string {
  const text = policy.config_text.trim();
  if (!text) {
    return "## Squid ACL policy\n(пустой config_text)\n";
  }
  return [
    "## Squid ACL policy (актуальная конфигурация)",
    "В журнале decision_rule_ref — текст строки ACL (например http_access allow all).",
    "",
    "```squid",
    text,
    "```",
  ].join("\n");
}

export function formatInspectRulesSection(rules: ProxyInspectRule[]): string {
  if (rules.length === 0) {
    return "## Правила инспекции (Lua)\n(нет правил)\n";
  }
  const blocks = rules.map((rule) =>
    [
      `### ${rule.name}`,
      `id: ${rule.id}`,
      `enabled: ${rule.enabled}`,
      `sort_order: ${rule.sort_order}`,
      `action: ${inspectActionLabel(rule.action)}`,
      `created_at: ${rule.created_at}`,
      `updated_at: ${rule.updated_at}`,
      `script:`,
      "```lua",
      rule.script,
      "```",
    ].join("\n"),
  );
  return [
    "## Правила инспекции (Lua, MITM / http)",
    "Порядок: sort_order. ctx:log в скрипте пишет в extra журнала под id правила (см. inspect_log.* в report spec).",
    "",
    INSPECT_LUA_API_REFERENCE,
    "",
    ...blocks,
  ].join("\n");
}
