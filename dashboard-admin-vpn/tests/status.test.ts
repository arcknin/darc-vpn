import { describe, expect, it } from "vitest";
import { parseRevocationOperation, statusDescription } from "../src/lib/status";

describe("revocation state fidelity", () => {
  it.each(["pending", "failed", "completed"] as const)("preserves %s", (status) => {
    const operation = parseRevocationOperation({
      accountId: "account-1",
      status,
      peers: [],
      updatedAt: "2026-09-29T00:00:00Z",
    });
    expect(operation?.status).toBe(status);
    expect(statusDescription(status)).toBe(status);
  });

  it("rejects a response without an approved state", () => {
    expect(
      parseRevocationOperation({
        accountId: "account-1",
        status: "success",
        peers: [],
        updatedAt: "2026-09-29T00:00:00Z",
      }),
    ).toBeNull();
  });
});
