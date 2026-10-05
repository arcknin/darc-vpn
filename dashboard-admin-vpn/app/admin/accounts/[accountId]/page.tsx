import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { AccountDetail } from "../../../../src/components/AccountDetail";
import { SESSION_COOKIE } from "../../../../src/lib/server-api";

export default async function AccountPage({
  params,
}: {
  params: Promise<{ accountId: string }>;
}) {
  const store = await cookies();
  if (!store.get(SESSION_COOKIE)) redirect("/login");
  const { accountId } = await params;
  return <AccountDetail accountId={accountId} />;
}
