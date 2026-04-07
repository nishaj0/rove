#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use std::{
    net::{SocketAddr, TcpStream},
    path::PathBuf,
    process::Command,
    time::Duration,
};
use tauri::Manager;

#[cfg(target_os = "windows")]
use std::os::windows::process::CommandExt;

const BACKEND_API_ADDR: &str = "127.0.0.1:7331";

#[cfg(target_os = "windows")]
const BACKEND_BINARY_NAME: &str = "cloud-saver-daemon.exe";

#[cfg(not(target_os = "windows"))]
const BACKEND_BINARY_NAME: &str = "cloud-saver-daemon";

#[cfg(target_os = "windows")]
const CREATE_NO_WINDOW: u32 = 0x08000000;

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_dialog::init())
        .setup(|app| {
            if let Err(err) = ensure_backend_running(app) {
                eprintln!("failed to ensure backend daemon is running: {err}");
            }
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

// avoids spawning extra daemon processes when one is already serving the local API.
fn backend_running() -> bool {
    let addr: SocketAddr = match BACKEND_API_ADDR.parse() {
        Ok(value) => value,
        Err(_) => return false,
    };
    TcpStream::connect_timeout(&addr, Duration::from_millis(250)).is_ok()
}

// resolves the packaged backend binary path for installed builds and local bundle runs.
fn locate_backend_binary(app: &tauri::App) -> Option<PathBuf> {
    let mut candidates: Vec<PathBuf> = Vec::new();

    if let Ok(resource_dir) = app.path().resource_dir() {
        candidates.push(resource_dir.join("backend").join(BACKEND_BINARY_NAME));
        candidates.push(resource_dir.join(BACKEND_BINARY_NAME));
    }

    if let Ok(current_exe) = std::env::current_exe() {
        if let Some(exe_dir) = current_exe.parent() {
            candidates.push(
                exe_dir
                    .join("resources")
                    .join("backend")
                    .join(BACKEND_BINARY_NAME),
            );
            candidates.push(exe_dir.join("backend").join(BACKEND_BINARY_NAME));
        }
    }

    candidates.into_iter().find(|path| path.is_file())
}

// starts the packaged backend daemon in background when the API is not already reachable.
fn ensure_backend_running(app: &tauri::App) -> Result<(), String> {
    if backend_running() {
        return Ok(());
    }

    let backend_path = locate_backend_binary(app)
        .ok_or_else(|| format!("could not find bundled backend binary '{BACKEND_BINARY_NAME}'"))?;

    let mut command = Command::new(&backend_path);
    apply_platform_process_flags(&mut command);

    command
        .spawn()
        .map(|_| ())
        .map_err(|err| format!("failed to launch backend from '{}': {err}", backend_path.display()))
}

#[cfg(target_os = "windows")]
// keeps daemon startup silent on Windows to avoid opening a separate console window.
fn apply_platform_process_flags(command: &mut Command) {
    command.creation_flags(CREATE_NO_WINDOW);
}

#[cfg(not(target_os = "windows"))]
// leaves process flags unchanged on non-Windows platforms.
fn apply_platform_process_flags(_command: &mut Command) {}
