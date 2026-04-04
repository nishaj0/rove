import type { Game } from "../types";

type GameDetailViewProps = {
  game: Game | null;
  exportPath: string;
  importPath: string;
  mergeOnImport: boolean;
  exportOperationState: "idle" | "working" | "success" | "failed";
  exportOperationMessage: string;
  importOperationState: "idle" | "working" | "success" | "failed";
  importOperationMessage: string;
  onBack: () => void;
  onPlay: () => void;
  onSyncGame: () => void;
  onDelete: () => void;
  onBrowseExportPath: () => void;
  onBrowseImportPath: () => void;
  onMergeChange: (value: boolean) => void;
  onExportBackup: () => void;
  onImportBackup: () => void;
  onUpdateCover: (coverImageData: string) => void;
  formatPlaytime: (seconds: number) => string;
  safeDateTime: (value: string) => string;
};

export function GameDetailView(props: GameDetailViewProps) {
  const {
    game,
    exportPath,
    importPath,
    mergeOnImport,
    exportOperationState,
    exportOperationMessage,
    importOperationState,
    importOperationMessage,
    onBack,
    onPlay,
    onSyncGame,
    onDelete,
    onBrowseExportPath,
    onBrowseImportPath,
    onMergeChange,
    onExportBackup,
    onImportBackup,
    onUpdateCover,
    formatPlaytime,
    safeDateTime,
  } = props;

  const exportStateClass = exportOperationState === "working"
    ? "syncing"
    : exportOperationState === "success"
      ? "success"
      : "error";

  const importStateClass = importOperationState === "working"
    ? "syncing"
    : importOperationState === "success"
      ? "success"
      : "error";

  if (!game) {
    return (
      <section className="panel">
        <h2>Game Not Found</h2>
        <p className="empty-state">This game no longer exists. Return to Games to pick another one.</p>
        <button className="btn nav-back" onClick={onBack} type="button"><span aria-hidden="true">&#8592;</span> Back to Games</button>
      </section>
    );
  }

  return (
    <>
      <section className="panel game-detail-head">
        <div className="game-detail-nav">
          <button className="btn nav-back" onClick={onBack} type="button"><span aria-hidden="true">&#8592;</span> Back to Games</button>
        </div>
        <div className="game-detail-hero">
          <div className="game-detail-cover">
            {game.coverImageData ? <img src={game.coverImageData} alt={`${game.name} cover`} /> : <span>{game.name.slice(0, 2).toUpperCase()}</span>}
          </div>
          <div className="game-detail-meta">
            <h2>{game.name}</h2>
            <p title={game.path}>Save folder: {game.path}</p>
            <p title={game.executablePath}>Executable: {game.executablePath}</p>
            <p>Play time: {formatPlaytime(game.playtimeSeconds ?? 0)}</p>
            <p>Last played: {game.lastPlayedAt ? safeDateTime(game.lastPlayedAt) : "Never"}</p>
            <p>Status: <strong className={`state-text ${game.running ? "success" : "warning"}`}>{game.running ? "Running" : "Idle"}</strong></p>
          </div>
        </div>
        <div className="detail-actions">
          <button className="btn primary" onClick={onPlay} type="button">Play</button>
          <button className="btn" onClick={onSyncGame} type="button">Sync This Game</button>
          <button className="btn danger" onClick={onDelete} type="button">Remove Game</button>
        </div>
      </section>

      <section className="panel split">
        <div className="backup-pane">
          <h2>Export Save</h2>
          <label>
            Destination folder
            <input
              className="picker-input"
              placeholder="D:\\Backups"
              value={exportPath}
              readOnly
              disabled={exportOperationState === "working"}
              onClick={onBrowseExportPath}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  onBrowseExportPath();
                }
              }}
            />
          </label>
          <button className="btn primary backup-action" onClick={onExportBackup} type="button" disabled={exportOperationState === "working"}>
            {exportOperationState === "working" ? "Exporting..." : "Export Backup"}
          </button>
          {exportOperationState !== "idle" ? <p className={`backup-status state-text ${exportStateClass}`}>{exportOperationMessage}</p> : null}
        </div>

        <div className="backup-pane">
          <h2>Import Save</h2>
          <label>
            ZIP source path
            <input
              className="picker-input"
              placeholder="D:\\Backups\\my-game.zip"
              value={importPath}
              readOnly
              disabled={importOperationState === "working"}
              onClick={onBrowseImportPath}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  onBrowseImportPath();
                }
              }}
            />
          </label>
          <label className="inline-check">
            <input type="checkbox" checked={mergeOnImport} disabled={importOperationState === "working"} onChange={(e) => onMergeChange(e.target.checked)} />
            Merge into existing folder
          </label>
          <button className="btn primary backup-action" onClick={onImportBackup} type="button" disabled={importOperationState === "working"}>
            {importOperationState === "working" ? "Importing..." : "Import Backup"}
          </button>
          {importOperationState !== "idle" ? <p className={`backup-status state-text ${importStateClass}`}>{importOperationMessage}</p> : null}
        </div>
      </section>

      <section className="panel">
        <h2>Cover Image</h2>
        <label className="file-input-wrap">
          <span>Upload image</span>
          <input
            type="file"
            accept="image/*"
            onChange={(e) => {
              const file = e.target.files?.[0];
              if (!file) {
                return;
              }
              const reader = new FileReader();
              reader.onload = () => {
                if (typeof reader.result === "string") {
                  onUpdateCover(reader.result);
                }
              };
              reader.readAsDataURL(file);
            }}
          />
        </label>
      </section>
    </>
  );
}
