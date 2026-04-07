import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const argv = process.argv.slice(2);
const checkOnly = argv.includes("--check");
const explicitVersionArg = argv.find((arg) => !arg.startsWith("--"));
const sourceVersion =
  explicitVersionArg ?? process.env.CLOUD_SAVER_VERSION ?? process.env.GITHUB_REF_NAME ?? "";

function exitWithError(message) {
  console.error(`[sync-version] ${message}`);
  process.exit(1);
}

function normalizeVersion(input) {
  const trimmed = String(input).trim();
  const match = trimmed.match(/^v?(\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?)$/);
  if (!match) {
    exitWithError(
      "expected version tag like v1.2.3 or v1.2.3-beta.1 via arg, CLOUD_SAVER_VERSION, or GITHUB_REF_NAME",
    );
  }
  return match[1];
}

function readText(filePath) {
  return fs.readFileSync(filePath, "utf8");
}

function writeText(filePath, content) {
  fs.writeFileSync(filePath, content, "utf8");
}

function readJson(filePath) {
  return JSON.parse(readText(filePath));
}

function writeJson(filePath, payload) {
  writeText(filePath, `${JSON.stringify(payload, null, 2)}\n`);
}

function updateJsonVersion(filePath, version) {
  const payload = readJson(filePath);
  const current = typeof payload.version === "string" ? payload.version : "";

  if (checkOnly) {
    if (current !== version) {
      exitWithError(`${path.relative(rootDir, filePath)} has version '${current}', expected '${version}'`);
    }
    return;
  }

  payload.version = version;
  writeJson(filePath, payload);
}

function updateCargoVersion(filePath, version) {
  const content = readText(filePath);
  const versionPattern = /^version\s*=\s*"([^"]+)"/m;
  const match = content.match(versionPattern);

  if (!match) {
    exitWithError(`could not find Cargo version field in ${path.relative(rootDir, filePath)}`);
  }

  const current = match[1];
  if (checkOnly) {
    if (current !== version) {
      exitWithError(`${path.relative(rootDir, filePath)} has version '${current}', expected '${version}'`);
    }
    return;
  }

  const updated = content.replace(versionPattern, `version = "${version}"`);
  writeText(filePath, updated);
}

function updateFallbackVersionConstant(filePath, version) {
  const content = readText(filePath);
  const pattern = /^export const FALLBACK_APP_VERSION = "([^"]+)";$/m;
  const match = content.match(pattern);

  if (!match) {
    exitWithError(`could not find FALLBACK_APP_VERSION in ${path.relative(rootDir, filePath)}`);
  }

  const current = match[1];
  if (checkOnly) {
    if (current !== version) {
      exitWithError(`${path.relative(rootDir, filePath)} has fallback '${current}', expected '${version}'`);
    }
    return;
  }

  const updated = content.replace(pattern, `export const FALLBACK_APP_VERSION = "${version}";`);
  writeText(filePath, updated);
}

const rootDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const normalizedVersion = normalizeVersion(sourceVersion);

updateJsonVersion(path.join(rootDir, "app", "package.json"), normalizedVersion);
updateJsonVersion(path.join(rootDir, "app", "src-tauri", "tauri.conf.json"), normalizedVersion);
updateCargoVersion(path.join(rootDir, "app", "src-tauri", "Cargo.toml"), normalizedVersion);
updateFallbackVersionConstant(path.join(rootDir, "app", "src", "constants", "updates.ts"), normalizedVersion);

if (checkOnly) {
  console.log(`[sync-version] all version targets already match ${normalizedVersion}`);
} else {
  console.log(`[sync-version] synchronized version targets to ${normalizedVersion}`);
}
