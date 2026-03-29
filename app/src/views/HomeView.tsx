import type { Game } from "../types";

type HomeViewProps = {
  pending: number;
  games: Game[];
  autostart: boolean;
  lastSyncLabel: string;
  syncLabel: string;
  syncClass: "success" | "warning" | "error" | "syncing";
  onManageGames: () => void;
  initials: (name: string) => string;
};

export function HomeView(props: HomeViewProps) {
  const { pending, games, autostart, lastSyncLabel, syncLabel, syncClass, onManageGames, initials } = props;

  return (
    <>
      <section className="stats-grid">
        <article className="stat-card">
          <h3>Pending events</h3>
          <p>{pending}</p>
        </article>
        <article className="stat-card">
          <h3>Tracked games</h3>
          <p>{games.length}</p>
        </article>
        <article className="stat-card">
          <h3>Last sync</h3>
          <p>{lastSyncLabel}</p>
        </article>
        <article className="stat-card">
          <h3>Autostart</h3>
          <p>{autostart ? "Enabled" : "Disabled"}</p>
        </article>
        <article className="stat-card">
          <h3>Sync state</h3>
          <p className={`state-text ${syncClass}`}>{syncLabel}</p>
        </article>
      </section>

      <section className="panel">
        <div className="panel-head">
          <h2>Library</h2>
          <button className="btn" onClick={onManageGames} type="button">Manage</button>
        </div>
        {games.length === 0 ? (
          <p className="empty-state">No games tracked yet. Open Games to add your first save folder.</p>
        ) : (
          <div className="game-grid">
            {games.slice(0, 6).map((g) => (
              <article className="game-card" key={g.id}>
                <div className="game-cover">
                  <span>{initials(g.name)}</span>
                </div>
                <div className="game-meta">
                  <strong>{g.name}</strong>
                  <p title={g.path}>{g.path}</p>
                </div>
              </article>
            ))}
          </div>
        )}
      </section>
    </>
  );
}
