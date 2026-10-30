/**
 * Static export (nginx image): one placeholder path so Next can pre-render shells.
 * Real instance IDs come from the browser URL (`useManageInstanceId`); nginx serves this shell for other ids.
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
