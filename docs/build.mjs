import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const GLOBAL_MANAGE_NAV = {
  ru: [
    { id: "manage-home", label: "Главная" },
    { id: "manage-access-log", label: "Журнал доступа" },
    { id: "manage-access-log-reports", label: "Отчёты журнала" },
    { id: "manage-users", label: "Пользователи" },
  ],
  en: [
    { id: "manage-home", label: "Home" },
    { id: "manage-access-log", label: "Access log" },
    { id: "manage-access-log-reports", label: "Access log reports" },
    { id: "manage-users", label: "Users" },
  ],
};

const INSTANCE_SIDEBAR = {
  ru: {
    sampleName: "Офис",
    stats: "Статистика",
    rules: "Правила прокси",
    settings: "Настройки",
    accessLog: "Журнал доступа",
  },
  en: {
    sampleName: "Office",
    stats: "Statistics",
    rules: "Proxy rules",
    settings: "Settings",
    accessLog: "Access log",
  },
};

/** Which instance sub-nav item is active (null = instance row only, no sub-nav). */
const INSTANCE_NAV_ACTIVE = {
  "manage-instance": "stats",
  "manage-rules": "rules",
  "manage-proxy": "settings",
};

const UI_MOCK_PAGE_IDS = new Set([
  "login",
  "manage-home",
  "manage-instance",
  "manage-rules",
  "rules-acl-syntax",
  "manage-access-log",
  "manage-access-log-reports",
  "manage-proxy",
  "manage-users",
]);

const MERMAID_PAGE_IDS = new Set(["implementation-examples"]);

const PAGES = [
  { id: "intro", file: "index.html" },
  { id: "quickstart", file: "quickstart.html" },
  { id: "implementation-examples", file: "implementation-examples.html" },
  { id: "login", file: "login.html" },
  { id: "manage-home", file: "manage/index.html" },
  { id: "manage-instance", file: "manage/instance.html" },
  { id: "manage-rules", file: "manage/rules.html" },
  { id: "rules-acl-syntax", file: "manage/rules-acl-syntax.html" },
  { id: "manage-access-log", file: "manage/access-log.html" },
  { id: "manage-access-log-reports", file: "manage/reports.html" },
  { id: "manage-proxy", file: "manage/proxy.html" },
  { id: "manage-users", file: "manage/users.html" },
];

const META = {
  ru: {
    intro: { section: "Разработка", title: "Разработчикам", pageTitle: "Oktopus — Разработчикам" },
    quickstart: { section: "Развёртывание", title: "Развёртывание", pageTitle: "Oktopus — Развёртывание" },
    "implementation-examples": {
      section: "Развёртывание",
      title: "Примеры реализаций",
      pageTitle: "Oktopus — Примеры реализаций",
    },
    login: { section: "Вход", title: "Вход", pageTitle: "Oktopus — Вход" },
    "manage-home": { section: "Панель управления", title: "Главная", pageTitle: "Oktopus — Главная" },
    "manage-instance": {
      section: "Панель управления",
      title: "Статистика инстанса",
      pageTitle: "Oktopus — Статистика инстанса",
    },
    "manage-rules": { section: "Панель управления", title: "Правила прокси", pageTitle: "Oktopus — Правила прокси" },
    "rules-acl-syntax": {
      section: "Справочник",
      title: "Синтаксис ACL",
      pageTitle: "Oktopus — Синтаксис ACL",
    },
    "manage-access-log": { section: "Панель управления", title: "Журнал доступа", pageTitle: "Oktopus — Журнал доступа" },
    "manage-access-log-reports": {
      section: "Панель управления",
      title: "Отчёты журнала",
      pageTitle: "Oktopus — Отчёты журнала",
    },
    "manage-proxy": { section: "Панель управления", title: "Настройки прокси", pageTitle: "Oktopus — Настройки прокси" },
    "manage-users": { section: "Панель управления", title: "Пользователи", pageTitle: "Oktopus — Пользователи" },
  },
  en: {
    intro: { section: "Development", title: "Developers", pageTitle: "Oktopus — Developers" },
    quickstart: { section: "Deployment", title: "Deployment", pageTitle: "Oktopus — Deployment" },
    "implementation-examples": {
      section: "Deployment",
      title: "Implementation patterns",
      pageTitle: "Oktopus — Implementation patterns",
    },
    login: { section: "Sign in", title: "Sign in", pageTitle: "Oktopus — Sign in" },
    "manage-home": { section: "Control panel", title: "Home", pageTitle: "Oktopus — Home" },
    "manage-instance": {
      section: "Control panel",
      title: "Instance statistics",
      pageTitle: "Oktopus — Instance statistics",
    },
    "manage-rules": { section: "Control panel", title: "Proxy rules", pageTitle: "Oktopus — Proxy rules" },
    "rules-acl-syntax": {
      section: "Reference",
      title: "ACL syntax",
      pageTitle: "Oktopus — ACL syntax",
    },
    "manage-access-log": { section: "Control panel", title: "Access log", pageTitle: "Oktopus — Access log" },
    "manage-access-log-reports": {
      section: "Control panel",
      title: "Access log reports",
      pageTitle: "Oktopus — Access log reports",
    },
    "manage-proxy": { section: "Control panel", title: "Proxy settings", pageTitle: "Oktopus — Proxy settings" },
    "manage-users": { section: "Control panel", title: "Users", pageTitle: "Oktopus — Users" },
  },
};

