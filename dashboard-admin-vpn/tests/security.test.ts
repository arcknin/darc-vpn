import { describe, expect, it, afterEach } from "vitest";
import {
  getApiBaseUrl,
  sanitizeAdminAccount,
  SESSION_COOKIE,
  SESSION_COOKIE_OPTIONS,
} from "../src/lib/server-api";

const oldNodeEnv = process.env.NODE_ENV;
const oldLocalMode = process.env.DASHBOARD_LOCAL_MODE;
const oldApi = process.env.VPN_API_BASE_URL;
const env = process.env as Record<string, string | undefined>;

afterEach(() => {
  if (oldNodeEnv === undefined) delete env.NODE_ENV;
  else env.NODE_ENV = oldNodeEnv;
  if (oldLocalMode === undefined) delete env.DASHBOARD_LOCAL_MODE;
  else env.DASHBOARD_LOCAL_MODE = oldLocalMode;
  if (oldApi === undefined) delete env.VPN_API_BASE_URL;
  else env.VPN_API_BASE_URL = oldApi;
});

describe("dashboard session boundary", () => {
  it("uses a secure HttpOnly SameSite cookie", () => {
    expect(SESSION_COOKIE).toBe("vpn_admin_session");
    expect(SESSION_COOKIE_OPTIONS).toMatchObject({
      httpOnly: true,
      secure: true,
      sameSite: "lax",
      path: "/",
    });
  });

  it("rejects an HTTP API outside local mode", () => {
    env.NODE_ENV = "production";
    env.DASHBOARD_LOCAL_MODE = "false";
    env.VPN_API_BASE_URL = "http://api.example.invalid";
    expect(() => getApiBaseUrl()).toThrow(/HTTPS/);
  });

  it("allows loopback HTTP only for local development", () => {
    env.NODE_ENV = "development";
    env.DASHBOARD_LOCAL_MODE = "true";
    env.VPN_API_BASE_URL = "http://127.0.0.1:8080";
    expect(getApiBaseUrl()).toBe("http://127.0.0.1:8080");
  });

  it("sanitizes administrative responses before they reach the browser", () => {
    const safe = sanitizeAdminAccount({
      id: "account-1",
      username: "friend",
      status: "active",
      createdAt: "2026-09-29T00:00:00Z",
      primaryProfileId: "profile-1",
      profileCount: 1,
      profiles: [
        {
          id: "profile-1",
          accountId: "account-1",
          displayName: "Principal",
          isPrimary: true,
          peerPublicKey: "public",
          createdAt: "2026-09-29T00:00:00Z",
          privateKey: "must-not-cross-the-proxy",
          presharedKey: "must-not-cross-the-proxy",
        },
      ],
      privateKey: "must-not-cross-the-proxy",
    });
    expect(JSON.stringify(safe)).not.toContain("privateKey");
    expect(JSON.stringify(safe)).not.toContain("presharedKey");
    expect(safe?.profiles[0]?.displayName).toBe("Principal");
  });
});
