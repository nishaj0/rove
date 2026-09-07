import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { newDefaultSettings } from "../constants/defaults";
import { SettingsView } from "./SettingsView";

describe("SettingsView", () => {
  it("keeps connection details and maintenance controls out of the default view", () => {
    const settings = { ...newDefaultSettings(), protectionTarget: "google_drive" as const, googleDrive: { connected: false, oauthClientSource: "environment" as const, oauthCredentialSource: "environment" as const } };
    render(<SettingsView
      settings={settings}
      activeProtectionTarget="google_drive"
      onChange={vi.fn()}
      onThemeModeChange={vi.fn()}
      onTargetChange={vi.fn()}
      onConnectGoogleDrive={vi.fn().mockResolvedValue(undefined)}
      onDisconnectGoogleDrive={vi.fn().mockResolvedValue(undefined)}
      onBrowseSyncRoot={vi.fn()}
      onAutostartChange={vi.fn().mockResolvedValue(undefined)}
      onPause={vi.fn().mockResolvedValue(undefined)}
      onResume={vi.fn().mockResolvedValue(undefined)}
      onProtectionEnabled={vi.fn().mockResolvedValue(undefined)}
      onRetry={vi.fn().mockResolvedValue(undefined)}
      onSaveSettings={vi.fn()}
      onSaveDestination={vi.fn()}
      updateState={null}
      updateInstallBlocked={false}
      onCheckForUpdate={vi.fn()}
      onDownloadUpdate={vi.fn()}
      onInstallUpdate={vi.fn()}
      onDiscardUpdate={vi.fn()}
      onAutomaticUpdateChecks={vi.fn()}
      onViewUpdateNotes={vi.fn()}
      onDownloadLogs={vi.fn()}
      activityEvents={[]}
      safeTime={() => "now"}
      recoveryLibrary={[]}
      onRestoreRecovery={vi.fn()}
      onOpenGame={vi.fn()}
      onListDriveFolders={vi.fn().mockResolvedValue([])}
      onSetDriveFolder={vi.fn().mockResolvedValue(undefined)}
      onCreateDriveFolder={vi.fn().mockResolvedValue(undefined)}
      onExportSettings={vi.fn().mockResolvedValue(undefined)}
      onImportSettingsFile={vi.fn()}
    />);

    expect(screen.getByRole("heading", { name: "Save location" })).toBeInTheDocument();
    expect(screen.getByText("Protect my game saves")).toBeInTheDocument();
    expect(screen.queryByText(/OAuth client/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/integration pending/i)).not.toBeInTheDocument();
    expect(screen.getByText("Advanced backup options").closest("details")).not.toHaveAttribute("open");
  });
});
