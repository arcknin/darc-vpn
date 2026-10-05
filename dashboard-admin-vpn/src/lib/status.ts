import type { RevocationOperation, RevocationStatus } from "./openapi-types";

export const REVOCATION_STATUSES: readonly RevocationStatus[] = [
  "pending",
  "failed",
  "completed",
];

export function isRevocationStatus(value: unknown): value is RevocationStatus {
  return (
    typeof value === "string" &&
    REVOCATION_STATUSES.includes(value as RevocationStatus)
  );
}

export function parseRevocationOperation(
  value: unknown,
): RevocationOperation | null {
  if (!value || typeof value !== "object") return null;
  const candidate = value as Partial<RevocationOperation>;
  if (
    typeof candidate.accountId !== "string" ||
    !isRevocationStatus(candidate.status) ||
    !Array.isArray(candidate.peers) ||
    typeof candidate.updatedAt !== "string"
  ) {
    return null;
  }
  return candidate as RevocationOperation;
}

export function statusDescription(status: RevocationStatus): string {
  return status;
}
