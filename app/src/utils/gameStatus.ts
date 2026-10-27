import type { Game, RecoveryLibraryEntry } from "../types";

export type Tone = "ok" | "warn" | "err" | "busy" | "idle";

// Go serialises a zero time.Time as year 0001; treat that as "never".
function realTime(value?: string): number | null {
  if (!value) return null;
  const ms = Date.parse(value);
  return Number.isNaN(ms) || new Date(ms).getUTCFullYear() < 2000 ? null : ms;
}

export function relativeTime(value?: string, now = Date.now()): string {
  const ms = realTime(value);
  if (ms === null) return "never";
  const seconds = Math.max(0, Math.round((now - ms) / 1000));
  if (seconds < 45) return "just now";
  if (seconds < 3600) return `${Math.round(seconds / 60)} min ago`;
  if (seconds < 86400) return `${Math.round(seconds / 3600)} h ago`;
  return `${Math.round(seconds / 86400)} d ago`;
}

export function gameStatus(game: Game, now = Date.now()): { tone: Tone; text: string } {
  const health = game.protectionStatus;
  if (game.running) return { tone: "busy", text: "Playing now" };
  if (health.conflictCount > 0) return { tone: "err", text: "Needs your decision" };
  if (health.serviceControl.pauseScope !== "none" || !health.serviceControl.protectionEnabled) return { tone: "warn", text: "Backups paused" };
  if (health.pendingCount > 0) return { tone: "busy", text: "Backing up…" };
  const last = realTime(health.lastLocalProtection);
  return last === null ? { tone: "idle", text: "Not backed up yet" } : { tone: "ok", text: `Backed up ${relativeTime(health.lastLocalProtection, now)}` };
}

// A freshly restored game has no local revision yet, so the backend still lists
// its cloud profile as cloud-only; match on sync ID as well.
export function cloudOnlyEntries(library: RecoveryLibraryEntry[], games: Game[]): RecoveryLibraryEntry[] {
  const tracked = new Set(games.map((game) => game.syncId));
  return library.filter((entry) => !entry.gameId && !tracked.has(entry.syncId));
}

export function formatSize(bytes: number): string {
  if (bytes < 1024 * 1024) return `${Math.max(1, Math.round(bytes / 1024))} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
