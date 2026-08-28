import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { GameDetailView } from "./GameDetailView";
import type { Game } from "../types";

const game: Game = {
  id: "game-1", name: "Example Game", path: "C:/Saves/Example", executablePath: "C:/Example.exe",
  backupPolicy: { enabled: true, intervalMinutes: 10, exitDelaySeconds: 10, retentionCount: 5 },
  playtimeSeconds: 120, running: false, syncId: "game-1",
  protectionPolicy: { enabled: true, mode: "balanced", checkpointIntervalMinutes: 10, exitProtection: true },
  protectionStatus: { gameId: "game-1", policy: { enabled: true, mode: "balanced", checkpointIntervalMinutes: 10, exitProtection: true }, serviceControl: { protectionEnabled: true, pauseScope: "none" }, dirtyFiles: 0, pendingCount: 0, pendingBytes: 0, conflictCount: 0 },
};

function renderView(overrides: Partial<React.ComponentProps<typeof GameDetailView>> = {}) {
  const props: React.ComponentProps<typeof GameDetailView> = {
    game, exportPath: "", importPath: "", mergeOnImport: true, exportOperationState: "idle", exportOperationMessage: "", importOperationState: "idle", importOperationMessage: "", revisions: [], recoveryPoints: [], storageUsage: null, legacySnapshots: [], syncJob: null, syncConflicts: [],
    onBack: vi.fn(), onPlay: vi.fn(), onProtect: vi.fn(), onOpenSettings: vi.fn(), onCreateShortcut: vi.fn(), onDelete: vi.fn(), onBrowseExportPath: vi.fn(), onBrowseImportPath: vi.fn(), onMergeChange: vi.fn(), onExportBackup: vi.fn(), onImportBackup: vi.fn(), onSavePolicy: vi.fn().mockResolvedValue(true), onPause: vi.fn().mockResolvedValue(undefined), onResume: vi.fn().mockResolvedValue(undefined), onCreateRecoveryPoint: vi.fn(), onRestoreRevision: vi.fn(), onDeleteRecoveryPoint: vi.fn(), onRestoreLegacySnapshot: vi.fn(), onDeleteLegacySnapshot: vi.fn(), onUpdateCover: vi.fn(), onResolveSyncConflict: vi.fn(), formatPlaytime: () => "2m", safeDateTime: () => "now", formatBytes: () => "1 KB", ...overrides,
  };
  return { props, ...render(<GameDetailView {...props} />) };
}

describe("GameDetailView", () => {
  it("keeps maintenance controls in collapsed Advanced setup", () => {
    const { props } = renderView();
    expect(screen.getByRole("button", { name: "Play" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Back up now" })).toBeInTheDocument();
    expect(screen.getByText("Saves are backed up")).toBeInTheDocument();
    const disclosure = screen.getByText("Advanced setup").closest("details")!;
    expect(disclosure).not.toHaveAttribute("open");
    fireEvent.click(screen.getByText("Advanced setup"));
    expect(disclosure).toHaveAttribute("open");
    fireEvent.click(screen.getByRole("button", { name: "Play" }));
    expect(props.onPlay).toHaveBeenCalledTimes(1);
  });

  it("makes a conflict actionable without exposing it as the normal page", () => {
    const { props } = renderView({ game: { ...game, protectionStatus: { ...game.protectionStatus, conflictCount: 1 } }, syncConflicts: [{ id: "conflict-1", gameId: "game-1", path: "save.dat", localHash: "local", remoteHash: "remote", createdAt: "2026-08-29T00:00:00Z", state: "open" }] });
    expect(screen.getAllByText("Choose which saved version to keep")).toHaveLength(2);
    fireEvent.click(screen.getByRole("button", { name: "Review saves" }));
    expect(screen.getByText("Advanced setup").closest("details")).toHaveAttribute("open");
    expect(props.onOpenSettings).not.toHaveBeenCalled();
  });
});
