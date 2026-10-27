import { ChevronRight, Home, Plus, Settings as SettingsIcon, CloudDownload } from "lucide-react";
import { GameCover } from "./GameCover";
import type { Game, RecoveryLibraryEntry } from "../types";
import { cloudOnlyEntries, gameStatus } from "../utils/gameStatus";

export type AppView = "home" | "game" | "settings";

type SidebarProps = {
  view: AppView;
  games: Game[];
  selectedGameId: string;
  query: string;
  recoveryLibrary: RecoveryLibraryEntry[];
  onHome: () => void;
  onOpenGame: (id: string) => void;
  onAddGame: () => void;
  onRestore: (entry: RecoveryLibraryEntry) => void;
  onOpenSettings: () => void;
};

export function Sidebar(props: SidebarProps) {
  const { view, games, selectedGameId, query, recoveryLibrary, onHome, onOpenGame, onAddGame, onRestore, onOpenSettings } = props;
  const needle = query.trim().toLowerCase();
  const visible = needle ? games.filter((game) => game.name.toLowerCase().includes(needle)) : games;
  const cloud = cloudOnlyEntries(recoveryLibrary, games).filter((entry) => !needle || entry.name.toLowerCase().includes(needle));

  return (
    <aside className="sidebar" aria-label="Library">
      <nav className="side-nav">
        <button className="side-item" type="button" aria-current={view === "home" ? "page" : undefined} onClick={onHome}>
          <Home aria-hidden="true" size={16} /> Home
        </button>
      </nav>

      <div className="side-head">
        <span>Library · {games.length}</span>
        <button className="icon-btn" type="button" onClick={onAddGame} aria-label="Add a game" title="Add a game"><Plus aria-hidden="true" size={16} /></button>
      </div>

      <div className="side-scroll">
        {games.length === 0 ? (
          <p className="side-empty">No games yet. Add one to start protecting its saves.</p>
        ) : visible.length === 0 ? (
          <p className="side-empty">No games match “{query}”.</p>
        ) : (
          <ul className="game-list">
            {visible.map((game) => {
              const status = gameStatus(game);
              return (
                <li key={game.id}>
                  <button className="game-row" type="button" aria-current={view === "game" && selectedGameId === game.id ? "page" : undefined} onClick={() => onOpenGame(game.id)}>
                    <GameCover game={game} className="game-thumb" alt="" />
                    <span className="game-meta"><strong>{game.name}</strong><span>{status.text}</span></span>
                    <span className={`dot ${status.tone === "idle" ? "" : status.tone}`} aria-hidden="true" />
                  </button>
                </li>
              );
            })}
          </ul>
        )}

        {cloud.length > 0 ? (
          <details className="side-section" open>
            <summary><ChevronRight aria-hidden="true" size={14} /> In your cloud <span className="count">{cloud.length}</span></summary>
            <ul className="game-list">
              {cloud.map((entry) => (
                <li className="cloud-row" key={entry.profileId}>
                  <span className="game-meta"><strong>{entry.name}</strong><span>{entry.requiresUpdate ? "Needs a newer Rove" : entry.version ? `v${entry.version}` : "Not on this PC"}</span></span>
                  {entry.requiresUpdate ? null : (
                    <button className="icon-btn" type="button" onClick={() => onRestore(entry)} aria-label={`Restore ${entry.name}`} title="Restore to this PC"><CloudDownload aria-hidden="true" size={16} /></button>
                  )}
                </li>
              ))}
            </ul>
          </details>
        ) : null}
      </div>

      <div className="side-foot">
        <button className="side-item" type="button" aria-current={view === "settings" ? "page" : undefined} onClick={onOpenSettings}>
          <SettingsIcon aria-hidden="true" size={16} /> Settings
        </button>
      </div>
    </aside>
  );
}
