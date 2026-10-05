import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { SESSION_COOKIE } from "../src/lib/server-api";

export default async function Home() {
  const store = await cookies();
  redirect(store.get(SESSION_COOKIE) ? "/admin" : "/login");
}
