import { invoke } from "@tauri-apps/api/core";
import { listen, type UnlistenFn } from "@tauri-apps/api/event";

export type UpdateLifecycle =
  | "idle"
  | "checking"
  | "available"
  | "downloading"
  | "ready"
  | "installing"
  | "error"
  | "development";

export type AppUpdateState = {
  lifecycle: UpdateLifecycle;
  currentVersion: string;
  availableVersion?: string;
  notes?: string;
  downloadedBytes: number;
  totalBytes?: number;
  lastCheckedAt?: number;
  downloadedAt?: number;
  retryable: boolean;
  errorCode?: string;
  error?: string;
  automaticChecksEnabled: boolean;
  development: boolean;
};

export const getUpdateStatus = () => invoke<AppUpdateState>("update_status");

export const checkForAppUpdate = (force = false) =>
  invoke<AppUpdateState>("check_for_update", { force });

export const downloadAppUpdate = () =>
  invoke<AppUpdateState>("download_update");

export const installDownloadedUpdate = () =>
  invoke<AppUpdateState>("install_downloaded_update");

export const discardDownloadedUpdate = () =>
  invoke<AppUpdateState>("discard_downloaded_update");

export const setAutomaticUpdateChecks = (enabled: boolean) =>
  invoke<AppUpdateState>("set_automatic_update_checks", { enabled });

export function listenForUpdateProgress(
  handler: (state: AppUpdateState) => void,
): Promise<UnlistenFn> {
  return listen<AppUpdateState>("cloud-saver://update-progress", (event) => {
    handler(event.payload);
  });
}

export function updateActionLabel(state: AppUpdateState | null): string {
  if (!state) return "Updates";
  if (state.lifecycle === "downloading") {
    const percent = state.totalBytes
      ? Math.min(100, Math.round((state.downloadedBytes / state.totalBytes) * 100))
      : null;
    return percent === null ? "Downloading update" : `Downloading ${percent}%`;
  }
  if (state.lifecycle === "ready") return `Install v${state.availableVersion ?? "update"}`;
  if (state.lifecycle === "installing") return "Installing";
  if (state.lifecycle === "error") return "Update needs attention";
  if (state.lifecycle === "available") return `Update v${state.availableVersion ?? ""}`;
  return "Updates";
}
