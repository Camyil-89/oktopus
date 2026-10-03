#!/usr/bin/env node
/**
 * Bumps web/package.json semver. Usage: node scripts/bump-web-version.mjs patch|minor
 * Prints the new version to stdout.
 */
import { readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";

const kind = process.argv[2];
if (kind !== "patch" && kind !== "minor") {
  console.error("Usage: bump-web-version.mjs <patch|minor>");
  process.exit(1);
}

const pkgPath = join(process.cwd(), "web", "package.json");
const pkg = JSON.parse(readFileSync(pkgPath, "utf8"));
const match = /^(\d+)\.(\d+)\.(\d+)$/.exec(pkg.version);
if (!match) {
  console.error(`Invalid version in package.json: ${pkg.version}`);
  process.exit(1);
}

const major = Number(match[1]);
const minor = Number(match[2]);
const patch = Number(match[3]);

const next =
  kind === "minor"
    ? `${major}.${minor + 1}.0`
    : `${major}.${minor}.${patch + 1}`;

pkg.version = next;
writeFileSync(pkgPath, `${JSON.stringify(pkg, null, 2)}\n`, "utf8");
process.stdout.write(next);
