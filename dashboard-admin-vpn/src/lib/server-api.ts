import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import type {
  AdminAccountList,
  AdminAccountMetadata,
  AdminProfileMetadata,
  LoginResponse,
  PeerRevocation,
  RevocationOperation,
} from "./openapi-types";

export const SESSION_COOKIE = "vpn_admin_session";

export const SESSION_COOKIE_OPTIONS = {
  httpOnly: true,
  secure: true,
  sameSite: "lax" as const,
  path: "/",
  maxAge: 8 * 60 * 60,
};

function localMode(): boolean {
  return process.env.DASHBOARD_LOCAL_MODE === "true";
}

function isLoopback(hostname: string): boolean {
  return hostname === "localhost" || hostname === "127.0.0.1" || hostname === "[::1]";
}

export function getApiBaseUrl(): string {
  const raw = process.env.VPN_API_BASE_URL?.trim();
  if (!raw) throw new Error("API não configurada");
  let parsed: URL;
  try {
    parsed = new URL(raw);
  } catch {
    throw new Error("URL da API inválida");
  }
  const isDevelopment = process.env.NODE_ENV !== "production";
  const localHttpAllowed = (localMode() || isDevelopment) && isLoopback(parsed.hostname);
  if (parsed.protocol !== "https:" && !localHttpAllowed) {
    throw new Error("A API exige HTTPS fora do modo local");
  }
  if (parsed.protocol !== "https:" && parsed.protocol !== "http:") {
    throw new Error("Protocolo da API inválido");
  }
  return parsed.toString().replace(/\/$/, "");
}

export function secureRequest(request: Request): boolean {
  if (localMode() || process.env.NODE_ENV !== "production") return true;
  const forwarded = request.headers.get("x-forwarded-proto");
  if (forwarded) return forwarded.split(",", 1)[0]?.trim() === "https";
  return new URL(request.url).protocol === "https:";
}

export async function readSessionToken(): Promise<string | null> {
  const store = await cookies();
  return store.get(SESSION_COOKIE)?.value ?? null;
}

export function errorResponse(
  status: number,
  code: string,
  message: string,
): NextResponse {
  return NextResponse.json({ code, message }, { status });
}

export function transportResponse(request: Request): NextResponse | null {
  if (secureRequest(request)) return null;
  return errorResponse(400, "invalid_request", "transporte HTTPS necessário");
}

export async function forwardToBackend(
  path: string,
  token: string | null,
  init: RequestInit = {},
): Promise<Response> {
  const headers = new Headers(init.headers);
  headers.set("Accept", "application/json");
  if (token) headers.set("Authorization", `Bearer ${token}`);
  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  return fetch(`${getApiBaseUrl()}${path}`, {
    ...init,
    headers,
    cache: "no-store",
  });
}

export async function forwardResponse(
  response: Response,
  sanitize?: (value: unknown) => unknown | null,
): Promise<NextResponse> {
  const body = await response.text();
  if (response.ok && sanitize && body) {
    let payload: unknown;
    try {
      payload = JSON.parse(body) as unknown;
    } catch {
      return errorResponse(502, "invalid_request", "resposta da API inválida");
    }
    const safePayload = sanitize(payload);
    if (safePayload === null) {
      return errorResponse(502, "invalid_request", "resposta da API inválida");
    }
    return NextResponse.json(safePayload, { status: response.status });
  }
  const headers = new Headers();
  const contentType = response.headers.get("content-type");
  if (contentType) headers.set("content-type", contentType);
  return new NextResponse(body || null, { status: response.status, headers });
}

function stringField(value: unknown): string | null {
  return typeof value === "string" ? value : null;
}

function booleanField(value: unknown): boolean | null {
  return typeof value === "boolean" ? value : null;
}

function numberField(value: unknown): number | null {
  return typeof value === "number" && Number.isFinite(value) ? value : null;
}