const NAV = {
  ru: {
    sections: [
      {
        label: "Развёртывание",
        items: [
          { id: "quickstart", label: "Развёртывание" },
          { id: "implementation-examples", label: "Примеры реализаций" },
        ],
      },
      {
        label: "Разработка",
        items: [{ id: "intro", label: "Разработчикам" }],
      },
      { label: "Вход", items: [{ id: "login", label: "Вход" }] },
      {
        label: "Панель управления",
        items: [
          { id: "manage-home", label: "Главная" },
          { id: "manage-instance", label: "Статистика инстанса" },
          { id: "manage-rules", label: "Правила прокси" },
          { id: "manage-access-log", label: "Журнал доступа" },
          { id: "manage-access-log-reports", label: "Отчёты журнала" },
          { id: "manage-proxy", label: "Настройки прокси" },
          { id: "manage-users", label: "Пользователи" },
        ],
      },
      {
        label: "Справочник",
        items: [{ id: "rules-acl-syntax", label: "Синтаксис ACL" }],
      },
    ],
    search: "Поиск по разделам…",
    toc: "На этой странице",
    pagerBack: "Назад",
    pagerNext: "Далее",
    github: "исходный код",
    copy: "Копировать",
    copied: "Скопировано",
    footer: "docs / ru",
    aiPromptTitle: "Помощь через ИИ",
    aiPromptLead:
      "Скопируйте промпт ниже и вставьте в чат с ИИ (ChatGPT, Claude и т.п.). Опишите, кого и куда нужно пустить или запретить — ассистент предложит фрагмент ACL/http_access для Oktopus.",
    aiPromptLabel: "промпт",
    aiPromptCopy: "Копировать промпт",
  },
  en: {
    sections: [
      {
        label: "Deployment",
        items: [
          { id: "quickstart", label: "Deployment" },
          { id: "implementation-examples", label: "Implementation patterns" },
        ],
      },
      {
        label: "Development",
        items: [{ id: "intro", label: "Developers" }],
      },
      { label: "Sign in", items: [{ id: "login", label: "Sign in" }] },
      {
        label: "Control panel",
        items: [
          { id: "manage-home", label: "Home" },
          { id: "manage-instance", label: "Instance statistics" },
          { id: "manage-rules", label: "Proxy rules" },
          { id: "manage-access-log", label: "Access log" },
          { id: "manage-access-log-reports", label: "Access log reports" },
          { id: "manage-proxy", label: "Proxy settings" },
          { id: "manage-users", label: "Users" },
        ],
      },
      {
        label: "Reference",
        items: [{ id: "rules-acl-syntax", label: "ACL syntax" }],
      },
    ],
    search: "Search sections…",
    toc: "On this page",
    pagerBack: "Back",
    pagerNext: "Next",
    github: "source code",
    copy: "Copy",
    copied: "Copied",
    footer: "docs / en",
    aiPromptTitle: "Help from AI",
    aiPromptLead:
      "Copy the prompt below and paste it into an AI chat (ChatGPT, Claude, etc.). Describe who should be allowed or denied — the assistant will suggest acl/http_access config for Oktopus.",
    aiPromptLabel: "prompt",
    aiPromptCopy: "Copy prompt",
  },
};

