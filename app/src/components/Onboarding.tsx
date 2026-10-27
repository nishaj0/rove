import { useState } from "react";
import { Check, Cloud, FolderSync, HardDrive } from "lucide-react";

type Destination = "local" | "google_drive" | "folder";

type OnboardingProps = {
  destination: Destination;
  driveConnected: boolean;
  syncRoot: string;
  cloudGameCount: number;
  onChooseLocal: () => Promise<void>;
  onConnectDrive: () => Promise<void>;
  onChooseFolder: () => Promise<void>;
  onAddGame: () => void;
  onFinish: () => void;
};

const STEPS = ["Welcome", "Save location", "First game"] as const;

const destinationCards: Array<{ id: Destination; title: string; body: string; Icon: typeof Cloud }> = [
  { id: "local", title: "This PC only", body: "No account needed. Saves are versioned locally but do not survive a disk failure.", Icon: HardDrive },
  { id: "google_drive", title: "Google Drive", body: "Direct upload to your own Drive. Rove only sees files it creates.", Icon: Cloud },
  { id: "folder", title: "Synced folder", body: "Use OneDrive, Dropbox, Syncthing or any folder another app keeps in sync.", Icon: FolderSync },
];

export function Onboarding(props: OnboardingProps) {
  const { destination, driveConnected, syncRoot, cloudGameCount, onChooseLocal, onConnectDrive, onChooseFolder, onAddGame, onFinish } = props;
  const [step, setStep] = useState(0);
  const [busy, setBusy] = useState<Destination | null>(null);

  async function choose(id: Destination) {
    setBusy(id);
    try {
      if (id === "local") await onChooseLocal();
      else if (id === "google_drive") await onConnectDrive();
      else await onChooseFolder();
    } finally {
      setBusy(null);
    }
  }

  const ready = destination === "local" || (destination === "google_drive" && driveConnected) || (destination === "folder" && syncRoot.trim() !== "");

  return (
    <div className="add-modal-backdrop" role="presentation">
      <section className="add-modal onboarding" role="dialog" aria-modal="true" aria-labelledby="onboarding-title">
        <ol className="onboarding-steps" aria-label="Setup progress">
          {STEPS.map((label, index) => (
            <li key={label} className={index === step ? "active" : index < step ? "done" : ""} aria-current={index === step ? "step" : undefined}>
              <span>{index < step ? <Check aria-hidden="true" size={13} /> : index + 1}</span>
              {label}
            </li>
          ))}
        </ol>

        {step === 0 ? (
          <div className="onboarding-body">
            <h2 id="onboarding-title">Cloud saves for any game</h2>
            <p>Rove watches your games' save folders and keeps safe, versioned copies, so a crash, a reinstall or a new PC never costs you progress.</p>
            <ul className="onboarding-points">
              <li>Backups happen in the background and stay light while you play.</li>
              <li>You choose where saves live. There is no Rove server and no account.</li>
              <li>On a new PC, restore your games from the cloud in a couple of clicks.</li>
            </ul>
          </div>
        ) : null}

        {step === 1 ? (
          <div className="onboarding-body">
            <h2 id="onboarding-title">Where should saves be kept?</h2>
            <p>You can change this any time in Settings.</p>
            <div className="onboarding-choices" role="radiogroup" aria-label="Save location">
              {destinationCards.map(({ id, title, body, Icon }) => {
                const selected = destination === id && (id !== "google_drive" || driveConnected) && (id !== "folder" || syncRoot.trim() !== "");
                return (
                  <button key={id} type="button" role="radio" aria-checked={selected} className={`onboarding-choice ${selected ? "selected" : ""}`} disabled={busy !== null} onClick={() => void choose(id)}>
                    <Icon aria-hidden="true" size={22} />
                    <strong>{title}</strong>
                    <span>{body}</span>
                    {id === "google_drive" && selected ? <em>Connected</em> : null}
                    {id === "folder" && selected ? <em>{syncRoot}</em> : null}
                    {busy === id ? <em>Working…</em> : null}
                  </button>
                );
              })}
            </div>
          </div>
        ) : null}

        {step === 2 ? (
          <div className="onboarding-body">
            <h2 id="onboarding-title">Add your first game</h2>
            {cloudGameCount > 0 ? (
              <p>Rove found {cloudGameCount} game{cloudGameCount === 1 ? "" : "s"} already saved in this location. Finish setup, then use <strong>Restore from cloud</strong> in your Library.</p>
            ) : (
              <p>Pick the game's executable and its save folder. Rove starts protecting it right away.</p>
            )}
          </div>
        ) : null}

        <div className="modal-actions">
          {step === 0 ? <button className="btn" type="button" onClick={onFinish}>Skip setup</button> : <button className="btn" type="button" onClick={() => setStep(step - 1)}>Back</button>}
          {step < 2 ? (
            <button className="btn primary" type="button" disabled={step === 1 && !ready} onClick={() => setStep(step + 1)}>Continue</button>
          ) : (
            <>
              {cloudGameCount > 0 ? <button className="btn" type="button" onClick={onFinish}>Go to Library</button> : null}
              <button className="btn primary" type="button" onClick={() => { onFinish(); onAddGame(); }}>Add a game</button>
            </>
          )}
        </div>
      </section>
    </div>
  );
}
