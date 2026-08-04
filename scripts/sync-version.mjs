import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const argv = process.argv.slice(2);
const checkOnly = argv.includes("--check");
const dryRun = argv.includes("--dry-run");
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
  try {
    return fs.readFileSync(filePath, "utf8");
  } catch (error) {
    if (error?.code === "ENOENT") {
      exitWithError(`required version target is missing: ${path.relative(rootDir, filePath)}`);
    }
    throw error;
  }
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

  if (dryRun) return;

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

  if (dryRun) return;

  const updated = content.replace(versionPattern, `version = "${version}"`);
  writeText(filePath, updated);
}

const rootDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const normalizedVersion = normalizeVersion(sourceVersion);

updateJsonVersion(path.join(rootDir, "app", "package.json"), normalizedVersion);
updateJsonVersion(path.join(rootDir, "app", "package-lock.json"), normalizedVersion);
const packageLock = readJson(path.join(rootDir, "app", "package-lock.json"));
if (packageLock.packages?.[""]) {
  if (checkOnly && packageLock.packages[""].version !== normalizedVersion) {
    exitWithError(`app/package-lock.json root package has version '${packageLock.packages[""].version}', expected '${normalizedVersion}'`);
  }
  if (!checkOnly && !dryRun) {
    packageLock.packages[""].version = normalizedVersion;
    writeJson(path.join(rootDir, "app", "package-lock.json"), packageLock);
  }
}
updateJsonVersion(path.join(rootDir, "app", "src-tauri", "tauri.conf.json"), normalizedVersion);
updateCargoVersion(path.join(rootDir, "app", "src-tauri", "Cargo.toml"), normalizedVersion);

if (checkOnly) {
  console.log(`[sync-version] all version targets already match ${normalizedVersion}`);
} else if (dryRun) {
  console.log(`[sync-version] validated version targets for ${normalizedVersion} without writing files`);
} else {
  console.log(`[sync-version] synchronized version targets to ${normalizedVersion}`);
}
