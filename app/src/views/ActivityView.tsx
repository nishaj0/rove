import type { DebugInfo, LogEntry } from "../types";

type ActivityViewProps = {
  debugInfo: DebugInfo | null;
  uptimeLabel: string;
  lastError: string;
  logs: LogEntry[];
  onRefresh: () => void;
  safeTime: (input: string) => string;
};

export function ActivityView(props: ActivityViewProps) {
  const { debugInfo, uptimeLabel, lastError, logs, onRefresh, safeTime } = props;

  return (
    <section className="panel activity-shell" data-window-title="ACTIVITY.LOG">
      <div className="activity-header">
        <div>
          <h1>Activity</h1>
          <p>Service diagnostics and recent sync events.</p>
        </div>
        <button className="btn" type="button" onClick={onRefresh}>Refresh</button>
      </div>

      <div className="activity-summary">
        <div className="summary-card">
          <span>Backend mode</span>
          <strong>{debugInfo ? (debugInfo.debug ? "Debug" : "Normal") : "Offline"}</strong>
        </div>
        <div className="summary-card">
          <span>PID</span>
          <strong>{debugInfo?.pid ?? "-"}</strong>
        </div>
        <div className="summary-card">
          <span>Uptime</span>
          <strong>{uptimeLabel}</strong>
        </div>
        <div className="summary-card full">
          <span>Config file</span>
          <strong>{debugInfo?.configPath ?? "-"}</strong>
        </div>
        <div className="summary-card full">
          <span>Log database</span>
          <strong>{debugInfo?.sqlitePath ?? "-"}</strong>
        </div>
      </div>

      {lastError ? (
        <div className="banner error">Last backend error: {lastError}</div>
      ) : null}

      {logs.length === 0 ? (
        <p className="empty-state">No logs yet.</p>
      ) : (
        <ul className="log-feed">
          {logs.map((entry) => (
            <li className="log-row" key={entry.id}>
              <div className="log-meta">
                <span>{safeTime(entry.timestamp)}</span>
                <strong className={`log-level ${entry.level.toLowerCase()}`}>{entry.level.toUpperCase()}</strong>
                <em>{entry.event}</em>
              </div>
              <p>{entry.message}</p>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
