import type { Game, Status } from "../types";
import { HOME_KICKER_LABEL } from "../constants/branding";

type HomeViewProps = {
  games: Game[];
  onOpenLibrary: () => void;
  onOpenActivity: () => void;
  onOpenGame: (gameId: string) => void;
  initials: (name: string) => string;
  status: Status | null;
};

export function HomeView(props: HomeViewProps) {
  const { games, onOpenLibrary, onOpenActivity, onOpenGame, initials, status } = props;

  return (
    <>
      <section className="panel home-hero">
        <p className="home-kicker">{HOME_KICKER_LABEL}</p>
        <h1 className="home-title">Manage your save library</h1>
        <p className="home-subtitle">Track saves, launch a game profile, and handle backup import or export from one place.</p>
        <div className="home-actions">
          <button className="btn primary" onClick={onOpenLibrary} type="button">Open Library</button>
          <button className="btn" onClick={onOpenActivity} type="button">View Activity</button>
        </div>
      </section>

      <section className="panel">
        <div className="panel-head"><h2>Cloud Sync</h2><span className={`state-text ${status?.syncConfigured ? "success" : "warning"}`}>{status?.syncConfigured ? "Ready" : "Setup needed"}</span></div>
        <p className="empty-state">{status?.syncConfigured ? `Last completed: ${status.lastSyncTime ? new Date(status.lastSyncTime).toLocaleString() : "Not yet"}. ${status.pending} request${status.pending === 1 ? "" : "s"} pending.` : "Choose a synced folder in Settings. Your existing local backups remain unchanged."}</p>
      </section>

      <section className="panel">
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
                {g.coverImageData ? <img className="library-card-image" src={g.coverImageData} alt={`${g.name} cover`} /> : <div className="library-card-fallback">{initials(g.name)}</div>}
                <div className="library-card-name">{g.name}</div>
              </button>
            ))}
          </div>
        )}
      </section>
    </>
  );
}
