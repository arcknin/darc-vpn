import { NextResponse } from "next/server";
import {
  errorResponse,
  forwardResponse,
  forwardToBackend,
  requireAdmin,
  sanitizeAdminAccount,
  sanitizeAdminAccountList,
} from "../../../../src/lib/server-api";
import type { CreateAccountRequest } from "../../../../src/lib/openapi-types";

function isCreateAccountRequest(value: unknown): value is CreateAccountRequest {
  if (!value || typeof value !== "object") return false;
  const candidate = value as Partial<CreateAccountRequest>;
  return (
    typeof candidate.username === "string" &&
    candidate.username.trim().length > 0 &&
    typeof candidate.password === "string" &&
    candidate.password.length > 0 &&
    typeof candidate.primaryDisplayName === "string" &&
    candidate.primaryDisplayName.trim().length > 0
  );
}

async function forwardAccounts(request: Request, init: RequestInit): Promise<NextResponse> {
  const auth = await requireAdmin(request);
  if (auth instanceof NextResponse) return auth;
  try {
    return await forwardResponse(
      await forwardToBackend("/api/v1/admin/accounts", auth.token, init),
      request.method === "GET" ? sanitizeAdminAccountList : sanitizeAdminAccount,
    );
  } catch {
    return errorResponse(503, "service_unavailable", "serviço indisponível");
  }
}

export function GET(request: Request): Promise<NextResponse> {
  return forwardAccounts(request, { method: "GET" });
}

export async function POST(request: Request): Promise<NextResponse> {
  let body: unknown;
  try {
    body = await request.json();
  } catch {
    return errorResponse(400, "invalid_request", "pedido inválido");
  }
  if (!isCreateAccountRequest(body)) {
    return errorResponse(400, "invalid_request", "pedido inválido");
  }
  return forwardAccounts(request, {
    method: "POST",
    body: JSON.stringify({
      username: body.username,
      password: body.password,
      primaryDisplayName: body.primaryDisplayName,
    }),
  });
}
