import { NextResponse } from "next/server";
import type { LoginRequest } from "../../../../src/lib/openapi-types";
import {
  errorResponse,
  forwardToBackend,
  isAdminLoginResponse,
  SESSION_COOKIE,
  SESSION_COOKIE_OPTIONS,
  transportResponse,
} from "../../../../src/lib/server-api";

function isLoginRequest(value: unknown): value is LoginRequest {
  if (!value || typeof value !== "object") return false;
  const candidate = value as Partial<LoginRequest>;
  return (
    typeof candidate.username === "string" &&
    candidate.username.trim().length > 0 &&
    typeof candidate.password === "string" &&
    candidate.password.length > 0
  );
}

export async function POST(request: Request): Promise<NextResponse> {
  const transport = transportResponse(request);
  if (transport) return transport;

  let body: unknown;
  try {
    body = await request.json();
  } catch {
    return errorResponse(400, "invalid_request", "pedido inválido");
  }
  if (!isLoginRequest(body)) {
    return errorResponse(400, "invalid_request", "pedido inválido");
  }

  let upstream: Response;
  try {
    upstream = await forwardToBackend("/api/v1/auth/admin/login", null, {
      method: "POST",
      body: JSON.stringify({ username: body.username, password: body.password }),
    });
  } catch {
    return errorResponse(503, "service_unavailable", "serviço indisponível");
  }

  const text = await upstream.text();
  let payload: unknown;
  try {
    payload = text ? (JSON.parse(text) as unknown) : undefined;
  } catch {
    payload = undefined;
  }

  if (!upstream.ok) {
    const response = NextResponse.json(payload ?? { code: "invalid_credentials", message: "credenciais inválidas" }, {
      status: upstream.status,
    });
    return response;
  }

  if (!isAdminLoginResponse(payload)) {
    return errorResponse(502, "invalid_request", "resposta de autenticação inválida");
  }

  const response = NextResponse.json(
    {
      accountId: payload.accountId,
      role: payload.role,
      expiresAt: payload.expiresAt,
    },
    { status: 200 },
  );
  response.cookies.set({
    name: SESSION_COOKIE,
    value: payload.accessToken,
    ...SESSION_COOKIE_OPTIONS,
  });
  return response;
}
