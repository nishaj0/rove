import { describe, expect, it } from "vitest";
import {
  normalizeGames,
  normalizeLogs,
  normalizeSettings,
  normalizeSnapshots,
  normalizeStatus,
  normalizeSyncConflicts,
  normalizeSyncJobs,
} from "./parsers";
import { newDefaultSettings } from "../constants/defaults";

describe("API response parsers", () => {
  it("return safe empty collections for malformed collection payloads", () => {
    expect(normalizeGames(null)).toEqual([]);
    expect(normalizeSnapshots("not an array")).toEqual([]);
    expect(normalizeLogs(undefined)).toEqual([]);
    expect(normalizeSyncJobs({})).toEqual([]);
    expect(normalizeSyncConflicts(null)).toEqual([]);
  });

  it("normalizes a game and nested protection state without trusting malformed values", () => {
    const [game] = normalizeGames([
      {
        id: "game-1",
        name: "Example",
        path: "C:/Saves",
        playtimeSeconds: Infinity,
        running: "yes",
        backupPolicy: { enabled: false, intervalMinutes: "bad", retentionCount: 8 },
        protectionPolicy: { mode: "unknown", checkpointIntervalMinutes: 2 },
        protectionStatus: {
          gameId: "server-id",
          serviceControl: { pauseScope: "invalid", protectionEnabled: false },
          provider: { state: "online" },
          activeJob: { id: "job-1", state: "running", progress: 40 },
        },
      },
      "invalid entry",
    ]);

    expect(game).toMatchObject({
      id: "game-1",
      name: "Example",
      playtimeSeconds: 0,
      running: false,
      backupPolicy: { enabled: false, intervalMinutes: 10, retentionCount: 8 },
      protectionPolicy: { mode: "balanced", checkpointIntervalMinutes: 2 },
      protectionStatus: {
        gameId: "server-id",
        serviceControl: { protectionEnabled: false, pauseScope: "none" },
        activeJob: { id: "job-1", state: "running", progress: 40 },
      },
    });
  });

  it("keeps only valid ignore patterns and falls back field-by-field", () => {
    const parsed = normalizeSettings({
      port: "7332",
      themeMode: "neon",
      conflictPolicy: "keep_both",
      ignorePatterns: ["*.bak", 12, "cache"],
      googleDrive: { connected: true, oauthClientSource: "environment", oauthCredentialSource: "bad" },
    });
    const defaults = newDefaultSettings();

    expect(parsed.port).toBe(defaults.port);
    expect(parsed.themeMode).toBe("dark");
    expect(parsed.conflictPolicy).toBe("keep_both");
    expect(parsed.ignorePatterns).toEqual(["*.bak", "cache"]);
    expect(parsed.googleDrive).toEqual({ connected: true, accountEmail: "", reconnectRequired: false, oauthClientSource: "environment", oauthCredentialSource: undefined });
  });

  it("normalizes status, jobs, conflicts, snapshots, and logs", () => {
    expect(normalizeStatus({
      running: true,
      protectionTarget: "folder",
      provider: { state: "offline", pendingCount: 2, pendingBytes: 99 },
      serviceControl: { pauseScope: "uploads", pauseUntilLogin: true },
      pending: 2,
      pendingBytes: 99,
      conflictCount: 1,
    })).toMatchObject({
      running: true,
      protectionTarget: "folder",
      provider: { state: "offline", pendingCount: 2, pendingBytes: 99 },
      serviceControl: { pauseScope: "uploads", pauseUntilLogin: true },
    });
    expect(normalizeSyncJobs([{ id: 1, conflictIds: ["c1", 2], state: "queued" }])).toEqual([expect.objectContaining({ id: "", conflictIds: ["c1"], state: "queued" })]);
    expect(normalizeSyncConflicts([{ id: 1, gameId: "g", path: "slot.dat" }])).toEqual([expect.objectContaining({ id: "", gameId: "g", path: "slot.dat" })]);
    expect(normalizeSnapshots([{ id: "s1", seq: "bad" }])).toEqual([expect.objectContaining({ id: "s1", seq: 0 })]);
    expect(normalizeLogs([{ id: "bad", level: "warn" }])).toEqual([expect.objectContaining({ id: 0, level: "warn", event: "app" })]);
  });
});
