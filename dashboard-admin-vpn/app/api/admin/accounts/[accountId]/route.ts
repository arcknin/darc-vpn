import { NextResponse } from "next/server";
import {
  errorResponse,
  forwardResponse,
  forwardToBackend,
  requireAdmin,
  sanitizeAdminAccount,
} from "../../../../../src/lib/server-api";

export async function GET(
  request: Request,
  context: { params: Promise<{ accountId: string }> },
): Promise<NextResponse> {
  const auth = await requireAdmin(request);
  if (auth instanceof NextResponse) return auth;
  const { accountId } = await context.params;
  if (!accountId) return errorResponse(400, "invalid_request", "pedido inválido");
  try {
    return await forwardResponse(
      await forwardToBackend(
        `/api/v1/admin/accounts/${encodeURIComponent(accountId)}`,
        auth.token,
      ),
      sanitizeAdminAccount,
    );
  } catch {
    return errorResponse(503, "service_unavailable", "serviço indisponível");
  }
}
