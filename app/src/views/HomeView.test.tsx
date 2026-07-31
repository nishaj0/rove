import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { HomeView } from "./HomeView";
import type { Status } from "../types";

const status: Status = {
  running: false, protectionTarget: "folder", lastSyncTime: "", pending: 3, syncConfigured: true,
  provider: { state: "offline", pendingCount: 3, pendingBytes: 2048, message: "Provider offline" },
  serviceControl: { protectionEnabled: true, pauseScope: "uploads" }, pendingBytes: 2048, conflictCount: 1,
};

describe("HomeView", () => {
  it("renders actionable protection and recovery states", () => {
    const onOpenLibrary = vi.fn();
    const onRestoreRecovery = vi.fn();
    render(<HomeView games={[]} onOpenLibrary={onOpenLibrary} onOpenActivity={vi.fn()} onOpenGame={vi.fn()} initials={() => "GS"} status={status} recoveryLibrary={[{ profileId: "profile-1", syncId: "sync-1", name: "Example", fileCount: 4, totalBytes: 2048, latestProtected: "", pathHint: "C:/Saves", automaticCount: 2, recoveryPoints: [] }]} onRestoreRecovery={onRestoreRecovery} />);

    expect(screen.getByText("Paused")).toBeInTheDocument();
    expect(screen.getByText("Provider offline")).toBeInTheDocument();
    expect(screen.getByText("3 · 2.0 KB")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Restore to folder" }));
    expect(onRestoreRecovery).toHaveBeenCalledWith(expect.objectContaining({ profileId: "profile-1" }));
    fireEvent.click(screen.getByRole("button", { name: "Open Library" }));
    expect(onOpenLibrary).toHaveBeenCalledTimes(1);
  });
});
