import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";

const [bundleArg, tagArg, repositoryArg] = process.argv.slice(2).filter((value) => value !== "--check");
const checkOnly = process.argv.includes("--check");
if (!bundleArg || !tagArg || !repositoryArg) {
  throw new Error("usage: generate-update-manifest.mjs <nsis-bundle-dir> <vX.Y.Z> <owner/repo> [--check]");
}
if (!/^v\d+\.\d+\.\d+$/.test(tagArg)) {
  throw new Error("Updater releases must use a stable tag such as v1.2.3; prerelease tags are not published to latest.json.");
}

const bundleDir = path.resolve(bundleArg);
const files = fs.readdirSync(bundleDir);
const archiveName =
  files.find((name) => name.endsWith(".nsis.zip") && files.includes(`${name}.sig`)) ??
  files.find((name) => name.endsWith("-setup.exe") && files.includes(`${name}.sig`));
if (!archiveName) throw new Error(`No signed NSIS updater payload found in ${bundleDir}`);
const archivePath = path.join(bundleDir, archiveName);
const signaturePath = `${archivePath}.sig`;
if (!fs.existsSync(signaturePath)) throw new Error(`Missing updater signature ${signaturePath}`);

const archive = fs.readFileSync(archivePath);
const signature = fs.readFileSync(signaturePath, "utf8").trim();
if (archive.length === 0 || signature.length === 0) throw new Error("Updater archive or signature is empty");
const version = tagArg.slice(1);
// Tauri's static updater protocol uses OS-architecture keys. NSIS describes
// the bundled installer artifact, not the manifest platform.
const tauriPlatform = "windows-x86_64";
const releaseNotes = process.env.CLOUD_SAVER_RELEASE_NOTES?.trim() || `Cloud Saver ${version}. See the GitHub release for complete notes.`;
const assetUrl = `https://github.com/${repositoryArg}/releases/download/${tagArg}/${encodeURIComponent(archiveName)}`;
const manifest = {
  version,
  notes: releaseNotes,
  pub_date: new Date().toISOString(),
  platforms: {
    [tauriPlatform]: {
      url: assetUrl,
      signature,
      sha256: crypto.createHash("sha256").update(archive).digest("hex"),
      size: archive.length,
    },
  },
};
const outputPath = path.resolve("latest.json");
const serialized = `${JSON.stringify(manifest, null, 2)}\n`;

if (checkOnly) {
  const existing = JSON.parse(fs.readFileSync(outputPath, "utf8"));
  const expected = manifest.platforms[tauriPlatform];
  const actual = existing.platforms?.[tauriPlatform];
  if (existing.version !== version || actual?.url !== expected.url || actual?.signature !== signature || actual?.sha256 !== expected.sha256 || actual?.size !== expected.size) {
    throw new Error("latest.json does not agree with the signed updater artifact");
  }
  console.log(`[update-manifest] verified ${archiveName} for ${tagArg}`);
} else {
  fs.writeFileSync(outputPath, serialized, "utf8");
  console.log(`[update-manifest] wrote ${path.relative(process.cwd(), outputPath)} for ${archiveName}`);
}
