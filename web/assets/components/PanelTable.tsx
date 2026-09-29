"use client";

export function PanelTable({ children }: { children: React.ReactNode }) {
  return (
    <div className="panel-table overflow-hidden rounded-xl border border-white/10 bg-white/[0.05]">
      {children}
    </div>
  );
}
