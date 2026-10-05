import { test, expect } from "playwright/test";

// Run with a local Next server and a contract-compatible API. No production or
// real credentials are used by this scenario.
test.describe("admin dashboard", () => {
  test("anonymous users are sent to login", async ({ page }) => {
    await page.goto(`${process.env.BASE_URL ?? "http://127.0.0.1:3000"}/admin`);
    await expect(page).toHaveURL(/\/login$/);
    await expect(page.getByRole("heading", { name: "Administração" })).toBeVisible();
  });
});
