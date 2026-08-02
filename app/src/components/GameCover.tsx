import type { CSSProperties } from "react";
import type { Game } from "../types";

type GameCoverProps = { game: Pick<Game, "id" | "syncId" | "name" | "coverImageData">; className?: string; alt?: string };

function seed(value: string) { let hash = 2166136261; for (const char of value) { hash ^= char.charCodeAt(0); hash = Math.imul(hash, 16777619); } return hash >>> 0; }
function fallbackStyle(game: GameCoverProps["game"]): CSSProperties {
  const palettes = [["#49315d", "#bd722b"], ["#173c55", "#b05b3b"], ["#2d4b45", "#a56835"], ["#643550", "#c78a2b"]];
  const pair = palettes[seed(game.syncId || game.id || game.name) % palettes.length];
  return { background: `linear-gradient(150deg, ${pair[0]}, ${pair[1]})` };
}
export function GameCover({ game, className = "", alt }: GameCoverProps) {
  if (game.coverImageData) return <img className={className} src={game.coverImageData} alt={alt ?? `${game.name} cover`} />;
  return <div className={`${className} game-cover-fallback`} style={fallbackStyle(game)} aria-label={alt ?? `${game.name} generated cover`}>{game.name.slice(0, 2).toUpperCase()}</div>;
}
