"use client";

import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import { EditorState } from "@codemirror/state";
import {
  EditorView,
  keymap,
  lineNumbers,
  placeholder as cmPlaceholder,
} from "@codemirror/view";
import { MODAL_TEMPLATE_FIELD_HEIGHT_CSS } from "@/assets/modals/modalConfig";
import { useEffect, useRef } from "react";

const monoFont =
  'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace';

type ProxyACLPatternEditorProps = {
  value?: string;
  onChange?: (value: string) => void;
  onLineCountChange?: (lines: number) => void;
  readOnly?: boolean;
  /** readOnly + визуально как disabled Input (прозрачный фон, приглушённый текст) */
  readOnlyMuted?: boolean;
  placeholder?: string;
  /** Высота под body модалки — скролл внутри редактора */
  fitModalBody?: boolean;
  /** Новый ключ — пересоздать редактор с актуальным doc */
  resetKey?: string;
};

function editorScrollerTheme(fitModalBody: boolean) {
  return EditorView.theme({
    ".cm-scroller": {
      fontFamily: monoFont,
      lineHeight: "1.5",
      overflow: "auto",
      ...(fitModalBody
        ? {
            height: MODAL_TEMPLATE_FIELD_HEIGHT_CSS,
            maxHeight: MODAL_TEMPLATE_FIELD_HEIGHT_CSS,
            minHeight: MODAL_TEMPLATE_FIELD_HEIGHT_CSS,
          }
        : {
            maxHeight: "min(60vh, 480px)",
            minHeight: "200px",
          }),
    },
  });
}

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
  ".cm-content": {
    caretColor: "#e6edf3",
  },
  ".cm-line": {
    color: "#e6edf3",
  },
  ".cm-cursor": {
    borderLeftColor: "#e6edf3",
  },
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
  ".cm-activeLine": {
    backgroundColor: "#161b22",
  },
  "&.cm-readonly .cm-scroller": {
    backgroundColor: "#0d1117",
  },
  ".cm-placeholder": {
    color: "#7d8590",
  },
});

const readOnlyMutedTheme = EditorView.theme({
  "&": {
    backgroundColor: "transparent",
    borderColor: "rgba(255,255,255,0.08)",
    cursor: "not-allowed",
  },
  "&.cm-focused": {
    outline: "none",
    borderColor: "rgba(255,255,255,0.08)",
  },
  ".cm-scroller": {
    backgroundColor: "transparent",
    cursor: "not-allowed",
  },
  ".cm-content": {
    caretColor: "transparent",
    cursor: "not-allowed",
  },
  ".cm-line": {
    color: "rgba(228,228,231,0.45)",
  },
  ".cm-cursor": {
    display: "none",
  },
  ".cm-gutters": {
    backgroundColor: "transparent",
    color: "rgba(148,148,156,0.5)",
    borderRight: "1px solid rgba(255,255,255,0.06)",
  },
  ".cm-activeLineGutter": {
    backgroundColor: "transparent",
    color: "rgba(148,148,156,0.55)",
  },
  ".cm-activeLine": {
    backgroundColor: "transparent",
  },
  ".cm-selectionBackground, &.cm-focused .cm-selectionBackground": {
    backgroundColor: "transparent",
  },
  ".cm-placeholder": {
    color: "rgba(113,113,122,0.65)",
  },
});

export function countPatternLines(text: string): number {
  if (!text) {
    return 0;
  }
  let lines = 1;
  for (let i = 0; i < text.length; i++) {
    if (text.charCodeAt(i) === 10) {
      lines += 1;
    }
  }
  return lines;
}

export function ProxyACLPatternEditor({
  value = "",
  onChange,
  onLineCountChange,
  readOnly = false,
  readOnlyMuted = false,
  placeholder = "example.com\n*.evil.com",
  fitModalBody = false,
  resetKey = "default",
}: ProxyACLPatternEditorProps) {
  const hostRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);
  const onChangeRef = useRef(onChange);
  const onLineCountRef = useRef(onLineCountChange);
  const suppressFormEchoRef = useRef(false);

  onChangeRef.current = onChange;
  onLineCountRef.current = onLineCountChange;

  useEffect(() => {
    const host = hostRef.current;
    if (!host) {
      return;
    }

    const extensions = [
        lineNumbers(),
        history(),
        keymap.of([...defaultKeymap, ...historyKeymap]),
        editorTheme,
        editorScrollerTheme(fitModalBody),
        cmPlaceholder(placeholder),
        EditorView.editable.of(!readOnly),
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            if (!suppressFormEchoRef.current) {
              onChangeRef.current?.(update.state.doc.toString());
            }
            onLineCountRef.current?.(update.state.doc.lines);
          }
        }),
    ];
    if (readOnly && readOnlyMuted) {
      extensions.push(readOnlyMutedTheme);
    }

    const state = EditorState.create({
      doc: value,
      extensions,
    });

    const view = new EditorView({ state, parent: host });
    viewRef.current = view;
    onLineCountRef.current?.(view.state.doc.lines);

    return () => {
      view.destroy();
      viewRef.current = null;
    };
  }, [readOnly, readOnlyMuted, placeholder, resetKey, fitModalBody]);

  useEffect(() => {
    const view = viewRef.current;
    if (!view) {
      return;
    }
    const current = view.state.doc.toString();
    if (value === current) {
      return;
    }
    suppressFormEchoRef.current = true;
    view.dispatch({
      changes: { from: 0, to: current.length, insert: value },
    });
    suppressFormEchoRef.current = false;
    onLineCountRef.current?.(view.state.doc.lines);
  }, [value]);

  return <div ref={hostRef} className="w-full" />;
}
