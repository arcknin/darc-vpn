import { existsSync } from "node:fs";
import { join } from "node:path";
import { spawnSync } from "node:child_process";

const executable = join(process.cwd(), "node_modules", ".bin", "playwright");
if (!existsSync(executable)) {
  process.stdout.write("Playwright indisponível neste ambiente; cenário E2E não executado.\n");
  process.exit(0);
}

const result = spawnSync(executable, ["test", "tests/playwright"], {
  stdio: "inherit",
  env: process.env,
});
process.exit(result.status ?? 1);
