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
    onPlay: vi.fn(), onProtect: vi.fn(), onOpenSettings: vi.fn(), onCreateShortcut: vi.fn(), onDelete: vi.fn(), onBrowseExportPath: vi.fn(), onBrowseImportPath: vi.fn(), onMergeChange: vi.fn(), onExportBackup: vi.fn(), onImportBackup: vi.fn(), onSavePolicy: vi.fn().mockResolvedValue(true), onPause: vi.fn().mockResolvedValue(undefined), onResume: vi.fn().mockResolvedValue(undefined), onCreateRecoveryPoint: vi.fn(), onRestoreRevision: vi.fn(), onDeleteRecoveryPoint: vi.fn(), onRestoreLegacySnapshot: vi.fn(), onDeleteLegacySnapshot: vi.fn(), onUpdateCover: vi.fn(), onResolveSyncConflict: vi.fn(), formatPlaytime: () => "2m", safeDateTime: () => "now", formatBytes: () => "1 KB", ...overrides,
  };
  return { props, ...render(<GameDetailView {...props} />) };
}

describe("GameDetailView", () => {
  it("shows primary actions up front and keeps maintenance in tabs", () => {
    const { props } = renderView();
    expect(screen.getByRole("button", { name: "Play" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Back up now" })).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("Saves are backed up");
    expect(screen.getByRole("tab", { name: "Saves" })).toHaveAttribute("aria-selected", "true");
    expect(screen.queryByRole("button", { name: "Remove game from Rove" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("tab", { name: "Details" }));
    expect(screen.getByRole("button", { name: "Remove game from Rove" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("tab", { name: "Backup options" }));
    expect(screen.getByRole("button", { name: "Save backup options" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Play" }));
    expect(props.onPlay).toHaveBeenCalledTimes(1);
  });

  it("makes a conflict actionable and jumps back to the Saves tab to review it", () => {
    const { props } = renderView({ game: { ...game, protectionStatus: { ...game.protectionStatus, conflictCount: 1 } }, syncConflicts: [{ id: "conflict-1", gameId: "game-1", path: "save.dat", localHash: "localhash", remoteHash: "remotehash", createdAt: "2026-01-01T00:00:00Z" } as never] });
    fireEvent.click(screen.getByRole("tab", { name: /Backup options/ }));
    fireEvent.click(screen.getByRole("button", { name: "Review" }));
    expect(screen.getByRole("region", { name: "Choose a saved version" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Keep both" }));
    expect(props.onResolveSyncConflict).toHaveBeenCalledWith("conflict-1", "both");
    expect(props.onOpenSettings).not.toHaveBeenCalled();
  });
});
