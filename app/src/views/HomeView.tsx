import type { Game } from "../types";

type HomeViewProps = {
  pending: number;
  games: Game[];
  autostart: boolean;
  lastSyncLabel: string;
  syncLabel: string;
  syncClass: "success" | "warning" | "error" | "syncing";
  onManageGames: () => void;
  onOpenGame: (gameId: string) => void;
  initials: (name: string) => string;
};

export function HomeView(props: HomeViewProps) {
  const { pending, games, autostart, lastSyncLabel, syncLabel, syncClass, onManageGames, onOpenGame, initials } = props;

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
          <h3>Service state</h3>
          <p className={`state-text ${syncClass}`}>{syncLabel}</p>
        </article>
      </section>

      <section className="panel">
        <div className="panel-head">
          <h2>Game Library</h2>
          <button className="btn" onClick={onManageGames} type="button">See all</button>
        </div>
        {games.length === 0 ? (
          <p className="empty-state">No games tracked yet. Open Games to add your first save folder.</p>
        ) : (
          <div className="game-grid">
            {games.slice(0, 6).map((g) => (
              <button className="game-card game-card-button" key={g.id} onClick={() => onOpenGame(g.id)} type="button">
                <div className="game-cover">
                  {g.coverImageData ? <img className="game-cover-image" src={g.coverImageData} alt={`${g.name} cover`} /> : <span>{initials(g.name)}</span>}
                  <span className={`game-state ${g.running ? "success" : "warning"}`}>{g.running ? "Running" : "Idle"}</span>
                </div>
                <div className="game-meta">
                  <div className="game-meta-main">
                    <strong>{g.name}</strong>
                    <p className="game-playtime">Open game profile</p>
                  </div>
                  <span className="open-chip">Open</span>
                </div>
              </button>
            ))}
          </div>
        )}
      </section>
    </>
  );
}
