import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "../api";
import type { LogEntry } from "../types";

type DebugConsoleViewProps = {
  onDownloadLogs: () => void;
};

const POLL_INTERVAL_MS = 3000;
const MAX_DISPLAY_ENTRIES = 300;

function levelClass(level: string): string {
  const l = level.toLowerCase();
  if (l === "error") return "error";
  if (l === "warn" || l === "warning") return "warn";
  if (l === "debug") return "debug";
  return "info";
}

function formatTs(ts: string): string {
  try {
    const d = new Date(ts);
    const hh = String(d.getHours()).padStart(2, "0");
    const mm = String(d.getMinutes()).padStart(2, "0");
    const ss = String(d.getSeconds()).padStart(2, "0");
    return `${hh}:${mm}:${ss}`;
  } catch {
    return ts.slice(11, 19) || ts;
  }
}

export function DebugConsoleView({ onDownloadLogs }: DebugConsoleViewProps) {
  const [entries, setEntries] = useState<LogEntry[]>([]);
  const [autoScroll, setAutoScroll] = useState(true);
  const [downloading, setDownloading] = useState(false);
  const [pollError, setPollError] = useState("");
  const [cleared, setCleared] = useState(false);
  const [clearedBefore, setClearedBefore] = useState<string | null>(null);

  const scrollRef = useRef<HTMLDivElement>(null);
  const lastIdRef = useRef<number>(0);
  const autoScrollRef = useRef(autoScroll);
  autoScrollRef.current = autoScroll;

  // Fetch recent logs — keep only entries newer than the last clear
  const fetchLogs = useCallback(async () => {
    try {
      const fresh = await api.getRecentLogs(200);
      setPollError("");
      setEntries((prev) => {
        // merge: keep newest by id
        const existingIds = new Set(prev.map((e) => e.id));
        const incoming = fresh.filter((e) => !existingIds.has(e.id));
        if (incoming.length === 0) return prev;
        const merged = [...prev, ...incoming].sort((a, b) => a.id - b.id);
        // trim to cap
        return merged.slice(-MAX_DISPLAY_ENTRIES);
      });
      if (fresh.length > 0) {
        lastIdRef.current = Math.max(lastIdRef.current, ...fresh.map((e) => e.id));
      }
    } catch {
      setPollError("Could not reach backend — is the daemon running?");
    }
  }, []);

  // Initial load + polling
  useEffect(() => {
    void fetchLogs();
    const timer = setInterval(() => void fetchLogs(), POLL_INTERVAL_MS);
    return () => clearInterval(timer);
  }, [fetchLogs]);

  // Auto-scroll
  useEffect(() => {
    if (autoScrollRef.current && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [entries]);

  function handleScroll() {
    const el = scrollRef.current;
    if (!el) return;
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 32;
    setAutoScroll(atBottom);
  }

  function handleClear() {
    setCleared(true);
    setClearedBefore(new Date().toISOString());
    setEntries([]);
  }

  async function handleDownload() {
    setDownloading(true);
    try {
      await Promise.resolve(onDownloadLogs());
    } finally {
      setDownloading(false);
    }
  }

  // Filter out entries before the last clear
  const visible = clearedBefore
    ? entries.filter((e) => e.timestamp > clearedBefore)
    : entries;

  return (
    <section className="panel debug-console-panel" data-window-title="DEBUG.CONSOLE">
      <div className="debug-toolbar">
        <span className="debug-toolbar-status">
          {pollError
            ? <span className="debug-status-error">{pollError}</span>
            : <span className="debug-status-ok">● Live · {visible.length} lines</span>
          }
        </span>
        <div className="debug-toolbar-actions">
          <label className="debug-autoscroll-label">
            <input
              type="checkbox"
              checked={autoScroll}
              onChange={(e) => setAutoScroll(e.target.checked)}
            />
            Auto-scroll
          </label>
          <button
            className="btn"
            type="button"
            onClick={handleClear}
            title="Clear display — does not delete stored logs"
          >
            Clear
          </button>
          <button
            className="btn primary"
            type="button"
            disabled={downloading}
            onClick={() => void handleDownload()}
            title="Download all logs as a ZIP file"
          >
            {downloading ? "Preparing…" : "Download logs"}
          </button>
        </div>
      </div>

      <div
        className="debug-terminal"
        ref={scrollRef}
        onScroll={handleScroll}
        aria-label="Debug log output"
        aria-live="polite"
        aria-atomic="false"
      >
        {cleared && clearedBefore && (
          <div className="debug-clear-marker">
            — display cleared at {clearedBefore.slice(11, 19)} UTC · stored logs unchanged —
          </div>
        )}
        {visible.length === 0 && !pollError && (
          <div className="debug-empty">No log entries yet. Trigger any action to see output.</div>
        )}
        {visible.map((entry) => (
          <div key={entry.id} className="debug-entry">
            <span className="debug-ts">{formatTs(entry.timestamp)}</span>
            <span className={`log-level ${levelClass(entry.level)}`}>{entry.level.toUpperCase()}</span>
            <span className="debug-event">{entry.event}</span>
            <span className="debug-msg">{entry.message}</span>
          </div>
        ))}
      </div>
    </section>
  );
}
