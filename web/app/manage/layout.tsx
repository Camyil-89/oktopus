import { AuthGate } from "@/assets/components/AuthGate";
import { ManageShell } from "@/assets/components/ManageShell";

export default function ManageLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AuthGate>
      <ManageShell>{children}</ManageShell>
    </AuthGate>
  );
}
