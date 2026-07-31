import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

const rootDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const secretDir = path.join(rootDir, ".secrets");
const privateKeyPath = path.join(secretDir, "cloud-saver-updater.key");
const passwordPath = path.join(secretDir, "cloud-saver-updater-password.txt");
const publicKeyPath = path.join(rootDir, "app", "src-tauri", "updater-public-key.txt");
const tauriConfigPath = path.join(rootDir, "app", "src-tauri", "tauri.conf.json");
const cliPath = path.join(rootDir, "app", "node_modules", "@tauri-apps", "cli", "tauri.js");
const force = process.argv.includes("--force");

if (!fs.existsSync(cliPath)) {
  throw new Error("Install app dependencies with npm install before generating updater keys.");
}
if (!force && (fs.existsSync(privateKeyPath) || fs.existsSync(publicKeyPath))) {
  throw new Error("Updater signing material already exists. Use --force only for an intentional key rotation.");
}

fs.mkdirSync(secretDir, { recursive: true });
const password = crypto.randomBytes(36).toString("base64url");
const args = [cliPath, "signer", "generate", "--ci", "--password", password, "--write-keys", privateKeyPath];
if (force) args.push("--force");
const result = spawnSync(process.execPath, args, { cwd: path.join(rootDir, "app"), stdio: "inherit" });
if (result.status !== 0) process.exit(result.status ?? 1);

const generatedPublicKey = `${privateKeyPath}.pub`;
if (!fs.existsSync(generatedPublicKey)) {
  throw new Error(`Tauri did not generate the expected public key at ${generatedPublicKey}`);
}
fs.copyFileSync(generatedPublicKey, publicKeyPath);
const publicKey = fs.readFileSync(generatedPublicKey, "utf8").trim();
const tauriConfig = JSON.parse(fs.readFileSync(tauriConfigPath, "utf8"));
tauriConfig.plugins ??= {};
tauriConfig.plugins.updater ??= {};
tauriConfig.plugins.updater.pubkey = publicKey;
fs.writeFileSync(tauriConfigPath, `${JSON.stringify(tauriConfig, null, 2)}\n`, "utf8");
fs.rmSync(generatedPublicKey);
fs.writeFileSync(passwordPath, `${password}\n`, { encoding: "utf8", mode: 0o600 });

console.log("Updater signing key generated.");
console.log(`Public key (commit this): ${path.relative(rootDir, publicKeyPath)}`);
console.log(`Private key (GitHub secret TAURI_SIGNING_PRIVATE_KEY): ${path.relative(rootDir, privateKeyPath)}`);
console.log(`Password (GitHub secret TAURI_SIGNING_PRIVATE_KEY_PASSWORD): ${path.relative(rootDir, passwordPath)}`);
console.log("The .secrets directory is ignored by Git. Keep a secure backup; losing this key prevents updates to existing installs.");
