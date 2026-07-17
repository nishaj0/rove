import type { Game } from "../types";

type BackupViewProps = {
  games: Game[];
  exportGameId: string;
  exportPath: string;
  importGameId: string;
  importPath: string;
  mergeOnImport: boolean;
  onExportGameChange: (value: string) => void;
  onBrowseExportPath: () => void;
  onImportGameChange: (value: string) => void;
  onBrowseImportPath: () => void;
  onMergeChange: (value: boolean) => void;
  onExportBackup: () => void;
  onImportBackup: () => void;
};

export function BackupView(props: BackupViewProps) {
  const {
    games,
    exportGameId,
    exportPath,
    importGameId,
    importPath,
    mergeOnImport,
    onExportGameChange,
    onBrowseExportPath,
    onImportGameChange,
    onBrowseImportPath,
    onMergeChange,
    onExportBackup,
    onImportBackup,
  } = props;

  return (
    <section className="panel split" data-window-title="ZIP.TRANSFER">
      <div className="backup-pane">
        <h2>Export ZIP</h2>
        <label>
          Game
          <select value={exportGameId} onChange={(e) => onExportGameChange(e.target.value)}>
            <option value="">Select game</option>
            {games.map((g) => (
              <option key={g.id} value={g.id}>{g.name}</option>
            ))}
          </select>
        </label>
        <label>
          Destination folder
          <input
            className="picker-input"
            placeholder="D:\\Backups"
            value={exportPath}
            readOnly
            onClick={onBrowseExportPath}
          />
        </label>
        <button className="btn primary backup-action" onClick={onExportBackup} type="button">Export Backup</button>
      </div>

      <div className="backup-pane">
        <h2>Import ZIP</h2>
        <label>
          Game
          <select value={importGameId} onChange={(e) => onImportGameChange(e.target.value)}>
            <option value="">Select game</option>
            {games.map((g) => (
              <option key={g.id} value={g.id}>{g.name}</option>
            ))}
          </select>
        </label>
        <label>
          ZIP source path
          <input
            className="picker-input"
            placeholder="D:\\Backups\\my-game.zip"
            value={importPath}
            readOnly
            onClick={onBrowseImportPath}
          />
        </label>
        <label className="inline-check">
          <input type="checkbox" checked={mergeOnImport} onChange={(e) => onMergeChange(e.target.checked)} />
          Merge into existing folder
        </label>
        <button className="btn primary backup-action" onClick={onImportBackup} type="button">Import Backup</button>
      </div>
    </section>
  );
}
