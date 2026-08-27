import { beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "./api";

function jsonResponse(payload: unknown, init: Partial<Response> = {}) {
  return {
    ok: true,
    status: 200,
    statusText: "OK",
    json: vi.fn().mockResolvedValue(payload),
    ...init,
  } as unknown as Response;
}

describe("frontend API client", () => {
  const fetchMock = vi.fn();

  beforeEach(() => {
    fetchMock.mockReset();
    vi.stubGlobal("fetch", fetchMock);
  });

  it("sends JSON requests and normalizes collection responses", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse([{ id: "g1", name: "Game" }]));

    await expect(api.getGames()).resolves.toEqual([expect.objectContaining({ id: "g1", name: "Game" })]);
    expect(fetchMock).toHaveBeenCalledWith("http://127.0.0.1:7331/games", expect.objectContaining({
      headers: { "Content-Type": "application/json" },
	  cache: "no-store",
    }));
  });

  it("encodes path and query identifiers", async () => {
    fetchMock.mockResolvedValue(jsonResponse([]));

    await api.getGame("save/one");
    await api.getRecoveryPoints("save one");
    await api.getProtectionStatus("save/one");

    expect(fetchMock.mock.calls.map(([url]) => url)).toEqual([
      "http://127.0.0.1:7331/games/save%2Fone",
      "http://127.0.0.1:7331/recovery-points?gameId=save%20one",
      "http://127.0.0.1:7331/protection/status?gameId=save%2Fone",
    ]);
  });

  it("falls back from the newer game-sync endpoint only for compatibility errors", async () => {
    fetchMock
      .mockResolvedValueOnce(jsonResponse({ error: "method not allowed" }, { ok: false, status: 405, statusText: "Method Not Allowed" }))
      .mockResolvedValueOnce(jsonResponse({ status: "queued", jobId: "job-1" }));

    await expect(api.syncGame({ gameId: "g1" })).resolves.toEqual({ status: "queued", jobId: "job-1" });
    expect(fetchMock.mock.calls.map(([url]) => url)).toEqual([
      "http://127.0.0.1:7331/games/sync",
      "http://127.0.0.1:7331/sync/game",
    ]);
  });

  it("does not hide non-compatibility failures and reports server error payloads", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse({ error: "provider offline" }, { ok: false, status: 503, statusText: "Unavailable" }));
    await expect(api.triggerSync()).rejects.toThrow("provider offline");
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("supports empty 204 responses and preserves request bodies", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse(undefined, { status: 204 }));
    await expect(api.deleteGame("g1")).resolves.toBeUndefined();

    fetchMock.mockResolvedValueOnce(jsonResponse({ status: "queued", jobId: "j1" }));
    await api.createRecoveryPoint("g1", "Before boss");
    expect(fetchMock.mock.calls[1][1]).toMatchObject({
      method: "POST",
      body: JSON.stringify({ gameId: "g1", name: "Before boss" }),
    });
  });
});
