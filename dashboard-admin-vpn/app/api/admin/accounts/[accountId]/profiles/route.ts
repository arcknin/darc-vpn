import { NextResponse } from "next/server";
import type { CreateProfileRequest } from "../../../../../../src/lib/openapi-types";
import {
  errorResponse,
  forwardResponse,
  forwardToBackend,
  requireAdmin,
  sanitizeAdminProfile,
} from "../../../../../../src/lib/server-api";

function isCreateProfileRequest(value: unknown): value is CreateProfileRequest {
  if (!value || typeof value !== "object") return false;
  const candidate = value as Partial<CreateProfileRequest>;
  return (
    typeof candidate.displayName === "string" &&
    candidate.displayName.trim().length > 0
  );
}

export async function POST(
  request: Request,
  context: { params: Promise<{ accountId: string }> },
): Promise<NextResponse> {
  const auth = await requireAdmin(request);
  if (auth instanceof NextResponse) return auth;
  const { accountId } = await context.params;
  let body: unknown;
  try {
    body = await request.json();
  } catch {
    return errorResponse(400, "invalid_request", "pedido inválido");
  }
  if (!accountId || !isCreateProfileRequest(body)) {
    return errorResponse(400, "invalid_request", "pedido inválido");
  }
  try {
    return await forwardResponse(
      await forwardToBackend(
        `/api/v1/admin/accounts/${encodeURIComponent(accountId)}/profiles`,
        auth.token,
        {
          method: "POST",
          body: JSON.stringify({ displayName: body.displayName }),
        },
      ),
      sanitizeAdminProfile,
    );
  } catch {
    return errorResponse(503, "service_unavailable", "serviço indisponível");
  }
}
