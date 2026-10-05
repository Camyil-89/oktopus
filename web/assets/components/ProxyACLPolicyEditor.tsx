"use client";

import {
  autocompletion,
  completionKeymap,
} from "@codemirror/autocomplete";
import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import {
  EditorState,
  RangeSetBuilder,
  StateEffect,
  StateField,
} from "@codemirror/state";
import {
  Decoration,
  type DecorationSet,
  EditorView,
  keymap,
  lineNumbers,
  placeholder as cmPlaceholder,
} from "@codemirror/view";
import { useEffect, useRef } from "react";
import type { ProxyACLPolicyDiagnostic } from "@/types/proxy";
import {
  type ACLListMeta,
  setSquidCompletionEnv,
  squidPolicyCompletions,
} from "@/assets/components/proxyAclCompletion";

const monoFont =
  'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace';

const setDiagnosticsEffect = StateEffect.define<ProxyACLPolicyDiagnostic[]>();
const setListNamesEffect = StateEffect.define<string[]>();

type ProxyACLPolicyEditorProps = {
  value?: string;
  onChange?: (value: string) => void;
  listNames?: string[];
  listMetas?: ACLListMeta[];
  diagnostics?: ProxyACLPolicyDiagnostic[];
  placeholder?: string;
  resetKey?: string;
};

const editorTheme = EditorView.theme({
  "&": {
    fontSize: "13px",
    border: "1px solid #30363d",
    borderRadius: "6px",
    backgroundColor: "#0d1117",
    color: "#e6edf3",
  },
  "&.cm-focused": {
    outline: "2px solid #1677ff55",
    borderColor: "#388bfd",
  },
  ".cm-content": { caretColor: "#e6edf3" },
  ".cm-line": { color: "#e6edf3" },
  ".cm-cursor": { borderLeftColor: "#e6edf3" },
  ".cm-selectionBackground, &.cm-focused .cm-selectionBackground": {
    backgroundColor: "#264f78",
  },
  ".cm-gutters": {
    backgroundColor: "#161b22",
    color: "#7d8590",
    borderRight: "1px solid #30363d",
  },
  ".cm-activeLineGutter": {
    backgroundColor: "#1c2128",
    color: "#e6edf3",
  },
  ".cm-activeLine": { backgroundColor: "#161b22" },
  ".cm-keyword": { color: "#ff7b72", fontWeight: 600 },
  ".cm-action": { color: "#79c0ff" },
  ".cm-unknown-acl": {
    textDecoration: "underline wavy #f85149",
    backgroundColor: "rgba(248, 81, 73, 0.12)",
  },
  ".cm-diag-error": {
    textDecoration: "underline wavy #f85149",
    backgroundColor: "rgba(248, 81, 73, 0.18)",
  },
  ".cm-tooltip.cm-tooltip-autocomplete": {
    backgroundColor: "#161b22",
    border: "1px solid #30363d",
    color: "#e6edf3",
  },
  ".cm-tooltip.cm-completionInfo": {
    backgroundColor: "#1c2128",
    border: "1px solid #30363d",
    color: "#e6edf3",
    padding: "8px 10px",
  },
  ".cm-completionIcon": {
    opacity: 0.85,
  },
  ".cm-completionMatchedText": {
    textDecoration: "underline",
    color: "#79c0ff",
  },
});

const keywordMark = Decoration.mark({ class: "cm-keyword" });
const actionMark = Decoration.mark({ class: "cm-action" });
const unknownMark = Decoration.mark({ class: "cm-unknown-acl" });
const diagErrorMark = Decoration.mark({ class: "cm-diag-error" });

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

function definedACLNames(text: string, listNames: string[]): Set<string> {
  const s = new Set(listNames);
  for (const raw of text.split("\n")) {
    const line = raw.trim();
    if (!line || line.startsWith("#")) continue;
    const f = splitFields(line);
    if (f[0]?.toLowerCase() === "acl" && f[1]) {
      s.add(f[1]);
    }
  }
  return s;
}

