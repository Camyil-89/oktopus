import { redirect } from "next/navigation";

export default function LegacyProxyPage() {
  redirect("/manage");
}