function pageFile(id) {
  return PAGES.find((p) => p.id === id).file;
}

function manageAsideGlobalActiveId(pageId) {
  if (pageId === "rules-acl-syntax") return "manage-rules";
  if (INSTANCE_NAV_ACTIVE[pageId]) return null;
  return pageId;
}

function renderManageAside(locale, pageId, assets) {
  const asideTour =
    locale === "ru"
      ? {
          title: "Боковое меню",
          body:
            "Общие разделы: главная, журнал и отчёты по всем инстансам, пользователи кабинета. Ниже — каждый прокси-инстанс: при открытии появляются подпункты (статистика, правила, настройки, журнал). Выход — внизу.",
        }
      : {
          title: "Sidebar",
          body:
            "Global sections: home, access log and reports across instances, console users. Below — each proxy instance; when opened, sub-items appear (statistics, rules, settings, log). Sign out is at the bottom.",
        };
  const globalActive = manageAsideGlobalActiveId(pageId);
  const globalItems = GLOBAL_MANAGE_NAV[locale]
    .map(
      ({ id, label }) =>
        `<div class="ui-mock-nav-item${id === globalActive ? " is-active" : ""}">${label}</div>`,
    )
    .join("\n                    ");

  const inst = INSTANCE_SIDEBAR[locale];
  const instSection = INSTANCE_NAV_ACTIVE[pageId];
  const instanceRowClass = instSection
    ? "ui-mock-nav-item is-active ui-mock-nav-instance"
    : "ui-mock-nav-item ui-mock-nav-instance";
  let instanceBlock = `<div class="${instanceRowClass}">${inst.sampleName}</div>`;
  if (instSection) {
    const subs = [
      { key: "stats", label: inst.stats },
      { key: "rules", label: inst.rules },
      { key: "settings", label: inst.settings },
      { key: "accessLog", label: inst.accessLog },
    ];
    instanceBlock += `\n                    <div class="ui-mock-nav-sub">`;
    instanceBlock += subs
      .map(
        ({ key, label }) =>
          `<div class="ui-mock-nav-item ui-mock-nav-sub-item${instSection === key ? " is-active" : ""}">${label}</div>`,
      )
      .join("\n                    ");
    instanceBlock += `</div>`;
  }

  return `<aside class="ui-mock-aside">
                  <div class="ui-mock-brand">
                    <img src="${assets}icon.svg" alt="">
                    <span class="ui-mock-brand-text">Oktopus</span>
                  </div>
                  <nav class="ui-mock-nav tour-zone" tabindex="0"
                    data-tour-title="${asideTour.title}"
                    data-tour-body="${asideTour.body}">
                    ${globalItems}
                    ${instanceBlock}
                  </nav>
                </aside>`;
}

function preparePageBody(locale, pageId, assets) {
  const contentPath = path.join(__dirname, "content", locale, `${pageId}.html`);
  let body = fs.readFileSync(contentPath, "utf8").replaceAll("__ASSETS__", assets);
  if (body.includes("__MANAGE_ASIDE__")) {
    body = body.replace(
      "__MANAGE_ASIDE__",
      renderManageAside(locale, pageId, assets),
    );
  }
  return body;
}

