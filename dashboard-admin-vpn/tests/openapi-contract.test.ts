import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

const openapi = readFileSync(
  resolve(process.cwd(), "..", "back-vpn", "api", "openapi.yaml"),
  "utf8",
);

describe("dashboard OpenAPI boundary", () => {
  it("keeps every administrative operation from the backend contract", () => {
    for (const route of [
      "/api/v1/auth/admin/login:",
      "/api/v1/auth/logout:",
      "/api/v1/admin/accounts:",
      "/api/v1/admin/accounts/{accountId}:",
      "/api/v1/admin/accounts/{accountId}/profiles:",
      "/api/v1/admin/accounts/{accountId}/block:",
      "/api/v1/admin/accounts/{accountId}/revocation:",
    ]) {
      expect(openapi).toContain(route);
    }
  });

  it("does not add private profile fields to the administrative schema", () => {
    const start = openapi.indexOf("    AdminProfileMetadata:");
    const end = openapi.indexOf("    AdminAccountMetadata:", start);
    expect(start).toBeGreaterThanOrEqual(0);
    const adminProfileSchema = openapi.slice(start, end);
    for (const forbidden of ["privateKey", "presharedKey", "allowedIPs", "dns"]) {
      expect(adminProfileSchema).not.toContain(forbidden);
    }
  });

  it("does not treat the VPN endpoint as a dashboard host", () => {
    expect(openapi).toContain("endpoint:\n          type: string\n          enum: [vpn.darcanix.com]");
    expect(openapi).not.toContain("dashboard-vpn.darcanix.com");
  });
});
