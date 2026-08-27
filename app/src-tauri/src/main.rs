#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use std::{
    net::{SocketAddr, TcpStream},
    process::Command,
    sync::Mutex,
    time::Duration,
};
use tauri::{Emitter, Manager};
use tauri_plugin_autostart::ManagerExt;

mod updater;

#[cfg(target_os = "windows")]
use std::os::windows::process::CommandExt;
#[cfg(target_os = "windows")]
use std::path::PathBuf;

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

#[derive(Default)]
struct LaunchGameState(Mutex<Option<String>>);

#[tauri::command]
fn daemon_bootstrap_error(state: tauri::State<'_, DaemonBootstrapState>) -> Option<String> {
    state.0.lock().unwrap().clone()
}

#[tauri::command]
fn take_launch_game(state: tauri::State<'_, LaunchGameState>) -> Option<String> {
    state.0.lock().unwrap().take()
}

#[tauri::command]
fn create_game_shortcut(game_id: String, game_name: String) -> Result<String, String> {
    let game_id =
        valid_game_id(&game_id).ok_or_else(|| "invalid game shortcut request".to_string())?;

    #[cfg(target_os = "windows")]
    {
        let target =
            std::env::current_exe().map_err(|err| format!("could not find Rove: {err}"))?;
        let desktop = std::env::var_os("USERPROFILE")
            .map(PathBuf::from)
            .map(|path| path.join("Desktop"))
            .filter(|path| path.is_dir())
            .ok_or_else(|| "could not find your Desktop folder".to_string())?;
        let shortcut = desktop.join(format!("{} — Rove.lnk", shortcut_file_stem(&game_name)));
        let working_dir = target
            .parent()
            .ok_or_else(|| "could not resolve Rove's folder".to_string())?;
        create_windows_shortcut(&target, working_dir, &shortcut, &game_id)?;
        return Ok(shortcut.display().to_string());
    }

    #[cfg(not(target_os = "windows"))]
    {
        let _ = game_name;
        Err("Rove game shortcuts are currently available on Windows only".into())
    }
}

fn valid_game_id(value: &str) -> Option<String> {
    let trimmed = value.trim();
    if trimmed.is_empty()
        || trimmed.len() > 128
        || !trimmed
            .bytes()
            .all(|byte| byte.is_ascii_alphanumeric() || byte == b'-' || byte == b'_')
    {
        return None;
    }
    Some(trimmed.to_string())
}

fn launch_game_arg(args: &[String]) -> Option<String> {
    args.windows(2)
        .find(|pair| pair[0] == "--launch-game")
        .and_then(|pair| valid_game_id(&pair[1]))
}

fn shortcut_file_stem(game_name: &str) -> String {
    let sanitized: String = game_name
        .trim()
        .chars()
        .map(|character| {
            if matches!(
                character,
                '<' | '>' | ':' | '"' | '/' | '\\' | '|' | '?' | '*'
            ) || character.is_control()
            {
                '_'
            } else {
                character
            }
        })
        .collect();
    let trimmed =
        sanitized.trim_matches(|character: char| character == '.' || character.is_whitespace());
    if trimmed.is_empty() {
        "Game".to_string()
    } else {
        trimmed.chars().take(100).collect()
    }
}

#[cfg(target_os = "windows")]
fn powershell_literal(value: &std::path::Path) -> String {
    format!("'{}'", value.display().to_string().replace('\'', "''"))
}

#[cfg(target_os = "windows")]
fn create_windows_shortcut(
    target: &std::path::Path,
    working_dir: &std::path::Path,
    shortcut: &std::path::Path,
    game_id: &str,
) -> Result<(), String> {
    let arguments = format!("--launch-game {game_id}");
    let arguments_literal = format!("'{}'", arguments.replace('\'', "''"));
    let script = format!(
        "$shell = New-Object -ComObject WScript.Shell; $shortcut = $shell.CreateShortcut({}); $shortcut.TargetPath = {}; $shortcut.Arguments = {}; $shortcut.WorkingDirectory = {}; $shortcut.IconLocation = {}; $shortcut.Save()",
        powershell_literal(shortcut),
        powershell_literal(target),
        arguments_literal,
        powershell_literal(working_dir),
        powershell_literal(target),
    );
    let mut command = Command::new("powershell.exe");
    command.args([
        "-NoProfile",
        "-NonInteractive",
        "-ExecutionPolicy",
        "Bypass",
        "-Command",
        &script,
    ]);
    apply_platform_process_flags(&mut command);
    let output = command
        .output()
        .map_err(|err| format!("could not create Rove shortcut: {err}"))?;
    if output.status.success() {
        Ok(())
    } else {
        Err(format!(
            "could not create Rove shortcut: {}",
            String::from_utf8_lossy(&output.stderr).trim()
        ))
    }
}

#[cfg(test)]
mod tests {
    use super::{launch_game_arg, shortcut_file_stem};

    #[test]
    fn reads_only_a_valid_game_launch_argument() {
        let args = vec![
            "Rove.exe".to_string(),
            "--launch-game".to_string(),
            "game_123-abc".to_string(),
        ];
        assert_eq!(launch_game_arg(&args).as_deref(), Some("game_123-abc"));
        assert_eq!(
            launch_game_arg(&["Rove.exe".to_string(), "--launch-game".to_string()]),
            None
        );
        assert_eq!(
            launch_game_arg(&[
                "Rove.exe".to_string(),
                "--launch-game".to_string(),
                "../bad".to_string()
            ]),
            None
        );
    }

    #[test]
    fn creates_safe_shortcut_file_names() {
        assert_eq!(shortcut_file_stem("  Wolf: A/B?  "), "Wolf_ A_B_");
        assert_eq!(shortcut_file_stem("..."), "Game");
    }
}

fn main() {
    let startup_args: Vec<String> = std::env::args().collect();
    let startup_launch_game = launch_game_arg(&startup_args);
    tauri::Builder::default()
        .invoke_handler(tauri::generate_handler![
            set_autostart,
            is_autostart_enabled,
            daemon_bootstrap_error,
			take_launch_game,
			create_game_shortcut,
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
		.manage(LaunchGameState(Mutex::new(startup_launch_game)))
        .plugin(
            tauri_plugin_single_instance::Builder::new()
                .callback(|app, argv, _cwd| {
                    if argv.iter().any(|arg| arg == "--background") {
                        return;
                    }
				if let Some(game_id) = launch_game_arg(&argv) {
					*app.state::<LaunchGameState>().0.lock().unwrap() = Some(game_id.clone());
					let _ = app.emit("rove://launch-game", game_id);
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
