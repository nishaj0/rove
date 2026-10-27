import { type FormEvent, useEffect, useMemo, useState } from "react";
import { Cloud, LayoutGrid, List, Plus, Search, X } from "lucide-react";
import { GameCover } from "../components/GameCover";
import type { Game, RecoveryLibraryEntry } from "../types";

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
  recoveryLibrary?: RecoveryLibraryEntry[];
  onRestoreRecovery?: (entry: RecoveryLibraryEntry) => void;
  cloudTarget?: string;
  onOpenSettings?: () => void;
  openAddSignal?: number;
};

const targetLabel: Record<string, string> = { local: "This PC only", google_drive: "Google Drive", folder: "Synced folder" };

function formatSize(bytes: number) {
  if (bytes < 1024 * 1024) return `${Math.max(1, Math.round(bytes / 1024))} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

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
    recoveryLibrary = [],
    onRestoreRecovery,
    cloudTarget = "local",
    onOpenSettings,
    openAddSignal = 0,
  } = props;

  const [searchTerm, setSearchTerm] = useState("");
  const [showAddModal, setShowAddModal] = useState(false);
  const [viewMode, setViewMode] = useState<"grid" | "list">("grid");

  useEffect(() => {
    if (openAddSignal > 0) setShowAddModal(true);
  }, [openAddSignal]);

  // A freshly restored game has no local revision yet, so match on its sync ID too.
  const cloudOnly = useMemo(() => {
    const tracked = new Set(games.map((game) => game.syncId));
    return recoveryLibrary.filter((entry) => !entry.gameId && !tracked.has(entry.syncId));
  }, [recoveryLibrary, games]);

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
          <div className="library-toolbar-actions">
            <div className="library-view-toggle" role="group" aria-label="Library view">
              <button className={`icon-btn ${viewMode === "grid" ? "active" : ""}`} onClick={() => setViewMode("grid")} type="button" aria-label="Grid view" aria-pressed={viewMode === "grid"} title="Grid view"><LayoutGrid aria-hidden="true" size={16} /></button>
              <button className={`icon-btn ${viewMode === "list" ? "active" : ""}`} onClick={() => setViewMode("list")} type="button" aria-label="List view" aria-pressed={viewMode === "list"} title="List view"><List aria-hidden="true" size={17} /></button>
            </div>
            <button className="btn action-outline" onClick={() => setShowAddModal(true)} type="button"><Plus aria-hidden="true" size={16} /> Add game</button>
          </div>
        </div>

        <div className="library-header">
          <h1>Library</h1>
          <p className="library-count">{filteredGames.length} game{filteredGames.length === 1 ? "" : "s"}</p>
          <button className="library-destination" type="button" onClick={onOpenSettings} title="Change where saves are kept">
            <Cloud aria-hidden="true" size={14} /> Saving to {targetLabel[cloudTarget] ?? "This PC only"}
          </button>
        </div>

        {games.length === 0 ? (
          <div className="library-empty">
            <h2>{cloudOnly.length > 0 ? "Welcome back" : "Protect your first game"}</h2>
            <p>{cloudOnly.length > 0 ? "Your cloud already has saves. Restore them below, or add a new game." : "Add a game's save folder and Rove keeps it backed up automatically."}</p>
            <button className="btn primary" type="button" onClick={() => setShowAddModal(true)}><Plus aria-hidden="true" size={16} /> Add a game</button>
          </div>
        ) : (
          viewMode === "grid" ? (
            <div className="library-grid">
              {filteredGames.map((g) => (
                <button className="library-card" key={g.id} onClick={() => onOpenGame(g.id)} type="button">
                  <GameCover game={g} className="library-card-image" />
                  <div className="library-card-name">{g.name}</div>
                </button>
              ))}
              <button className="library-card library-add-card" onClick={() => setShowAddModal(true)} type="button"><Plus aria-hidden="true" size={25} /><span>Add game</span></button>
            </div>
          ) : (
            <div className="library-list" aria-label="Games list">
              {filteredGames.map((g) => (
                <button className="library-list-row" key={g.id} onClick={() => onOpenGame(g.id)} type="button">
                  <GameCover game={g} className="library-list-cover" />
                  <span>{g.name}</span>
                </button>
              ))}
              <button className="library-list-row library-add-row" onClick={() => setShowAddModal(true)} type="button"><Plus aria-hidden="true" size={18} /><span>Add game</span></button>
            </div>
          )
        )}

        {cloudOnly.length > 0 ? (
          <div className="cloud-library" aria-label="Games in your cloud">
            <h2>In your cloud <span className="library-count">{cloudOnly.length}</span></h2>
            <p className="snapshot-meta">Saved from another PC or an earlier install. Restore to continue where you left off.</p>
            <div className="cloud-library-list">
              {cloudOnly.map((entry) => (
                <article className="cloud-library-row" key={entry.profileId}>
                  <div>
                    <strong>{entry.name}</strong>
                    <span className="snapshot-meta">{entry.version ? `v${entry.version} · ` : ""}{entry.fileCount} file{entry.fileCount === 1 ? "" : "s"} · {formatSize(entry.totalBytes)}</span>
                  </div>
                  {entry.requiresUpdate ? (
                    <span className="snapshot-meta" role="note">Needs a newer Rove</span>
                  ) : (
                    <button className="btn primary" type="button" onClick={() => onRestoreRecovery?.(entry)}>Restore</button>
                  )}
                </article>
              ))}
            </div>
          </div>
        ) : null}
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
