import { readdir, readFile } from "node:fs/promises";
import { join, relative } from "node:path";

const root = process.cwd();
const roots = ["src", "app"];
const sourceFiles = [];

async function collect(directory) {
  for (const entry of await readdir(join(root, directory), { withFileTypes: true })) {
    const path = join(directory, entry.name);
    if (entry.isDirectory()) await collect(path);
    else if (/\.(ts|tsx)$/.test(entry.name)) sourceFiles.push(path);
  }
}

for (const directory of roots) await collect(directory);

const violations = [];
for (const path of sourceFiles) {
  const source = await readFile(join(root, path), "utf8");
  if (/\bconsole\.(log|warn|error|info|debug)\s*\(/.test(source)) {
    violations.push(`${path}: logging is not allowed in the dashboard boundary`);
  }
  if (/\b(?:localStorage|sessionStorage)\b/.test(source)) {
    violations.push(`${path}: browser storage must not hold the admin session`);
  }
  if (/dashboard-vpn\.darcanix\.com/.test(source)) {
    violations.push(`${path}: dashboard FQDN was invented`);
  }
}

if (violations.length) {
  process.stderr.write(`${violations.join("\n")}\n`);
  process.exitCode = 1;
} else {
  process.stdout.write(`lint ok (${sourceFiles.map((path) => relative(root, path)).length} source files)\n`);
}
