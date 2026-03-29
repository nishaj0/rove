import type { Game } from "../types";

type BackupViewProps = {
  games: Game[];
  exportGameId: string;
  exportPath: string;
  importGameId: string;
  importPath: string;
  mergeOnImport: boolean;
  onExportGameChange: (value: string) => void;
  onExportPathChange: (value: string) => void;
  onImportGameChange: (value: string) => void;
  onImportPathChange: (value: string) => void;
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
    onExportPathChange,
    onImportGameChange,
    onImportPathChange,
    onMergeChange,
    onExportBackup,
    onImportBackup,
  } = props;

  return (
    <section className="panel split">
      <div>
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
          Destination path
          <input
            placeholder="D:\\Backups\\my-game.zip"
            value={exportPath}
            onChange={(e) => onExportPathChange(e.target.value)}
          />
        </label>
        <button className="btn primary" onClick={onExportBackup} type="button">Export Backup</button>
      </div>

      <div>
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
            placeholder="D:\\Backups\\my-game.zip"
            value={importPath}
            onChange={(e) => onImportPathChange(e.target.value)}
          />
        </label>
        <label className="inline-check">
          <input type="checkbox" checked={mergeOnImport} onChange={(e) => onMergeChange(e.target.checked)} />
          Merge into existing folder
        </label>
        <button className="btn primary" onClick={onImportBackup} type="button">Import Backup</button>
      </div>
    </section>
  );
}