function fieldSpanColumns(
  raw: string,
  tokenIndex: number,
): { col: number; endCol: number } | null {
  const trimLeft = raw.length - raw.trimStart().length;
  const line = raw.trim();
  const spans: { start: number; end: number }[] = [];
  let cur = "";
  let start = -1;
  let inQuote = false;
  for (let i = 0; i < line.length; i++) {
    const c = line[i];
    if (c === '"') {
      if (!cur) start = i;
      inQuote = !inQuote;
      cur += c;
      continue;
    }
    if (!inQuote && /\s/.test(c)) {
      if (cur) {
        spans.push({ start, end: i });
        cur = "";
        start = -1;
      }
      continue;
    }
    if (!cur) start = i;
    cur += c;
  }
  if (cur) spans.push({ start, end: line.length });
  if (tokenIndex < 0 || tokenIndex >= spans.length) return null;
  const sp = spans[tokenIndex];
  return {
    col: sp.start + trimLeft + 1,
    endCol: sp.end + trimLeft + 1,
  };
}

function posAtLineCol(doc: string, line1: number, col1: number): number {
  const lines = doc.split("\n");
  if (line1 < 1 || line1 > lines.length) return 0;
  let pos = 0;
  for (let i = 0; i < line1 - 1; i++) {
    pos += lines[i].length + 1;
  }
  const line = lines[line1 - 1] ?? "";
  return pos + Math.min(Math.max(col1 - 1, 0), line.length);
}

type MarkSpan = { from: number; to: number; mark: Decoration };

function pushMark(spans: MarkSpan[], from: number, to: number, mark: Decoration) {
  if (to > from) {
    spans.push({ from, to, mark });
  }
}

function buildDecorations(
  doc: string,
  listNames: string[],
  diagnostics: ProxyACLPolicyDiagnostic[],
): DecorationSet {
  const spans: MarkSpan[] = [];
  const defined = definedACLNames(doc, listNames);
  const lines = doc.split("\n");

  for (let i = 0; i < lines.length; i++) {
    const raw = lines[i];
    const line = raw.trim();
    if (!line || line.startsWith("#")) continue;
    const fields = splitFields(line);
    if (fields.length === 0) continue;
    const kw = fields[0].toLowerCase();

    const kwSpan = fieldSpanColumns(raw, 0);
    if (kwSpan && (kw === "acl" || kw === "http_access")) {
      pushMark(
        spans,
        posAtLineCol(doc, i + 1, kwSpan.col),
        posAtLineCol(doc, i + 1, kwSpan.endCol),
        keywordMark,
      );
    }

    if (kw === "http_access" && fields[1]) {
      const act = fields[1].toLowerCase();
      const actSpan = fieldSpanColumns(raw, 1);
      if (actSpan && (act === "allow" || act === "deny")) {
        pushMark(
          spans,
          posAtLineCol(doc, i + 1, actSpan.col),
          posAtLineCol(doc, i + 1, actSpan.endCol),
          actionMark,
        );
      }
      for (let ti = 2; ti < fields.length; ti++) {
        let name = fields[ti];
        if (name.startsWith("!")) name = name.slice(1);
        if (!name || defined.has(name)) continue;
        const sp = fieldSpanColumns(raw, ti);
        if (!sp) continue;
        pushMark(
          spans,
          posAtLineCol(doc, i + 1, sp.col),
          posAtLineCol(doc, i + 1, sp.endCol),
          unknownMark,
        );
      }
    }
  }

  for (const d of diagnostics) {
    if (d.line < 1) continue;
    const col = d.column > 0 ? d.column : 1;
    const endCol = d.end_column && d.end_column > col ? d.end_column : col + 1;
    pushMark(
      spans,
      posAtLineCol(doc, d.line, col),
      posAtLineCol(doc, d.line, endCol),
      diagErrorMark,
    );
  }

  spans.sort((a, b) => a.from - b.from || a.to - b.to);

  const builder = new RangeSetBuilder<Decoration>();
  for (const s of spans) {
    builder.add(s.from, s.to, s.mark);
  }
  return builder.finish();
}