function escapeHtml(s) {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function renderAiPrompt(locale, pageId, cfg) {
  const promptPath = path.join(__dirname, "content", "prompts", locale, `${pageId}.txt`);
  if (!fs.existsSync(promptPath)) return "";
  const prompt = fs.readFileSync(promptPath, "utf8").trim();
  if (!prompt) return "";
  return `
            <div class="ai-prompt-block">
              <h2 id="ai-help">${cfg.aiPromptTitle}</h2>
              <p>${cfg.aiPromptLead}</p>
              <div class="code">
                <div class="code-head"><span class="code-name">${cfg.aiPromptLabel}</span><button type="button" class="copy-btn" onclick="copyAiPrompt(this)">${cfg.aiPromptCopy}</button></div>
                <pre id="ai-prompt-pre"><code>${escapeHtml(prompt)}</code></pre>
              </div>
            </div>`;
}

function relHref(fromFile, toFile) {
  const fromDir = path.dirname(fromFile);
  let rel = path.relative(fromDir, toFile).split(path.sep).join("/");
  if (!rel.startsWith(".")) rel = "./" + rel;
  return rel;
}

function assetPrefix(file) {
  const depth = file.split("/").length - 1;
  return "../".repeat(depth + 1) + "assets/";
}

function langHref(locale, fromFile, toFile) {
  const other = locale === "ru" ? "en" : "ru";
  return relHref(fromFile, path.join(other, toFile));
}

function renderNav(locale, activeId, fromFile) {
  const cfg = NAV[locale];
  return cfg.sections
    .map(
      (sec) => `
      <div>
        <p class="px-3 pb-1.5 text-[10px] font-mono uppercase tracking-[0.16em] text-zinc-700">${sec.label}</p>
        <div class="flex flex-col gap-0.5">
          ${sec.items
            .map((item) => {
              const href = relHref(fromFile, pageFile(item.id));
              const active = item.id === activeId ? " active" : "";
              return `<a href="${href}" class="nav-item${active}"><span class="dot"></span>${item.label}</a>`;
            })
            .join("\n")}
        </div>
      </div>`,
    )
    .join("\n");
}

function renderPager(locale, pageId, fromFile) {
  const cfg = NAV[locale];
  const i = PAGES.findIndex((p) => p.id === pageId);
  const prev = PAGES[i - 1];
  const next = PAGES[i + 1];
  const meta = META[locale];

  const prevBlock = prev
    ? `<a href="${relHref(fromFile, pageFile(prev.id))}" class="pager-link group text-left">
        <svg class="w-4 h-4 text-zinc-600 transition flex-shrink-0" fill="none" stroke="currentColor" stroke-width="1.7" viewBox="0 0 24 24" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" d="M15.75 19.5 8.25 12l7.5-7.5"/></svg>
        <span class="min-w-0">
          <span class="block text-[10.5px] font-mono uppercase tracking-[0.12em] text-zinc-700">${cfg.pagerBack}</span>
          <span class="pager-title block text-[13px] text-zinc-400 transition truncate">${meta[prev.id].title}</span>
        </span>
      </a>`
    : `<span class="pager-link pager-hidden" aria-hidden="true"></span>`;

  const nextBlock = next
    ? `<a href="${relHref(fromFile, pageFile(next.id))}" class="pager-link group text-right ml-auto">
        <span class="min-w-0">
          <span class="block text-[10.5px] font-mono uppercase tracking-[0.12em] text-zinc-700">${cfg.pagerNext}</span>
          <span class="pager-title block text-[13px] text-zinc-400 transition truncate">${meta[next.id].title}</span>
        </span>
        <svg class="w-4 h-4 text-zinc-600 transition flex-shrink-0" fill="none" stroke="currentColor" stroke-width="1.7" viewBox="0 0 24 24" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" d="m8.25 4.5 7.5 7.5-7.5 7.5"/></svg>
      </a>`
    : `<span class="pager-link pager-hidden ml-auto" aria-hidden="true"></span>`;

  return `<div class="flex items-center justify-between gap-3 mt-14 pt-6 border-t border-white/5">${prevBlock}${nextBlock}</div>`;
}

function renderPage(locale, pageId) {
  const page = PAGES.find((p) => p.id === pageId);
  const meta = META[locale][pageId];
  const cfg = NAV[locale];
  const assets = assetPrefix(page.file);
  const body =
    preparePageBody(locale, pageId, assets) +
    (pageId === "rules-acl-syntax" ? renderAiPrompt(locale, pageId, cfg) : "");
  const useUiMock = UI_MOCK_PAGE_IDS.has(pageId);
  const extraCss = useUiMock ? `\n  <link rel="stylesheet" href="${assets}ui-mock.css">` : "";
  const extraJs = useUiMock ? `\n<script src="${assets}ui-tour.js"></script>` : "";
  const mermaidJs = MERMAID_PAGE_IDS.has(pageId)
    ? `\n<script type="module">
import mermaid from "https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs";
mermaid.initialize({
  startOnLoad: true,
  theme: "dark",
  securityLevel: "loose",
  flowchart: { nodeSpacing: 48, rankSpacing: 56, padding: 16, htmlLabels: true },
});
</script>`
    : "";

  const langOther = locale === "ru" ? "en" : "ru";
  const langSwitch = locale === "ru"
    ? `<span class="text-zinc-500 border border-teal-400/30 text-teal-300 rounded px-2 py-0.5">RU</span>
          <a href="${langHref(locale, page.file, page.file)}" class="text-zinc-500 border border-white/8 rounded px-2 py-0.5 hover:text-zinc-200 hover:bg-white/5 transition">EN</a>`
    : `<a href="${langHref(locale, page.file, page.file)}" class="text-zinc-500 border border-white/8 rounded px-2 py-0.5 hover:text-zinc-200 hover:bg-white/5 transition">RU</a>
          <span class="text-zinc-500 border border-teal-400/30 text-teal-300 rounded px-2 py-0.5">EN</span>`;

  const hreflangRu = relHref(page.file, path.join("ru", page.file));
  const hreflangEn = relHref(page.file, path.join("en", page.file));

  return `<!DOCTYPE html>
<html lang="${locale}">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>${meta.pageTitle}</title>
  <link rel="icon" href="${assets}icon.svg" type="image/svg+xml">
  <link rel="alternate" hreflang="ru" href="${hreflangRu}">
  <link rel="alternate" hreflang="en" href="${hreflangEn}">
  <script src="https://cdn.tailwindcss.com"></script>
  <script>
  tailwind.config = {
    theme: {
      extend: {
        fontFamily: {
          sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
          mono: ['JetBrains Mono', 'ui-monospace', 'SFMono-Regular', 'monospace'],
        },
        colors: { ink: '#08090a' }
      }
    }
  }
  </script>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
  <link rel="stylesheet" href="${assets}docs.css">${extraCss}
</head>
<body class="bg-ink text-zinc-400 font-sans antialiased overflow-hidden" data-copy="${cfg.copy}" data-copied="${cfg.copied}">

<div class="flex h-screen">

  <aside class="w-[262px] flex-shrink-0 border-r border-white/5 flex flex-col bg-ink/50">
    <div class="h-14 flex items-center gap-2.5 px-5 border-b border-white/5">
      <img src="${assets}icon.svg" alt="" class="w-7 h-7 rounded-md border border-teal-400/25 bg-teal-400/10 p-0.5" width="28" height="28">
      <span class="text-zinc-200 text-sm font-medium tracking-tight">Oktopus</span>
      <span class="ml-auto text-[10px] font-mono text-zinc-600 border border-white/10 rounded px-1.5 py-0.5">docs</span>
    </div>

    <div class="px-3 pt-3 pb-1">
      <div class="flex items-center gap-2 border border-white/8 rounded-lg px-3 py-2 bg-white/[0.02] focus-within:border-teal-400/30 transition">
        <svg class="w-3.5 h-3.5 text-zinc-600 flex-shrink-0" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607Z"/></svg>
        <input id="nav-search" type="search" placeholder="${cfg.search}" class="bg-transparent text-[12.5px] text-zinc-300 w-full placeholder:text-zinc-600 font-mono outline-none">
      </div>
    </div>

    <nav id="nav" class="flex-1 px-3 py-3 overflow-y-auto flex flex-col gap-4">
${renderNav(locale, pageId, page.file)}
    </nav>

    <div class="border-t border-white/5 p-3">
      <a href="https://github.com/Camyil-89/oktopus" target="_blank" rel="noopener noreferrer"
         class="flex items-center gap-2.5 px-2 py-2 rounded-lg hover:bg-white/[0.03] transition group">
        <svg class="w-4 h-4 text-zinc-600 group-hover:text-zinc-400 transition" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true"><path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82a7.4 7.4 0 0 1 2-.27c.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z"/></svg>
        <div class="min-w-0">
          <p class="text-[12px] text-zinc-400 group-hover:text-zinc-200 transition leading-tight">Camyil-89/oktopus</p>
          <p class="text-[10.5px] font-mono text-zinc-600">${cfg.github}</p>
        </div>
      </a>
    </div>
  </aside>

  <div class="flex-1 flex flex-col min-w-0">
    <header class="h-14 flex-shrink-0 border-b border-white/5 flex items-center px-7 gap-3 bg-ink/40 backdrop-blur">
      <nav class="flex items-center gap-2 text-[12.5px] font-mono text-zinc-600 min-w-0" aria-label="Breadcrumb">
        <span class="text-zinc-700">docs</span>
        <span class="text-zinc-700">/</span>
        <span class="text-zinc-600">${locale}</span>
        <span class="text-zinc-700">/</span>
        <span class="text-zinc-600">${meta.section}</span>
        <span class="text-zinc-700">/</span>
        <span class="text-zinc-300 truncate">${meta.title}</span>
      </nav>
      <div class="ml-auto flex items-center gap-3">
        <div class="flex items-center gap-1 text-[12px] font-mono">${langSwitch}</div>
        <a href="https://github.com/Camyil-89/oktopus" target="_blank" rel="noopener noreferrer"
           class="hidden sm:inline-flex items-center gap-1.5 text-[12px] text-zinc-400 border border-white/8 rounded-lg px-3 py-1.5 hover:bg-white/5 hover:text-zinc-200 transition">GitHub</a>
      </div>
    </header>

    <main id="main" class="flex-1 overflow-y-auto grid-bg">
      <div class="max-w-[1180px] mx-auto px-7 py-9 flex gap-12">
        <div class="min-w-0 flex-1 max-w-[740px]">

          <article class="doc doc-page">
${body}
          </article>

${renderPager(locale, pageId, page.file)}

          <footer class="mt-10 pb-4 flex items-center justify-between text-[11px] font-mono text-zinc-700">
            <span>© 2026 Oktopus</span>
            <span>${cfg.footer}</span>
          </footer>
        </div>

        <aside class="hidden xl:block w-[196px] flex-shrink-0">
          <div class="sticky top-2">
            <p class="text-[10px] font-mono uppercase tracking-[0.16em] text-zinc-700 mb-3">${cfg.toc}</p>
            <ul id="toc-list" class="flex flex-col gap-0.5"></ul>
          </div>
        </aside>
      </div>
    </main>
  </div>
</div>

<script src="${assets}docs-page.js"></script>${extraJs}${mermaidJs}
</body>
</html>
`;
}

for (const locale of ["ru", "en"]) {
  for (const page of PAGES) {
    const out = path.join(__dirname, locale, page.file);
    fs.mkdirSync(path.dirname(out), { recursive: true });
    fs.writeFileSync(out, renderPage(locale, page.id), "utf8");
    console.log("wrote", path.relative(__dirname, out));
  }
}
