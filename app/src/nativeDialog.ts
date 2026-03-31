import { invoke } from "@tauri-apps/api/core";

type DialogFilter = {
  name: string;
  extensions: string[];
};

type OpenDialogOptions = {
  title?: string;
  filters?: DialogFilter[];
  defaultPath?: string;
  multiple?: boolean;
  directory?: boolean;
};

type OpenDialogResult = string | string[] | null;

async function openSinglePath(options: OpenDialogOptions): Promise<string | null> {
  const result = await invoke<OpenDialogResult>("plugin:dialog|open", { options });
  if (typeof result === "string") {
    return result;
  }
  if (Array.isArray(result) && result.length > 0 && typeof result[0] === "string") {
    return result[0];
  }
  return null;
}

export function pickFolder(defaultPath?: string): Promise<string | null> {
  return openSinglePath({
    title: "Select Folder",
    directory: true,
    multiple: false,
    defaultPath,
  });
}

export function pickExecutable(defaultPath?: string): Promise<string | null> {
  return openSinglePath({
    title: "Select Game Executable",
    multiple: false,
    defaultPath,
    filters: [{ name: "Executable", extensions: ["exe", "bat", "cmd"] }],
  });
}

export function pickZipFile(defaultPath?: string): Promise<string | null> {
  return openSinglePath({
    title: "Select Backup ZIP",
    multiple: false,
    defaultPath,
    filters: [{ name: "ZIP", extensions: ["zip"] }],
  });
}
