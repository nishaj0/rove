import type { FormEvent } from "react";
import { X } from "lucide-react";

type AddGameDialogProps = {
  open: boolean;
  newName: string;
  newPath: string;
  newExecutablePath: string;
  newCoverImageData: string;
  newVersion: string;
  onNameChange: (value: string) => void;
  onVersionChange: (value: string) => void;
  onBrowseSaveFolder: () => void;
  onBrowseExecutable: () => void;
  onCoverChange: (value: string) => void;
  onSubmit: (e: FormEvent) => Promise<boolean>;
  onClose: () => void;
};

export function AddGameDialog(props: AddGameDialogProps) {
  const { open, newName, newPath, newExecutablePath, newCoverImageData, newVersion, onNameChange, onVersionChange, onBrowseSaveFolder, onBrowseExecutable, onCoverChange, onSubmit, onClose } = props;
  if (!open) return null;

  async function submit(e: FormEvent) {
    if (await onSubmit(e)) onClose();
  }

  const pickerKeys = (open: () => void) => (e: React.KeyboardEvent) => {
    if (e.key === "Enter" || e.key === " ") { e.preventDefault(); open(); }
  };

  return (
    <div className="scrim" role="presentation" onMouseDown={(e) => { if (e.target === e.currentTarget) onClose(); }}>
      <section className="dialog" role="dialog" aria-modal="true" aria-labelledby="add-game-title">
        <div className="dialog-head">
          <h2 id="add-game-title">Add a game</h2>
          <button className="icon-btn" type="button" onClick={onClose} aria-label="Close"><X aria-hidden="true" size={16} /></button>
        </div>
        <form className="stack" onSubmit={submit}>
          <label className="field"><span>Game name</span><input autoFocus value={newName} onChange={(e) => onNameChange(e.target.value)} required /></label>
          <label className="field"><span>Version (optional)</span><input value={newVersion} onChange={(e) => onVersionChange(e.target.value)} maxLength={80} /></label>
          <label className="field"><span>Save folder</span><input className="picker" placeholder="Choose the folder that holds your saves…" value={newPath} readOnly required onClick={onBrowseSaveFolder} onKeyDown={pickerKeys(onBrowseSaveFolder)} /></label>
          <label className="field"><span>Game executable</span><input className="picker" placeholder="Choose the game’s .exe…" value={newExecutablePath} readOnly required onClick={onBrowseExecutable} onKeyDown={pickerKeys(onBrowseExecutable)} /></label>
          {!newExecutablePath || newCoverImageData ? (
            <label className="field"><span>Cover image (optional)</span>
              <input type="file" accept="image/*" onChange={(e) => {
                const file = e.target.files?.[0];
                if (!file) { onCoverChange(""); return; }
                const reader = new FileReader();
                reader.onload = () => onCoverChange(typeof reader.result === "string" ? reader.result : "");
                reader.readAsDataURL(file);
              }} />
            </label>
          ) : <p className="hint">Rove will use the executable’s icon as the cover. You can change it later.</p>}
          {newCoverImageData ? <img className="cover-preview" src={newCoverImageData} alt="Cover preview" /> : null}
          <div className="dialog-actions">
            <button className="btn" type="button" onClick={onClose}>Cancel</button>
            <button className="btn primary" type="submit">Add game</button>
          </div>
        </form>
      </section>
    </div>
  );
}
