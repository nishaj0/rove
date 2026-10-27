import { Plus } from "lucide-react";
import type { ActivityEvent, Game, RecoveryLibraryEntry, Status } from "../types";
import { cloudOnlyEntries, formatSize, gameStatus, relativeTime } from "../utils/gameStatus";

type HomeViewProps = {
  games: Game[];
  status: Status | null;
  recoveryLibrary: RecoveryLibraryEntry[];
  activityEvents: ActivityEvent[];
  safeTime: (input: string) => string;
  targetLabel: string;
  onAddGame: () => void;
  onOpenGame: (id: string) => void;
  onRestore: (entry: RecoveryLibraryEntry) => void;
  onOpenSettings: () => void;
};

function CloudList({ entries, onRestore }: { entries: RecoveryLibraryEntry[]; onRestore: (entry: RecoveryLibraryEntry) => void }) {
  return (
    <div className="list">
      {entries.map((entry) => (
        <div className="list-row" key={entry.profileId}>
          <div className="meta">
            <strong>{entry.name}</strong>
            <span>{entry.version ? `v${entry.version} · ` : ""}{entry.fileCount} file{entry.fileCount === 1 ? "" : "s"} · {formatSize(entry.totalBytes)}</span>
          </div>
          {entry.requiresUpdate ? <span className="pill warn">Needs a newer Rove</span> : <button className="btn sm" type="button" onClick={() => onRestore(entry)}>Restore</button>}
        </div>
      ))}
    </div>
  );
}

export function HomeView(props: HomeViewProps) {
  const { games, status, recoveryLibrary, activityEvents, safeTime, targetLabel, onAddGame, onOpenGame, onRestore, onOpenSettings } = props;
  const cloud = cloudOnlyEntries(recoveryLibrary, games);

  if (games.length === 0) {
    const saveLocationChosen = targetLabel !== "This PC only";
    return (
      <div className="pane">
        <div className="welcome">
          <h1>{cloud.length > 0 ? "Welcome back" : "Protect your first game"}</h1>
          <p>{cloud.length > 0 ? "Your cloud already holds saves. Restore them to this PC, or add a new game." : "Rove keeps safe, versioned copies of your game saves in the background, so a crash, a reinstall or a new PC never costs you progress."}</p>
          <div className="steps">
            <div className={`step ${saveLocationChosen ? "done" : ""}`}>
              <div><strong>Choose where saves are kept</strong><span>Currently: {targetLabel}</span></div>
              <button className="btn" type="button" onClick={onOpenSettings}>{saveLocationChosen ? "Change" : "Choose"}</button>
            </div>
            <div className="step">
              <div><strong>Add a game</strong><span>Pick its save folder and executable. Protection starts right away.</span></div>
              <button className="btn primary" type="button" onClick={onAddGame}><Plus aria-hidden="true" size={15} /> Add a game</button>
            </div>
          </div>
        </div>
        {cloud.length > 0 ? (
          <section className="section" aria-label="Games in your cloud">
            <div className="section-head"><h2>In your cloud</h2><p>{cloud.length} game{cloud.length === 1 ? "" : "s"}</p></div>
            <CloudList entries={cloud} onRestore={onRestore} />
          </section>
        ) : null}
      </div>
    );
  }

  const attention = games.map((game) => ({ game, state: gameStatus(game) })).filter(({ state }) => state.tone === "warn" || state.tone === "err");
  const lastBackup = games.map((game) => Date.parse(game.protectionStatus.lastLocalProtection ?? "")).filter((ms) => !Number.isNaN(ms) && new Date(ms).getUTCFullYear() > 2000).sort((a, b) => b - a)[0];

  return (
    <div className="pane">
      <div className="page-head">
        <div><h1>Home</h1><p>{attention.length === 0 ? "All your saves are protected." : `${attention.length} game${attention.length === 1 ? " needs" : "s need"} attention.`}</p></div>
        <button className="btn primary" type="button" onClick={onAddGame}><Plus aria-hidden="true" size={15} /> Add a game</button>
      </div>

      <div className="stat-grid">
        <div className="stat"><span>Games protected</span><strong>{games.length}</strong></div>
        <div className="stat"><span>Last backup</span><strong>{lastBackup ? relativeTime(new Date(lastBackup).toISOString()) : "Not yet"}</strong></div>
        <div className="stat"><span>Save location</span><strong>{targetLabel}</strong></div>
        <div className="stat"><span>Waiting to upload</span><strong>{status?.pending ?? 0}</strong></div>
      </div>

      {attention.length > 0 ? (
        <section className="section" aria-label="Needs attention">
          <div className="section-head"><h2>Needs attention</h2></div>
          <div className="list">
            {attention.map(({ game, state }) => (
              <div className="list-row" key={game.id}>
                <div className="meta"><strong>{game.name}</strong><span>{state.text}</span></div>
                <button className="btn sm" type="button" onClick={() => onOpenGame(game.id)}>Review</button>
              </div>
            ))}
          </div>
        </section>
      ) : null}

      {cloud.length > 0 ? (
        <section className="section" aria-label="Games in your cloud">
          <div className="section-head"><h2>In your cloud</h2><p>Saved from another PC or an earlier install</p></div>
          <CloudList entries={cloud} onRestore={onRestore} />
        </section>
      ) : null}

      <section className="section" aria-label="Recent activity">
        <div className="section-head"><h2>Recent activity</h2></div>
        {activityEvents.length === 0 ? (
          <p className="empty-note">Backups and restores will appear here.</p>
        ) : (
          <div className="list">
            {activityEvents.slice(0, 5).map((entry) => (
              <div className="log-row" key={entry.id}>
                <strong>{entry.title}</strong>
                {entry.detail ? <span className="hint">{entry.detail}</span> : null}
                <time>{safeTime(entry.timestamp)}</time>
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
