import type { Game, RecoveryLibraryEntry, Status } from "../types";
import type { StorageSummary } from "../types";
import { GameCover } from "../components/GameCover";
import { HOME_KICKER_LABEL } from "../constants/branding";

type HomeViewProps = {
  games: Game[];
  onOpenLibrary: () => void;
  onOpenActivity: () => void;
  onOpenGame: (gameId: string) => void;
  initials: (name: string) => string;
  status: Status | null;
  recoveryLibrary: RecoveryLibraryEntry[];
  onRestoreRecovery: (entry: RecoveryLibraryEntry) => void;
  storageSummary?: StorageSummary | null;
};

export function HomeView(props: HomeViewProps) {
  const { games, onOpenLibrary, onOpenActivity, onOpenGame, initials, status, recoveryLibrary, onRestoreRecovery, storageSummary } = props;

  return (
    <>
      <section className="panel home-hero" data-window-title="WELCOME.TXT">
        <p className="home-kicker">{HOME_KICKER_LABEL}</p>
        <h1 className="home-title">Manage your save library</h1>
        <p className="home-subtitle">Track saves, launch a game profile, and handle backup import or export from one place.</p>
        <div className="home-actions">
          <button className="btn primary" onClick={onOpenLibrary} type="button">Open Library</button>
          <button className="btn" onClick={onOpenActivity} type="button">View Activity</button>
        </div>
      </section>

      <section className="panel" data-window-title="CLOUD.SYNC.STATUS">
        <div className="panel-head">
          <h2>Cloud Sync health</h2>
          <span className={`state-text ${status?.provider.state === "online" || status?.provider.state === "local_only" ? "success" : status?.provider.state === "offline" || status?.serviceControl.pauseScope !== "none" ? "warning" : "error"}`}>
            {status?.serviceControl.pauseScope !== "none" ? "Paused" : (status?.provider.state ?? "Starting").replace(/_/g, " ")}
          </span>
        </div>
        <div className="health-grid">
          <div><span>Last local sync</span><strong>{status?.lastLocalProtection ? new Date(status.lastLocalProtection).toLocaleString() : "Not yet"}</strong></div>
          <div><span>Last cloud upload</span><strong>{status?.lastCloudUpload ? new Date(status.lastCloudUpload).toLocaleString() : status?.provider.state === "local_only" ? "Local only" : "Not yet"}</strong></div>
          <div><span>Pending uploads</span><strong>{status?.pending ?? 0} · {formatBytes(status?.pendingBytes ?? 0)}</strong></div>
          <div><span>Conflicts</span><strong>{status?.conflictCount ?? 0} need review</strong></div>
          <div><span>Rove storage</span><strong>{storageSummary ? `${formatBytes(storageSummary.cloudAvailable ? storageSummary.cloudBytes : storageSummary.localBytes)}${storageSummary.cloudAvailable ? " cloud" : " local"}` : "Calculating"}</strong></div>
        </div>
        {status?.provider.message ? <p className="empty-state">{status.provider.message}</p> : null}
        {status?.lastError ? <p className="banner error">{status.lastError}</p> : null}
      </section>

      <section className="panel" data-window-title="RECOVERY.LIBRARY">
        <div className="panel-head">
          <h2>Recovery Library</h2>
          <span className="state-text">{recoveryLibrary.length} synced</span>
        </div>
        {recoveryLibrary.length === 0 ? (
          <p className="empty-state">Games appear here after their first local sync or provider connection.</p>
        ) : (
          <div className="recovery-library-list">
            {recoveryLibrary.map((entry) => (
              <article className="recovery-library-row" key={entry.profileId}>
                <div>
                  <strong>{entry.name}</strong>
                  <span>{entry.fileCount} files · {formatBytes(entry.totalBytes)} · {entry.latestProtected ? new Date(entry.latestProtected).toLocaleString() : "Sync time unavailable"}</span>
                  {entry.pathHint ? <span>Suggested location: {entry.pathHint}</span> : null}
                </div>
                {entry.gameId ? (
                  <button className="btn" type="button" onClick={() => onOpenGame(entry.gameId!)}>Open game</button>
                ) : (
                  <button className="btn primary" type="button" onClick={() => onRestoreRecovery(entry)}>Restore and track</button>
                )}
              </article>
            ))}
          </div>
        )}
      </section>

      <section className="panel" data-window-title="RECENT.GAMES">
        <div className="panel-head">
          <h2>Recent Games</h2>
          <button className="btn" onClick={onOpenLibrary} type="button">See all</button>
        </div>
        {games.length === 0 ? (
          <p className="empty-state">No games tracked yet. Open Games to add your first save folder.</p>
        ) : (
          <div className="library-grid">
            {games.slice(0, 4).map((g) => (
              <button className="library-card" key={g.id} onClick={() => onOpenGame(g.id)} type="button">
                <GameCover game={g} className="library-card-image" />
                <div className="library-card-name">{g.name}</div>
              </button>
            ))}
          </div>
        )}
      </section>
    </>
  );
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
