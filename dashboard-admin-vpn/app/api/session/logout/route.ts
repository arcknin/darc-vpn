import { NextResponse } from "next/server";
import {
  forwardToBackend,
  readSessionToken,
  SESSION_COOKIE,
  SESSION_COOKIE_OPTIONS,
  transportResponse,
} from "../../../../src/lib/server-api";

export async function POST(request: Request): Promise<NextResponse> {
  const transport = transportResponse(request);
  if (transport) return transport;
  const token = await readSessionToken();
  if (token) {
    try {
      await forwardToBackend("/api/v1/auth/logout", token, { method: "POST" });
    } catch {
      // A local session is always removed even when the upstream is unavailable.
    }
  }
  const response = new NextResponse(null, { status: 204 });
  response.cookies.set({
    name: SESSION_COOKIE,
    value: "",
    ...SESSION_COOKIE_OPTIONS,
    maxAge: 0,
  });
  return response;
}