let decorationListNames: string[] = [];
let decorationDiagnostics: ProxyACLPolicyDiagnostic[] = [];

const policyDecorationsField = StateField.define<DecorationSet>({
  create(state) {
    return buildDecorations(
      state.doc.toString(),
      decorationListNames,
      decorationDiagnostics,
    );
  },
  update(deco, tr) {
    for (const e of tr.effects) {
      if (e.is(setListNamesEffect)) decorationListNames = e.value;
      if (e.is(setDiagnosticsEffect)) decorationDiagnostics = e.value;
    }
    if (
      tr.docChanged ||
      tr.effects.some(
        (e) => e.is(setListNamesEffect) || e.is(setDiagnosticsEffect),
      )
    ) {
      return buildDecorations(
        tr.newDoc.toString(),
        decorationListNames,
        decorationDiagnostics,
      );
    }
    return deco.map(tr.changes);
  },
  provide: (f) => EditorView.decorations.from(f),
});

export function ProxyACLPolicyEditor({
  value = "",
  onChange,
  listNames = [],
  listMetas = [],
  diagnostics = [],
  placeholder = "acl …\nhttp_access …",
  resetKey = "policy",
}: ProxyACLPolicyEditorProps) {
  const hostRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);
  const onChangeRef = useRef(onChange);
  const suppressOnChangeRef = useRef(false);
  const listNamesRef = useRef(listNames);
  const listMetasRef = useRef(listMetas);
  const diagnosticsRef = useRef(diagnostics);

  listNamesRef.current = listNames;
  listMetasRef.current = listMetas;
  diagnosticsRef.current = diagnostics;
  onChangeRef.current = onChange;

  useEffect(() => {
    const host = hostRef.current;
    if (!host) return;

    decorationListNames = listNamesRef.current;
    decorationDiagnostics = diagnosticsRef.current;
    setSquidCompletionEnv({ listMetas: listMetasRef.current });

    const state = EditorState.create({
      doc: value,
      extensions: [
        lineNumbers(),
        history(),
        autocompletion({
          override: [squidPolicyCompletions],
          activateOnTyping: true,
          maxRenderedOptions: 24,
          defaultKeymap: true,
          icons: true,
          closeOnBlur: true,
          tooltipClass: () => "oktopus-acl-completion-tooltip",
        }),
        keymap.of([
          ...completionKeymap,
          ...defaultKeymap,
          ...historyKeymap,
        ]),
        editorTheme,
        EditorView.theme({
          ".cm-scroller": {
            fontFamily: monoFont,
            lineHeight: "1.5",
            overflow: "auto",
            maxHeight: "min(60vh, 520px)",
            minHeight: "280px",
          },
        }),
        cmPlaceholder(placeholder),
        policyDecorationsField,
        EditorView.updateListener.of((update) => {
          if (update.docChanged && !suppressOnChangeRef.current) {
            onChangeRef.current?.(update.state.doc.toString());
            update.view.dispatch({
              effects: [
                setListNamesEffect.of(listNamesRef.current),
                setDiagnosticsEffect.of(diagnosticsRef.current),
              ],
            });
          }
        }),
      ],
    });

    const view = new EditorView({ state, parent: host });
    viewRef.current = view;

    return () => {
      view.destroy();
      viewRef.current = null;
    };
  }, [placeholder, resetKey]);

  useEffect(() => {
    const view = viewRef.current;
    if (!view) return;
    view.dispatch({
      effects: [
        setListNamesEffect.of(listNames),
        setDiagnosticsEffect.of(diagnostics),
      ],
    });
  }, [listNames, listMetas, diagnostics]);

  useEffect(() => {
    setSquidCompletionEnv({ listMetas });
  }, [listMetas]);

  useEffect(() => {
    const view = viewRef.current;
    if (!view) return;
    const current = view.state.doc.toString();
    if (value === current) return;
    suppressOnChangeRef.current = true;
    view.dispatch({
      changes: { from: 0, to: current.length, insert: value },
    });
    suppressOnChangeRef.current = false;
  }, [value]);

  return <div ref={hostRef} className="w-full" />;
}
