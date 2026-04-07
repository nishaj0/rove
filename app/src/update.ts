import { getVersion } from "@tauri-apps/api/app";
import { FALLBACK_APP_VERSION, RELEASES_API_URL } from "./constants/updates";

export type AppUpdateInfo = {
  currentVersion: string;
  latestVersion: string;
  releaseUrl: string;
};

type LatestReleasePayload = {
  tag_name?: string;
  html_url?: string;
};

function normalizeVersion(input: string): string {
  const trimmed = input.trim().replace(/^v/i, "");
  const core = trimmed.split("-")[0] ?? "";
  return core;
}

function compareSemver(a: string, b: string): number {
  const aParts = normalizeVersion(a).split(".").map((value) => Number.parseInt(value, 10));
  const bParts = normalizeVersion(b).split(".").map((value) => Number.parseInt(value, 10));
  const maxLength = Math.max(aParts.length, bParts.length);

  for (let i = 0; i < maxLength; i += 1) {
    const left = Number.isFinite(aParts[i]) ? (aParts[i] as number) : 0;
    const right = Number.isFinite(bParts[i]) ? (bParts[i] as number) : 0;
    if (left > right) {
      return 1;
    }
    if (left < right) {
      return -1;
    }
  }

  return 0;
}

async function resolveCurrentVersion(): Promise<string> {
  try {
    const version = await getVersion();
    if (version.trim() !== "") {
      return version;
    }
  } catch {
    // keep fallback when running outside tauri context.
  }
  return FALLBACK_APP_VERSION;
}

export async function checkForAppUpdate(): Promise<AppUpdateInfo | null> {
  const currentVersion = await resolveCurrentVersion();

  try {
    const response = await fetch(RELEASES_API_URL, {
      method: "GET",
      headers: {
        Accept: "application/vnd.github+json",
      },
    });
    if (!response.ok) {
      return null;
    }

    const payload = (await response.json()) as LatestReleasePayload;
    const latestVersion = (payload.tag_name ?? "").trim();
    const releaseUrl = (payload.html_url ?? "").trim();
    if (latestVersion === "" || releaseUrl === "") {
      return null;
    }

    if (compareSemver(latestVersion, currentVersion) <= 0) {
      return null;
    }

    return {
      currentVersion,
      latestVersion,
      releaseUrl,
    };
  } catch {
    return null;
  }
}
