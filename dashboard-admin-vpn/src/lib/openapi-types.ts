/**
 * This file is generated from back-vpn/api/openapi.yaml (OpenAPI 3.0.3).
 *
 * The dashboard intentionally models only the administration DTOs. The
 * private profile response is not part of this client type surface.
 */

export type AccountStatus = "active" | "blocked";
export type RevocationStatus = "pending" | "failed" | "completed";

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  accessToken: string;
  expiresAt: string;
  accountId: string;
  role: "friend" | "admin";
}

export interface ApiError {
  code: string;
  message: string;
  eventId?: string;
}

export interface AdminProfileMetadata {
  id: string;
  accountId: string;
  displayName: string;
  isPrimary: boolean;
  peerPublicKey: string;
  createdAt: string;
}

export interface AdminAccountMetadata {
  id: string;
  username: string;
  status: AccountStatus;
  createdAt: string;
  primaryProfileId: string;
  profileCount: number;
  profiles: AdminProfileMetadata[];
}

export interface AdminAccountList {
  accounts: AdminAccountMetadata[];
}

export interface CreateAccountRequest {
  username: string;
  password: string;
  primaryDisplayName: string;
}

export interface CreateProfileRequest {
  displayName: string;
}

export interface PeerRevocation {
  profileId: string;
  peerId: string;
  removed: boolean;
  error?: string;
}

export interface RevocationOperation {
  accountId: string;
  status: RevocationStatus;
  peers: PeerRevocation[];
  error?: string;
  updatedAt: string;
}

export interface OpenApiPaths {
  "/api/v1/auth/admin/login": {
    post: { request: LoginRequest; response: LoginResponse };
  };
  "/api/v1/auth/logout": { post: { response: void } };
  "/api/v1/admin/accounts": {
    get: { response: AdminAccountList };
    post: { request: CreateAccountRequest; response: AdminAccountMetadata };
  };
  "/api/v1/admin/accounts/{accountId}": {
    get: { response: AdminAccountMetadata };
  };
  "/api/v1/admin/accounts/{accountId}/profiles": {
    post: { request: CreateProfileRequest; response: AdminProfileMetadata };
  };
  "/api/v1/admin/accounts/{accountId}/block": {
    post: { response: RevocationOperation };
  };
  "/api/v1/admin/accounts/{accountId}/revocation": {
    get: { response: RevocationOperation };
  };
}
