import { type FormEvent, useMemo, useState } from "react";
import { Plus, Search, X } from "lucide-react";
import { GameCover } from "../components/GameCover";
import type { Game } from "../types";

type GamesViewProps = {
  games: Game[];
  newName: string;
  newPath: string;
  newExecutablePath: string;
  newCoverImageData: string;
  newVersion?: string;
  onNewNameChange: (value: string) => void;
  onBrowseSaveFolder: () => void;
  onBrowseExecutable: () => void;
  onNewCoverImageChange: (value: string) => void;
  onNewVersionChange: (value: string) => void;
  onAddGame: (e: FormEvent) => Promise<boolean>;
  onOpenGame: (gameId: string) => void;
  initials: (name: string) => string;
};

export function GamesView(props: GamesViewProps) {
  const {
    games,
    newName,
    newPath,
    newExecutablePath,
    newCoverImageData, newVersion = "",
    onNewNameChange,
    onBrowseSaveFolder,
    onBrowseExecutable,
    onNewCoverImageChange, onNewVersionChange,
    onAddGame,
    onOpenGame,
    initials,
  } = props;

  const [searchTerm, setSearchTerm] = useState("");
  const [showAddModal, setShowAddModal] = useState(false);

  const filteredGames = useMemo(() => {
    const query = searchTerm.trim().toLowerCase();
    if (!query) {
      return games;
    }
    return games.filter((game) => game.name.toLowerCase().includes(query));
  }, [games, searchTerm]);

  async function onSubmitAddGame(e: FormEvent) {
    const added = await onAddGame(e);
    if (added) {
      setShowAddModal(false);
    }
  }

  return (
    <>
      <section className="library-shell" data-window-title="GAMES.LIBRARY">
        <div className="library-toolbar">
          <label className="library-search">
            <Search aria-hidden="true" size={17} />
            <input
              type="search"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              placeholder="Search games"
            />
          </label>
          <button className="btn action-outline" onClick={() => setShowAddModal(true)} type="button"><Plus aria-hidden="true" size={16} /> Add game</button>
        </div>

        <div className="library-header">
          <h1>Library</h1>
          <p className="library-count">{filteredGames.length} game{filteredGames.length === 1 ? "" : "s"}</p>
        </div>

        {games.length === 0 ? (
          <p className="empty-state">No games added yet.</p>
        ) : (
          <div className="library-grid">
            {filteredGames.map((g) => (
              <button className="library-card" key={g.id} onClick={() => onOpenGame(g.id)} type="button">
                <GameCover game={g} className="library-card-image" />
                <div className="library-card-name">{g.name}</div>
              </button>
            ))}
            <button className="library-card library-add-card" onClick={() => setShowAddModal(true)} type="button"><Plus aria-hidden="true" size={25} /><span>Add game</span></button>
          </div>
        )}
      </section>

      {showAddModal ? (
        <div className="add-modal-backdrop" onClick={() => setShowAddModal(false)} role="presentation">
          <section className="add-modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-head">
              <h2>Add a game</h2>
              <button className="modal-close" onClick={() => setShowAddModal(false)} type="button" aria-label="Close"><X aria-hidden="true" size={18} /></button>
            </div>
            <form className="add-game-form" onSubmit={onSubmitAddGame}>
              <input placeholder="Game name" value={newName} onChange={(e) => onNewNameChange(e.target.value)} required />
              <input placeholder="Version (optional)" value={newVersion} onChange={(e) => onNewVersionChange(e.target.value)} maxLength={80} />
              <input
                className="picker-input"
                placeholder="Save folder"
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
                placeholder="Executable"
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
              {!newExecutablePath || newCoverImageData ? <label className="file-input-wrap">
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
              </label> : <p className="snapshot-meta">Rove will use this executable's icon when it can; you can replace it later.</p>}
              {newCoverImageData ? <img className="cover-preview" src={newCoverImageData} alt="Cover preview" /> : null}
              <div className="modal-actions">
                <button className="btn" onClick={() => setShowAddModal(false)} type="button">Cancel</button>
                <button className="btn action-outline" type="submit">Add Game</button>
              </div>
            </form>
          </section>
        </div>
      ) : null}
    </>
  );
}
