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
    <section className="panel">
      <div className="panel-head">
        <h2>Recent Activity</h2>
        <button className="btn" type="button" onClick={onRefresh}>Refresh</button>
      </div>

      <div className="debug-grid">
        <div className="debug-item">
          <span>Backend debug</span>
          <strong>{debugInfo ? (debugInfo.debug ? "enabled" : "disabled") : "offline"}</strong>
        </div>
        <div className="debug-item">
          <span>PID</span>
          <strong>{debugInfo?.pid ?? "-"}</strong>
        </div>
        <div className="debug-item">
          <span>Uptime</span>
          <strong>{uptimeLabel}</strong>
        </div>
        <div className="debug-item full">
          <span>Config file</span>
          <strong>{debugInfo?.configPath ?? "-"}</strong>
        </div>
        <div className="debug-item full">
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
        <ul className="activity-list">
          {logs.map((entry) => (
            <li key={entry.id}>
              <span>{safeTime(entry.timestamp)}</span>
              <strong>{entry.level.toUpperCase()}</strong>
              <em>{entry.event}</em>
              <p>{entry.message}</p>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
