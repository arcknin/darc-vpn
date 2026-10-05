import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { StatusBadge } from "../src/components/StatusBadge";
import { AccountForm } from "../src/components/AccountForm";

describe("dashboard components", () => {
  it.each(["pending", "failed", "completed"] as const)("renders API state %s verbatim", (status) => {
    const html = renderToStaticMarkup(
      <StatusBadge
        operation={{
          accountId: "account-1",
          status,
          peers: [],
          updatedAt: "2026-09-29T00:00:00Z",
        }}
      />,
    );
    expect(html).toContain(`data-revocation-status="${status}"`);
    expect(html).toContain(`>${status}<`);
  });

  it("keeps account creation limited to administrative fields", () => {
    const html = renderToStaticMarkup(<AccountForm onCreated={() => undefined} />);
    expect(html).toContain("Criar conta e perfil principal");
    for (const forbidden of ["privateKey", "presharedKey", "configuration", "QR"]) {
      expect(html.toLowerCase()).not.toContain(forbidden.toLowerCase());
    }
  });
});
