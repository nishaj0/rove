import type { ActivityEvent, Status } from "../types";

type ActivityViewProps = {
  events: ActivityEvent[];
  status: Status | null;
  onRefresh: () => void;
  onCopyDiagnostics: () => void;
  safeTime: (input: string) => string;
};

export function ActivityView({ events, status, onRefresh, onCopyDiagnostics, safeTime }: ActivityViewProps) {
  return (
    <>
      <section className="panel activity-shell" data-window-title="ACTIVITY.LOG">
        <div className="activity-header">
          <div>
            <h1>Activity</h1>
            <p>Cloud sync, recovery, provider, and conflict history.</p>
          </div>
          <div className="detail-actions">
            <button className="btn" type="button" onClick={onCopyDiagnostics}>Copy diagnostics</button>
            <button className="btn" type="button" onClick={onRefresh}>Refresh</button>
          </div>
        </div>
        <div className="activity-summary">
          <div className="summary-card"><span>Provider</span><strong>{(status?.provider.state ?? "starting").replace(/_/g, " ")}</strong></div>
          <div className="summary-card"><span>Pending uploads</span><strong>{status?.pending ?? 0}</strong></div>
          <div className="summary-card"><span>Conflicts</span><strong>{status?.conflictCount ?? 0}</strong></div>
        </div>
      </section>

      <section className="panel" data-window-title="USEFUL.HISTORY">
        {events.length === 0 ? (
          <p className="empty-state">No cloud sync activity yet. Routine scans and technical request logs are intentionally hidden.</p>
        ) : (
          <ul className="log-feed">
            {events.map((entry) => (
              <li className="log-row" key={entry.id}>
                <div className="log-meta">
                  <span>{safeTime(entry.timestamp)}</span>
                  <strong className={`log-level ${entry.severity}`}>{entry.severity.toUpperCase()}</strong>
                  <em>{entry.category}</em>
                </div>
                <p><strong>{entry.title}</strong></p>
                <p>{entry.detail}</p>
              </li>
            ))}
          </ul>
        )}
      </section>
    </>
  );
}
