import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

const rootDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const backendDir = path.join(rootDir, "backend");
const resourcesDir = path.join(rootDir, "app", "src-tauri", "resources", "backend");
const outputBinary = path.join(resourcesDir, process.platform === "win32" ? "cloud-saver-daemon.exe" : "cloud-saver-daemon");
const sourceTrayIcon = path.join(rootDir, "app", "src-tauri", "icons", "icon.ico");
const stagedTrayIcon = path.join(resourcesDir, "icon.ico");

fs.mkdirSync(resourcesDir, { recursive: true });

let oauthClientID = process.env.CLOUD_SAVER_GOOGLE_CLIENT_ID?.trim() ?? "";
let oauthClientSecret = process.env.CLOUD_SAVER_GOOGLE_CLIENT_SECRET?.trim() ?? "";
let oauthSource = oauthClientSecret ? "environment" : "none";
const oauthJSONPath = process.env.CLOUD_SAVER_GOOGLE_OAUTH_JSON?.trim();
if (oauthJSONPath && (!oauthClientID || !oauthClientSecret)) {
  const parsed = JSON.parse(fs.readFileSync(oauthJSONPath, "utf8"));
  const candidate = parsed.installed ?? parsed.web;
  if (!candidate) {
    throw new Error("OAuth JSON must contain an installed or web client");
  }
  oauthClientID ||= String(candidate.client_id ?? "").trim();
  oauthClientSecret ||= String(candidate.client_secret ?? "").trim();
  oauthSource = "oauth_json";
}

const args = ["build"];
const linkerValues = [];
if (oauthClientID) {
  linkerValues.push(`-X=cloud-saver/backend/internal/auth/googledrive.defaultClientID=${oauthClientID}`);
}
if (oauthClientSecret) {
  linkerValues.push(`-X=cloud-saver/backend/internal/auth/googledrive.defaultClientSecret=${oauthClientSecret}`);
}
if (linkerValues.length > 0) {
  args.push("-ldflags", linkerValues.join(" "));
}
args.push("-o", outputBinary, "./cmd/daemon");
const result = spawnSync("go", args, {
  cwd: backendDir,
  stdio: "inherit",
});

if (result.status !== 0) {
  process.exit(result.status ?? 1);
}

console.log(`[stage-backend-daemon] OAuth credential source: ${oauthSource}`);

if (fs.existsSync(sourceTrayIcon)) {
  fs.copyFileSync(sourceTrayIcon, stagedTrayIcon);
  console.log(`[stage-backend-daemon] staged tray icon at ${path.relative(rootDir, stagedTrayIcon)}`);
}

console.log(`[stage-backend-daemon] staged daemon binary at ${path.relative(rootDir, outputBinary)}`);
