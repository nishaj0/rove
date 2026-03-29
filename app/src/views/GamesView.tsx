import type { FormEvent } from "react";
import type { Game } from "../types";

type GamesViewProps = {
  games: Game[];
  newName: string;
  newPath: string;
  syncLabel: string;
  syncClass: "success" | "warning" | "error" | "syncing";
  onNewNameChange: (value: string) => void;
  onNewPathChange: (value: string) => void;
  onAddGame: (e: FormEvent) => void;
  onQuickExport: (gameId: string) => void;
  onQuickImport: (gameId: string) => void;
  onQuickEnableSync: () => void;
  onDeleteGame: (gameId: string) => void;
  initials: (name: string) => string;
};

export function GamesView(props: GamesViewProps) {
  const {
    games,
    newName,
    newPath,
    syncLabel,
    syncClass,
    onNewNameChange,
    onNewPathChange,
    onAddGame,
    onQuickExport,
    onQuickImport,
    onQuickEnableSync,
    onDeleteGame,
    initials,
  } = props;

  return (
    <>
      <section className="panel">
        <h2>Add New Game Folder</h2>
        <form className="add-game-form" onSubmit={onAddGame}>
          <input placeholder="Game name" value={newName} onChange={(e) => onNewNameChange(e.target.value)} />
          <input
            placeholder="Save folder path (e.g. D:\\Games\\EldenRing\\Save)"
            value={newPath}
            onChange={(e) => onNewPathChange(e.target.value)}
          />
          <button className="btn primary" type="submit">Add Game</button>
        </form>
      </section>

      <section className="panel">
        <h2>Tracked Games</h2>
        {games.length === 0 ? (
          <p className="empty-state">No games added yet.</p>
        ) : (
          <div className="game-grid">
            {games.map((g) => (
              <article className="game-card" key={g.id}>
                <div className="game-cover">
                  <span>{initials(g.name)}</span>
                  <span className={`game-state ${syncClass}`}>{syncLabel}</span>
                </div>
                <div className="game-meta">
                  <strong>{g.name}</strong>
                  <p title={g.path}>{g.path}</p>
                </div>
                <div className="game-actions">
                  <button className="btn" onClick={() => onQuickExport(g.id)} type="button">Export</button>
                  <button className="btn" onClick={() => onQuickImport(g.id)} type="button">Import</button>
                  <button className="btn primary" onClick={onQuickEnableSync} type="button">Enable Sync</button>
                </div>
                <div className="game-actions game-actions-danger">
                  <button className="btn danger" onClick={() => onDeleteGame(g.id)} type="button">Remove</button>
                </div>
              </article>
            ))}
          </div>
        )}
      </section>
    </>
  );
}