export function sanitizeAdminProfile(value: unknown): AdminProfileMetadata | null {
  if (!value || typeof value !== "object") return null;
  const candidate = value as Record<string, unknown>;
  const id = stringField(candidate.id);
  const accountId = stringField(candidate.accountId);
  const displayName = stringField(candidate.displayName);
  const isPrimary = booleanField(candidate.isPrimary);
  const peerPublicKey = stringField(candidate.peerPublicKey);
  const createdAt = stringField(candidate.createdAt);
  if (!id || !accountId || !displayName || isPrimary === null || !peerPublicKey || !createdAt) {
    return null;
  }
  return { id, accountId, displayName, isPrimary, peerPublicKey, createdAt };
}

export function sanitizeAdminAccount(value: unknown): AdminAccountMetadata | null {
  if (!value || typeof value !== "object") return null;
  const candidate = value as Record<string, unknown>;
  const id = stringField(candidate.id);
  const username = stringField(candidate.username);
  const status = candidate.status === "active" || candidate.status === "blocked" ? candidate.status : null;
  const createdAt = stringField(candidate.createdAt);
  const primaryProfileId = stringField(candidate.primaryProfileId);
  const profileCount = numberField(candidate.profileCount);
  const profiles = Array.isArray(candidate.profiles)
    ? candidate.profiles.map(sanitizeAdminProfile)
    : null;
  if (
    !id ||
    !username ||
    !status ||
    !createdAt ||
    !primaryProfileId ||
    profileCount === null ||
    !profiles ||
    profiles.some((profile): profile is null => profile === null)
  ) {
    return null;
  }
  return {
    id,
    username,
    status,
    createdAt,
    primaryProfileId,
    profileCount,
    profiles: profiles as AdminProfileMetadata[],
  };
}

export function sanitizeAdminAccountList(value: unknown): AdminAccountList | null {
  if (!value || typeof value !== "object") return null;
  const accounts = (value as Record<string, unknown>).accounts;
  if (!Array.isArray(accounts)) return null;
  const safeAccounts = accounts.map(sanitizeAdminAccount);
  if (safeAccounts.some((account): account is null => account === null)) return null;
  return { accounts: safeAccounts as AdminAccountMetadata[] };
}

function sanitizePeer(value: unknown): PeerRevocation | null {
  if (!value || typeof value !== "object") return null;
  const candidate = value as Record<string, unknown>;
  const profileId = stringField(candidate.profileId);
  const peerId = stringField(candidate.peerId);
  const removed = booleanField(candidate.removed);
  if (!profileId || !peerId || removed === null) return null;
  const error = candidate.error === undefined ? undefined : stringField(candidate.error);
  if (candidate.error !== undefined && error === null) return null;
  return { profileId, peerId, removed, ...(error ? { error } : {}) };
}

export function sanitizeRevocation(value: unknown): RevocationOperation | null {
  if (!value || typeof value !== "object") return null;
  const candidate = value as Record<string, unknown>;
  const accountId = stringField(candidate.accountId);
  const status =
    candidate.status === "pending" || candidate.status === "failed" || candidate.status === "completed"
      ? candidate.status
      : null;
  const updatedAt = stringField(candidate.updatedAt);
  const peers = Array.isArray(candidate.peers) ? candidate.peers.map(sanitizePeer) : null;
  if (!accountId || !status || !updatedAt || !peers || peers.some((peer): peer is null => peer === null)) {
    return null;
  }
  const error = candidate.error === undefined ? undefined : stringField(candidate.error);
  if (candidate.error !== undefined && error === null) return null;
  return {
    accountId,
    status,
    updatedAt,
    peers: peers as PeerRevocation[],
    ...(error ? { error } : {}),
  };
}

export async function requireAdmin(
  request: Request,
): Promise<{ token: string } | NextResponse> {
  const transport = transportResponse(request);
  if (transport) return transport;
  const token = await readSessionToken();
  if (!token) return errorResponse(401, "not_authenticated", "autenticação necessária");
  return { token };
}

export function isAdminLoginResponse(value: unknown): value is LoginResponse {
  if (!value || typeof value !== "object") return false;
  const candidate = value as Partial<LoginResponse>;
  return (
    typeof candidate.accessToken === "string" &&
    typeof candidate.expiresAt === "string" &&
    typeof candidate.accountId === "string" &&
    candidate.role === "admin"
  );
}
