import type {
  AdminAccountList,
  AdminAccountMetadata,
  AdminProfileMetadata,
  CreateAccountRequest,
  CreateProfileRequest,
  RevocationOperation,
} from "./openapi-types";
import { parseRevocationOperation } from "./status";

export class DashboardApiError extends Error {
  readonly status: number;
  readonly code?: string;

  constructor(message: string, status: number, code?: string) {
    super(message);
    this.name = "DashboardApiError";
    this.status = status;
    this.code = code;
  }
}

async function request<T>(
  path: string,
  init: RequestInit = {},
): Promise<T> {
  let response: Response;
  try {
    response = await fetch(path, {
      ...init,
      headers: {
        Accept: "application/json",
        ...(init.body ? { "Content-Type": "application/json" } : {}),
        ...init.headers,
      },
      cache: "no-store",
    });
  } catch {
    throw new DashboardApiError("serviço indisponível", 0);
  }

  const text = await response.text();
  let payload: unknown = undefined;
  if (text) {
    try {
      payload = JSON.parse(text) as unknown;
    } catch {
      payload = undefined;
    }
  }
  if (!response.ok) {
    const error = payload && typeof payload === "object" ? payload : {};
    const message =
      typeof (error as { message?: unknown }).message === "string"
        ? (error as { message: string }).message
        : "operação não concluída";
    const code =
      typeof (error as { code?: unknown }).code === "string"
        ? (error as { code: string }).code
        : undefined;
    throw new DashboardApiError(message, response.status, code);
  }
  if (response.status === 204 || payload === undefined) return undefined as T;
  return payload as T;
}

export const dashboardApi = {
  listAccounts(): Promise<AdminAccountList> {
    return request<AdminAccountList>("/api/admin/accounts");
  },

  getAccount(accountId: string): Promise<AdminAccountMetadata> {
    return request<AdminAccountMetadata>(
      `/api/admin/accounts/${encodeURIComponent(accountId)}`,
    );
  },

  createAccount(input: CreateAccountRequest): Promise<AdminAccountMetadata> {
    return request<AdminAccountMetadata>("/api/admin/accounts", {
      method: "POST",
      body: JSON.stringify(input),
    });
  },

  createProfile(
    accountId: string,
    input: CreateProfileRequest,
  ): Promise<AdminProfileMetadata> {
    return request<AdminProfileMetadata>(
      `/api/admin/accounts/${encodeURIComponent(accountId)}/profiles`,
      { method: "POST", body: JSON.stringify(input) },
    );
  },

  blockAccount(accountId: string): Promise<RevocationOperation> {
    return request<RevocationOperation>(
      `/api/admin/accounts/${encodeURIComponent(accountId)}/block`,
      { method: "POST" },
    ).then((value) => {
      const parsed = parseRevocationOperation(value);
      if (!parsed) {
        throw new DashboardApiError("resposta de revogação inválida", 502);
      }
      return parsed;
    });
  },

  getRevocation(accountId: string): Promise<RevocationOperation> {
    return request<RevocationOperation>(
      `/api/admin/accounts/${encodeURIComponent(accountId)}/revocation`,
    ).then((value) => {
      const parsed = parseRevocationOperation(value);
      if (!parsed) {
        throw new DashboardApiError("resposta de revogação inválida", 502);
      }
      return parsed;
    });
  },
};
