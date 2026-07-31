import { beforeEach, describe, expect, it, vi } from "vitest";
import { updateActionLabel, checkForAppUpdate, listenForUpdateProgress } from "./update";
import type { AppUpdateState } from "./update";

const { invokeMock, listenMock } = vi.hoisted(() => ({
  invokeMock: vi.fn(),
  listenMock: vi.fn(),
}));

vi.mock("@tauri-apps/api/core", () => ({ invoke: invokeMock }));
vi.mock("@tauri-apps/api/event", () => ({ listen: listenMock }));

const state = (overrides: Partial<AppUpdateState> = {}): AppUpdateState => ({
  lifecycle: "idle", currentVersion: "0.1.0", downloadedBytes: 0,
  retryable: false, automaticChecksEnabled: true, development: false, ...overrides,
});

describe("update helpers", () => {
  beforeEach(() => {
    invokeMock.mockReset();
    listenMock.mockReset();
  });

  it("uses stable labels for every update lifecycle", () => {
    expect(updateActionLabel(null)).toBe("Updates");
    expect(updateActionLabel(state({ lifecycle: "downloading", downloadedBytes: 25, totalBytes: 100 }))).toBe("Downloading 25%");
    expect(updateActionLabel(state({ lifecycle: "downloading", downloadedBytes: 25 }))).toBe("Downloading update");
    expect(updateActionLabel(state({ lifecycle: "ready", availableVersion: "1.2.0" }))).toBe("Install v1.2.0");
    expect(updateActionLabel(state({ lifecycle: "error" }))).toBe("Update needs attention");
    expect(updateActionLabel(state({ lifecycle: "available", availableVersion: "1.2.0" }))).toBe("Update v1.2.0");
  });

  it("passes command arguments to the native update bridge", async () => {
    invokeMock.mockResolvedValue(state({ lifecycle: "checking" }));
    await checkForAppUpdate(true);
    expect(invokeMock).toHaveBeenCalledWith("check_for_update", { force: true });
  });

  it("forwards progress events and returns the native unsubscribe function", async () => {
    const unlisten = vi.fn();
    listenMock.mockResolvedValue(unlisten);
    const handler = vi.fn();
    const promise = listenForUpdateProgress(handler);
    const callback = listenMock.mock.calls[0][1];
    callback({ payload: state({ lifecycle: "ready", availableVersion: "1.2.0" }) });

    expect(handler).toHaveBeenCalledWith(expect.objectContaining({ lifecycle: "ready" }));
    await expect(promise).resolves.toBe(unlisten);
    expect(listenMock).toHaveBeenCalledWith("cloud-saver://update-progress", expect.any(Function));
  });
});
