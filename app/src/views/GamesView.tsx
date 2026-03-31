import type { FormEvent } from "react";
import type { Game } from "../types";

type GamesViewProps = {
  games: Game[];
  newName: string;
  newPath: string;
  newExecutablePath: string;
  newCoverImageData: string;
  onNewNameChange: (value: string) => void;
  onBrowseSaveFolder: () => void;
  onBrowseExecutable: () => void;
  onNewCoverImageChange: (value: string) => void;
  onAddGame: (e: FormEvent) => void;
  onOpenGame: (gameId: string) => void;
  initials: (name: string) => string;
  formatPlaytime: (seconds: number) => string;
};

export function GamesView(props: GamesViewProps) {
  const {
    games,
    newName,
    newPath,
    newExecutablePath,
    newCoverImageData,
    onNewNameChange,
    onBrowseSaveFolder,
    onBrowseExecutable,
    onNewCoverImageChange,
    onAddGame,
    onOpenGame,
    initials,
    formatPlaytime,
  } = props;

  return (
    <>
      <section className="panel">
        <h2>Add New Game Folder</h2>
        <form className="add-game-form" onSubmit={onAddGame}>
          <input placeholder="Game name" value={newName} onChange={(e) => onNewNameChange(e.target.value)} required />
          <input
            className="picker-input"
            placeholder="Save folder path (e.g. D:\\Games\\EldenRing\\Save)"
            value={newPath}
            readOnly
            onClick={onBrowseSaveFolder}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                onBrowseSaveFolder();
              }
            }}
            required
          />
          <input
            className="picker-input"
            placeholder="Executable path (e.g. D:\\Games\\EldenRing\\eldenring.exe)"
            value={newExecutablePath}
            readOnly
            onClick={onBrowseExecutable}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                onBrowseExecutable();
              }
            }}
            required
          />
          <label className="file-input-wrap">
            <span>Cover image (optional)</span>
            <input
              type="file"
              accept="image/*"
              onChange={(e) => {
                const file = e.target.files?.[0];
                if (!file) {
                  onNewCoverImageChange("");
                  return;
                }
                const reader = new FileReader();
                reader.onload = () => {
                  onNewCoverImageChange(typeof reader.result === "string" ? reader.result : "");
                };
                reader.readAsDataURL(file);
              }}
            />
          </label>
          {newCoverImageData ? <img className="cover-preview" src={newCoverImageData} alt="Cover preview" /> : null}
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
              <button className="game-card game-card-button" key={g.id} onClick={() => onOpenGame(g.id)} type="button">
                <div className="game-cover">
                  {g.coverImageData ? <img className="game-cover-image" src={g.coverImageData} alt={`${g.name} cover`} /> : <span>{initials(g.name)}</span>}
                  <span className={`game-state ${g.running ? "success" : "warning"}`}>{g.running ? "Running" : "Idle"}</span>
                </div>
                <div className="game-meta">
                  <div className="game-meta-main">
                    <strong>{g.name}</strong>
                    <p className="game-playtime">Play time: {formatPlaytime(g.playtimeSeconds ?? 0)}</p>
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
