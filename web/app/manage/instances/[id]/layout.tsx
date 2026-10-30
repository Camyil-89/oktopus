/**
 * Static export (nginx image): one placeholder path so Next can pre-render shells.
 * Real instance IDs are resolved client-side; nginx falls back to index.html for other paths.
 */
export function generateStaticParams() {
  return [{ id: "_" }];
}

export default function InstanceIdLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return children;
}
