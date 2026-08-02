#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use std::{
    net::{SocketAddr, TcpStream},
    path::PathBuf,
    process::Command,
    sync::Mutex,
    time::Duration,
};
use tauri::Manager;
use tauri_plugin_autostart::ManagerExt;

mod updater;

#[cfg(target_os = "windows")]
use std::os::windows::process::CommandExt;

#[cfg(not(debug_assertions))]
use std::io::{Read, Write};

const BACKEND_API_ADDR: &str = "127.0.0.1:7331";

#[cfg(target_os = "windows")]
const BACKEND_BINARY_NAME: &str = "cloud-saver-daemon.exe";

#[cfg(not(target_os = "windows"))]
const BACKEND_BINARY_NAME: &str = "cloud-saver-daemon";

#[cfg(target_os = "windows")]
const CREATE_NO_WINDOW: u32 = 0x08000000;

#[tauri::command]
fn set_autostart(app: tauri::AppHandle, enabled: bool) -> Result<bool, String> {
    let manager = app.autolaunch();
    if enabled {
        manager.enable().map_err(|err| err.to_string())?;
    } else {
        manager.disable().map_err(|err| err.to_string())?;
    }
    manager.is_enabled().map_err(|err| err.to_string())
}

#[tauri::command]
fn is_autostart_enabled(app: tauri::AppHandle) -> Result<bool, String> {
    app.autolaunch().is_enabled().map_err(|err| err.to_string())
}

#[derive(Default)]
struct DaemonBootstrapState(Mutex<Option<String>>);

#[tauri::command]
fn daemon_bootstrap_error(state: tauri::State<'_, DaemonBootstrapState>) -> Option<String> {
    state.0.lock().unwrap().clone()
}

fn main() {
    tauri::Builder::default()
        .invoke_handler(tauri::generate_handler![
            set_autostart,
            is_autostart_enabled,
            daemon_bootstrap_error,
            updater::update_status,
            updater::check_for_update,
            updater::download_update,
            updater::install_downloaded_update,
            updater::discard_downloaded_update,
            updater::set_automatic_update_checks,
        ])
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .manage(DaemonBootstrapState::default())
        .plugin(
            tauri_plugin_single_instance::Builder::new()
                .callback(|app, argv, _cwd| {
                    if argv.iter().any(|arg| arg == "--background") {
                        return;
                    }
                    if let Some(window) = app.get_webview_window("main") {
                        let _ = window.show();
                        let _ = window.unminimize();
                        let _ = window.set_focus();
                    }
                })
                .build(),
        )
        .setup(|app| {
            app.manage(updater::UpdateManager::load(app.handle()));
            app.handle()
                .plugin(tauri_plugin_autostart::init(
                    tauri_plugin_autostart::MacosLauncher::LaunchAgent,
                    Some(vec!["--background"]),
                ))
                .map_err(|err| Box::<dyn std::error::Error>::from(err))?;
            if let Err(err) = ensure_backend_running(app.handle()) {
                eprintln!("failed to ensure backend daemon is running: {err}");
                *app.state::<DaemonBootstrapState>().0.lock().unwrap() = Some(format!(
                    "Rove could not start a compatible background service: {err}. Your saves and settings were not changed."
                ));
            }
            if std::env::args().any(|arg| arg == "--background") {
                if let Some(window) = app.get_webview_window("main") {
                    let _ = window.hide();
                }
                app.handle().exit(0);
            }
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

// avoids spawning extra daemon processes when one is already serving the local API.
pub(crate) fn backend_running() -> bool {
    let addr: SocketAddr = match BACKEND_API_ADDR.parse() {
        Ok(value) => value,
        Err(_) => return false,
    };
    TcpStream::connect_timeout(&addr, Duration::from_millis(250)).is_ok()
}

// resolves the packaged backend binary path for installed builds and local bundle runs.
fn locate_backend_binary(app: &tauri::AppHandle) -> Option<PathBuf> {
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
pub(crate) fn ensure_backend_running(app: &tauri::AppHandle) -> Result<(), String> {
    if backend_running() {
        #[cfg(debug_assertions)]
        return Ok(());

        #[cfg(not(debug_assertions))]
        {
            let desktop_version = app.package_info().version.to_string();
            let daemon_version = backend_version()?;
            if daemon_version == desktop_version {
                return Ok(());
            }
            request_backend_restart()?;
            let started = std::time::Instant::now();
            while backend_running() && started.elapsed() < Duration::from_secs(10) {
                std::thread::sleep(Duration::from_millis(150));
            }
            if backend_running() {
                return Err(format!(
                    "background service version {daemon_version} is still running; expected {desktop_version}"
                ));
            }
        }
    }

    let backend_path = locate_backend_binary(app)
        .ok_or_else(|| format!("could not find bundled backend binary '{BACKEND_BINARY_NAME}'"))?;

    let mut command = Command::new(&backend_path);
    apply_platform_process_flags(&mut command);

    command.spawn().map(|_| ()).map_err(|err| {
        format!(
            "failed to launch backend from '{}': {err}",
            backend_path.display()
        )
    })
}

#[cfg(not(debug_assertions))]
fn backend_version() -> Result<String, String> {
    let response = backend_http_request(
        "GET /version HTTP/1.1\r\nHost: 127.0.0.1:7331\r\nConnection: close\r\n\r\n",
    )?;
    if !response
        .lines()
        .next()
        .unwrap_or_default()
        .contains(" 200 ")
    {
        return Err(
            "the running background service does not expose compatible version information".into(),
        );
    }
    let body = response.split("\r\n\r\n").nth(1).unwrap_or_default();
    let value: serde_json::Value = serde_json::from_str(body).map_err(|err| err.to_string())?;
    value
        .get("version")
        .and_then(|version| version.as_str())
        .map(str::to_owned)
        .ok_or_else(|| "the running background service returned no version".into())
}

#[cfg(not(debug_assertions))]
fn request_backend_restart() -> Result<(), String> {
    let response = backend_http_request(concat!(
        "POST /lifecycle/prepare-update HTTP/1.1\r\n",
        "Host: 127.0.0.1:7331\r\n",
        "Content-Length: 0\r\n",
        "Connection: close\r\n\r\n"
    ))?;
    if response
        .lines()
        .next()
        .unwrap_or_default()
        .contains(" 202 ")
    {
        Ok(())
    } else {
        Err("the stale background service could not be restarted safely; close tracked games and restart Rove".into())
    }
}

#[cfg(not(debug_assertions))]
fn backend_http_request(request: &str) -> Result<String, String> {
    let addr: SocketAddr = BACKEND_API_ADDR
        .parse()
        .map_err(|err| format!("invalid background service address: {err}"))?;
    let mut stream =
        TcpStream::connect_timeout(&addr, Duration::from_secs(2)).map_err(|err| err.to_string())?;
    stream
        .set_read_timeout(Some(Duration::from_secs(3)))
        .map_err(|err| err.to_string())?;
    stream
        .write_all(request.as_bytes())
        .map_err(|err| err.to_string())?;
    let mut response = String::new();
    stream
        .read_to_string(&mut response)
        .map_err(|err| err.to_string())?;
    Ok(response)
}

#[cfg(target_os = "windows")]
// keeps daemon startup silent on Windows to avoid opening a separate console window.
fn apply_platform_process_flags(command: &mut Command) {
    command.creation_flags(CREATE_NO_WINDOW);
}

#[cfg(not(target_os = "windows"))]
// leaves process flags unchanged on non-Windows platforms.
fn apply_platform_process_flags(_command: &mut Command) {}
