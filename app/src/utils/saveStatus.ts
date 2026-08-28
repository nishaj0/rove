import type { ProviderHealth, ServiceControlState } from "../types";

export type SaveStatus = {
  label: string;
  detail: string;
  tone: "success" | "warning" | "error" | "syncing";
  action: "none" | "settings" | "review";
};

type SaveStatusInput = {
  provider?: ProviderHealth;
  serviceControl?: ServiceControlState;
  conflictCount?: number;
  pendingCount?: number;
  active?: boolean;
};

export function getSaveStatus({ provider, serviceControl, conflictCount = 0, pendingCount = 0, active = false }: SaveStatusInput): SaveStatus {
  if (conflictCount > 0) {
    return {
      label: "Choose which saved version to keep",
      detail: "Your saves are safe until you decide which version to use.",
      tone: "error",
      action: "review",
    };
  }

  if (serviceControl && (!serviceControl.protectionEnabled || serviceControl.pauseScope !== "none")) {
    return {
      label: "Backups are paused",
      detail: "Your current saves stay on this PC until backups are turned back on.",
      tone: "warning",
      action: "settings",
    };
  }

  if (provider && ["offline", "authentication_required", "rate_limited", "quota_full", "provider_unavailable", "remote_data_problem"].includes(provider.state)) {
    return {
      label: "Cloud backup needs attention",
      detail: "Your saves are still safe on this PC. Open Settings to reconnect or choose another save location.",
      tone: "warning",
      action: "settings",
    };
  }

  if (active || pendingCount > 0 || provider?.state === "reconnecting") {
    return {
      label: "Backing up saves",
      detail: "Rove is safely preparing your latest saves.",
      tone: "syncing",
      action: "none",
    };
  }

  if (provider?.state === "local_only") {
    return {
      label: "Saved on this PC",
      detail: "Rove is keeping a local copy of your saves on this computer.",
      tone: "success",
      action: "none",
    };
  }

  return {
    label: "Saves are backed up",
    detail: "Rove keeps your saves protected before and after you play.",
    tone: "success",
    action: "none",
  };
}
