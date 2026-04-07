import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

const rootDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const backendDir = path.join(rootDir, "backend");
const resourcesDir = path.join(rootDir, "app", "src-tauri", "resources", "backend");
const outputBinary = path.join(resourcesDir, process.platform === "win32" ? "cloud-saver-daemon.exe" : "cloud-saver-daemon");

fs.mkdirSync(resourcesDir, { recursive: true });

const args = ["build", "-o", outputBinary, "./cmd/daemon"];
const result = spawnSync("go", args, {
  cwd: backendDir,
  stdio: "inherit",
});

if (result.status !== 0) {
  process.exit(result.status ?? 1);
}

console.log(`[stage-backend-daemon] staged daemon binary at ${path.relative(rootDir, outputBinary)}`);
