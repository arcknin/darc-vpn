import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { AccountList } from "../../src/components/AccountList";
import { SESSION_COOKIE } from "../../src/lib/server-api";

export default async function AdminPage() {
  const store = await cookies();
  if (!store.get(SESSION_COOKIE)) redirect("/login");
  return <AccountList />;
}
