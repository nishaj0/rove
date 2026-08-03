import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { GamesView } from "./GamesView";
import type { Game } from "../types";

const game = (id: string, name: string): Game => ({
  id, name, path: `C:/saves/${id}`, executablePath: `C:/${id}.exe`, coverImageData: "",
  backupPolicy: { enabled: true, intervalMinutes: 10, exitDelaySeconds: 10, retentionCount: 5 },
  playtimeSeconds: 0, lastPlayedAt: "", running: false, syncId: id,
  protectionPolicy: { enabled: true, mode: "balanced", checkpointIntervalMinutes: 10, exitProtection: true },
  protectionStatus: { gameId: id, policy: { enabled: true, mode: "balanced", checkpointIntervalMinutes: 10, exitProtection: true }, serviceControl: { protectionEnabled: true, pauseScope: "none" }, dirtyFiles: 0, pendingCount: 0, pendingBytes: 0, conflictCount: 0 },
});

function renderView(overrides: Partial<React.ComponentProps<typeof GamesView>> = {}) {
  const props: React.ComponentProps<typeof GamesView> = {
    games: [game("one", "One Game"), game("two", "Two Game")], newName: "", newPath: "", newExecutablePath: "", newCoverImageData: "", newVersion: "",
    onNewNameChange: vi.fn(), onBrowseSaveFolder: vi.fn(), onBrowseExecutable: vi.fn(), onNewCoverImageChange: vi.fn(), onNewVersionChange: vi.fn(), onAddGame: vi.fn().mockResolvedValue(true), onOpenGame: vi.fn(), initials: (name) => name.slice(0, 2).toUpperCase(), ...overrides,
  };
  return { ...render(<GamesView {...props} />), props };
}

describe("GamesView", () => {
  it("filters games by name and opens the selected game", () => {
    const { props } = renderView();
    fireEvent.change(screen.getByPlaceholderText("Search games"), { target: { value: "two" } });
    expect(screen.getByText("Two Game")).toBeInTheDocument();
    expect(screen.queryByText("One Game")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Two Game/ }));
    expect(props.onOpenGame).toHaveBeenCalledWith("two");
  });

  it("shows the empty state and add-game modal", async () => {
    renderView({ games: [] });
    expect(screen.getByText("No games added yet.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Add game" }));
    expect(screen.getByRole("heading", { name: "Add a game" })).toBeInTheDocument();
    fireEvent.submit(screen.getByRole("heading", { name: "Add a game" }).closest("section")!.querySelector("form")!);
    await waitFor(() => expect(screen.queryByRole("heading", { name: "Add a game" })).not.toBeInTheDocument());
  });
});
