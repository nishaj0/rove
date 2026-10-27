import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { HomeView } from "./HomeView";
import type { Game, RecoveryLibraryEntry } from "../types";

const entry = (id: string, extra: Partial<RecoveryLibraryEntry> = {}): RecoveryLibraryEntry => ({ profileId: id, name: `Cloud ${id}`, syncId: id, latestProtected: "", fileCount: 2, totalBytes: 2048, automaticCount: 0, recoveryPoints: [], ...extra });
const game = (id: string, conflictCount = 0): Game => ({
  id, name: `Game ${id}`, path: "C:/s", executablePath: "C:/g.exe", coverImageData: "",
  backupPolicy: { enabled: true, intervalMinutes: 10, exitDelaySeconds: 10, retentionCount: 5 },
  playtimeSeconds: 0, lastPlayedAt: "", running: false, syncId: id,
  protectionPolicy: { enabled: true, mode: "balanced", checkpointIntervalMinutes: 10, exitProtection: true },
  protectionStatus: { gameId: id, policy: { enabled: true, mode: "balanced", checkpointIntervalMinutes: 10, exitProtection: true }, serviceControl: { protectionEnabled: true, pauseScope: "none" }, dirtyFiles: 0, pendingCount: 0, pendingBytes: 0, conflictCount, lastLocalProtection: new Date().toISOString() },
});

function renderHome(overrides: Partial<React.ComponentProps<typeof HomeView>> = {}) {
  const props: React.ComponentProps<typeof HomeView> = {
    games: [], status: null, recoveryLibrary: [], activityEvents: [], safeTime: () => "now", targetLabel: "This PC only",
    onAddGame: vi.fn(), onOpenGame: vi.fn(), onRestore: vi.fn(), onOpenSettings: vi.fn(), ...overrides,
  };
  return { props, ...render(<HomeView {...props} />) };
}

describe("HomeView", () => {
  it("teaches first-run steps when the library is empty", () => {
    const { props } = renderHome();
    expect(screen.getByRole("heading", { name: "Protect your first game" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Add a game/ }));
    expect(props.onAddGame).toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Choose" }));
    expect(props.onOpenSettings).toHaveBeenCalled();
  });

  it("offers cloud games for restore and blocks saves from a newer Rove", () => {
    const { props } = renderHome({ recoveryLibrary: [entry("a"), entry("b", { requiresUpdate: true }), entry("c", { gameId: "local-c" })] });
    expect(screen.getByRole("heading", { name: "Welcome back" })).toBeInTheDocument();
    expect(screen.queryByText("Cloud c")).not.toBeInTheDocument();
    expect(screen.getByText("Needs a newer Rove")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Restore" }));
    expect(props.onRestore).toHaveBeenCalledWith(expect.objectContaining({ syncId: "a" }));
  });

  it("hides a cloud entry once its game is tracked locally", () => {
    renderHome({ games: [game("one")], recoveryLibrary: [entry("one")] });
    expect(screen.queryByRole("region", { name: "Games in your cloud" })).not.toBeInTheDocument();
  });

  it("surfaces games that need attention", () => {
    const { props } = renderHome({ games: [game("ok"), game("bad", 1)] });
    expect(screen.getByText("1 game needs attention.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Review" }));
    expect(props.onOpenGame).toHaveBeenCalledWith("bad");
  });
});
