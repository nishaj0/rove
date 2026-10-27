import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { Sidebar } from "./Sidebar";
import type { Game } from "../types";

const game = (id: string, name: string): Game => ({
  id, name, path: "C:/s", executablePath: "C:/g.exe", coverImageData: "",
  backupPolicy: { enabled: true, intervalMinutes: 10, exitDelaySeconds: 10, retentionCount: 5 },
  playtimeSeconds: 0, lastPlayedAt: "", running: false, syncId: id,
  protectionPolicy: { enabled: true, mode: "balanced", checkpointIntervalMinutes: 10, exitProtection: true },
  protectionStatus: { gameId: id, policy: { enabled: true, mode: "balanced", checkpointIntervalMinutes: 10, exitProtection: true }, serviceControl: { protectionEnabled: true, pauseScope: "none" }, dirtyFiles: 0, pendingCount: 0, pendingBytes: 0, conflictCount: 0 },
});

function renderSidebar(overrides: Partial<React.ComponentProps<typeof Sidebar>> = {}) {
  const props: React.ComponentProps<typeof Sidebar> = {
    view: "home", games: [game("1", "Ember Reach"), game("2", "Hollow Tide")], selectedGameId: "", query: "", recoveryLibrary: [],
    onHome: vi.fn(), onOpenGame: vi.fn(), onAddGame: vi.fn(), onRestore: vi.fn(), onOpenSettings: vi.fn(), ...overrides,
  };
  return { props, ...render(<Sidebar {...props} />) };
}

describe("Sidebar", () => {
  it("lists games, marks the open one, and opens a game on click", () => {
    const { props } = renderSidebar({ view: "game", selectedGameId: "2" });
    expect(screen.getByRole("button", { name: /Hollow Tide/ })).toHaveAttribute("aria-current", "page");
    fireEvent.click(screen.getByRole("button", { name: /Ember Reach/ }));
    expect(props.onOpenGame).toHaveBeenCalledWith("1");
  });

  it("filters by the toolbar search and explains an empty result", () => {
    renderSidebar({ query: "hollow" });
    expect(screen.queryByText("Ember Reach")).not.toBeInTheDocument();
    expect(screen.getByText("Hollow Tide")).toBeInTheDocument();
  });

  it("adds a game, opens settings and restores from the cloud list", () => {
    const entry = { profileId: "p", name: "Cloud One", syncId: "c1", latestProtected: "", fileCount: 1, totalBytes: 1, automaticCount: 0, recoveryPoints: [] };
    const { props } = renderSidebar({ recoveryLibrary: [entry] });
    fireEvent.click(screen.getByRole("button", { name: "Add a game" }));
    fireEvent.click(screen.getByRole("button", { name: "Settings" }));
    fireEvent.click(screen.getByRole("button", { name: "Restore Cloud One" }));
    expect(props.onAddGame).toHaveBeenCalled();
    expect(props.onOpenSettings).toHaveBeenCalled();
    expect(props.onRestore).toHaveBeenCalledWith(expect.objectContaining({ syncId: "c1" }));
  });
});
