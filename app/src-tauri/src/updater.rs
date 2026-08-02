use std::{
    fs,
    io::{Read, Write},
    net::{SocketAddr, TcpStream},
    path::{Path, PathBuf},
    process::Command,
    sync::{
        atomic::{AtomicBool, Ordering},
        Mutex,
    },
    time::{Duration, SystemTime, UNIX_EPOCH},
};

use base64::Engine as _;
use futures_util::StreamExt;
use minisign_verify::{PublicKey, Signature};
use semver::Version;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use tauri::{AppHandle, Emitter, Manager, State};
use tauri_plugin_updater::{Update, UpdaterExt};

use crate::{backend_running, ensure_backend_running, BACKEND_API_ADDR};

const UPDATE_ENDPOINT: &str =
    "https://github.com/nishaj0/rove/releases/latest/download/latest.json";
// Tauri static updater manifests identify Windows by OS and architecture. The
// installer format belongs to the release asset, not to this platform key.
const UPDATE_TARGET: &str = "windows-x86_64";
const UPDATE_STATE_SCHEMA: u32 = 1;
const CHECK_CACHE_SECONDS: u64 = 6 * 60 * 60;
const CHECK_TIMEOUT: Duration = Duration::from_secs(3);
const DOWNLOAD_TIMEOUT: Duration = Duration::from_secs(10 * 60);
const DOWNLOAD_NO_PROGRESS_TIMEOUT: Duration = Duration::from_secs(45);
const DAEMON_SHUTDOWN_TIMEOUT: Duration = Duration::from_secs(15);

#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct UpdateState {
    pub lifecycle: String,
    pub current_version: String,
    pub available_version: Option<String>,
    pub notes: Option<String>,
    pub downloaded_bytes: u64,
    pub total_bytes: Option<u64>,
    pub last_checked_at: Option<u64>,
    pub downloaded_at: Option<u64>,
    pub retryable: bool,
    pub error_code: Option<String>,
    pub error: Option<String>,
    pub automatic_checks_enabled: bool,
    pub development: bool,
}

impl UpdateState {
    fn initial(current_version: String) -> Self {
        let development = cfg!(debug_assertions) || updater_public_key().trim().is_empty();
        Self {
            lifecycle: if development { "development" } else { "idle" }.into(),
            current_version,
            available_version: None,
            notes: None,
            downloaded_bytes: 0,
            total_bytes: None,
            last_checked_at: None,
            downloaded_at: None,
            retryable: false,
            error_code: None,
            error: None,
            automatic_checks_enabled: true,
            development,
        }
    }
}

#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
struct PersistedUpdateState {
    schema_version: u32,
    #[serde(flatten)]
    view: UpdateState,
    signature: Option<String>,
    download_url: Option<String>,
    cache_path: Option<PathBuf>,
    sha256: Option<String>,
}

struct RuntimeUpdateState {
    persisted: PersistedUpdateState,
    pending: Option<Update>,
}

pub struct UpdateManager {
    inner: Mutex<RuntimeUpdateState>,
    busy: AtomicBool,
    state_path: PathBuf,
    cache_dir: PathBuf,
}

impl UpdateManager {
    pub fn load(app: &AppHandle) -> Self {
        let current_version = app.package_info().version.to_string();
        let cache_dir = app
            .path()
            .app_cache_dir()
            .unwrap_or_else(|_| std::env::temp_dir().join("CloudSaver"))
            .join("updates");
        let state_path = cache_dir.join("state.json");
        let _ = fs::create_dir_all(&cache_dir);

        // An interrupted download is never considered installable.
        if let Ok(entries) = fs::read_dir(&cache_dir) {
            for entry in entries.flatten() {
                let path = entry.path();
                if path.extension().and_then(|v| v.to_str()) == Some("part") {
                    let _ = fs::remove_file(path);
                }
            }
        }

        let mut persisted =
            load_persisted_state(&state_path).unwrap_or_else(|| PersistedUpdateState {
                schema_version: UPDATE_STATE_SCHEMA,
                view: UpdateState::initial(current_version.clone()),
                signature: None,
                download_url: None,
                cache_path: None,
                sha256: None,
            });

        persisted.view.current_version = current_version.clone();
        persisted.view.development =
            cfg!(debug_assertions) || updater_public_key().trim().is_empty();
        if persisted.view.development {
            persisted.view.lifecycle = "development".into();
            persisted.view.error = None;
            persisted.view.error_code = None;
        } else if candidate_is_not_newer(
            persisted.view.available_version.as_deref(),
            &current_version,
        ) {
            discard_candidate_files(&persisted);
            clear_candidate(&mut persisted, "idle");
        } else if matches!(
            persisted.view.lifecycle.as_str(),
            "checking" | "downloading" | "installing"
        ) {
            if verified_cache_exists(&persisted) {
                persisted.view.lifecycle = "ready".into();
            } else if persisted.view.available_version.is_some() {
                persisted.view.lifecycle = "available".into();
                persisted.view.downloaded_bytes = 0;
                persisted.view.total_bytes = None;
                persisted.view.error_code = Some("update_interrupted".into());
                persisted.view.error = Some(
                    "The previous update operation was interrupted. You can retry safely.".into(),
                );
                persisted.view.retryable = true;
            } else {
                clear_candidate(&mut persisted, "idle");
            }
        } else if persisted.view.lifecycle == "ready" && !verified_cache_exists(&persisted) {
            clear_candidate(&mut persisted, "idle");
        }

        let manager = Self {
            inner: Mutex::new(RuntimeUpdateState {
                persisted,
                pending: None,
            }),
            busy: AtomicBool::new(false),
            state_path,
            cache_dir,
        };
        manager.persist();
        manager
    }

    fn snapshot(&self) -> UpdateState {
        self.inner.lock().unwrap().persisted.view.clone()
    }

    fn persist(&self) {
        let persisted = self.inner.lock().unwrap().persisted.clone();
        let _ = persist_state(&self.state_path, &persisted);
    }

    fn emit(&self, app: &AppHandle) {
        let _ = app.emit("cloud-saver://update-progress", self.snapshot());
    }

    fn begin(&self) -> Result<(), String> {
        self.busy
            .compare_exchange(false, true, Ordering::SeqCst, Ordering::SeqCst)
            .map(|_| ())
            .map_err(|_| "Another update operation is already running.".to_string())
    }

    fn finish(&self) {
        self.busy.store(false, Ordering::SeqCst);
    }
}

#[tauri::command]
pub fn update_status(manager: State<'_, UpdateManager>) -> UpdateState {
    manager.snapshot()
}

#[tauri::command]
pub fn set_automatic_update_checks(
    enabled: bool,
    manager: State<'_, UpdateManager>,
) -> UpdateState {
    manager
        .inner
        .lock()
        .unwrap()
        .persisted
        .view
        .automatic_checks_enabled = enabled;
    manager.persist();
    manager.snapshot()
}

#[tauri::command]
pub async fn check_for_update(
    app: AppHandle,
    manager: State<'_, UpdateManager>,
    force: Option<bool>,
) -> Result<UpdateState, String> {
    if manager.snapshot().development {
        return Ok(manager.snapshot());
    }
    let force = force.unwrap_or(false);
    let before = manager.snapshot();
    if !force && !before.automatic_checks_enabled {
        return Ok(before);
    }
    if !force
        && before
            .last_checked_at
            .is_some_and(|last| now_epoch().saturating_sub(last) < CHECK_CACHE_SECONDS)
    {
        return Ok(before);
    }

    manager.begin()?;
    {
        let mut runtime = manager.inner.lock().unwrap();
        runtime.persisted.view.lifecycle = "checking".into();
        runtime.persisted.view.error = None;
        runtime.persisted.view.error_code = None;
    }
    manager.emit(&app);

    let result = async {
        let endpoint = UPDATE_ENDPOINT
            .parse()
            .map_err(|err| format!("invalid update endpoint: {err}"))?;
        let updater = app
            .updater_builder()
            .target(UPDATE_TARGET)
            .endpoints(vec![endpoint])
            .map_err(|err| err.to_string())?
            .pubkey(updater_public_key())
            .timeout(CHECK_TIMEOUT)
            .configure_client(|client| client.timeout(DOWNLOAD_TIMEOUT))
            .build()
            .map_err(|err| err.to_string())?;
        updater.check().await.map_err(|err| err.to_string())
    }
    .await;

    match result {
        Ok(Some(update)) => {
            let mut runtime = manager.inner.lock().unwrap();
            let keep_ready = runtime.persisted.view.available_version.as_deref()
                == Some(update.version.as_str())
                && verified_cache_exists(&runtime.persisted);
            runtime.persisted.view.last_checked_at = Some(now_epoch());
            runtime.persisted.view.available_version = Some(update.version.clone());
            runtime.persisted.view.notes = update.body.clone();
            runtime.persisted.view.retryable = false;
            runtime.persisted.view.error = None;
            runtime.persisted.view.error_code = None;
            runtime.persisted.signature = Some(update.signature.clone());
            runtime.persisted.download_url = Some(update.download_url.to_string());
            runtime.pending = Some(update);
            if !keep_ready {
                discard_candidate_files(&runtime.persisted);
                runtime.persisted.view.lifecycle = "available".into();
                runtime.persisted.view.downloaded_bytes = 0;
                runtime.persisted.view.total_bytes = None;
                runtime.persisted.view.downloaded_at = None;
                runtime.persisted.cache_path = None;
                runtime.persisted.sha256 = None;
            }
        }
        Ok(None) => {
            let mut runtime = manager.inner.lock().unwrap();
            runtime.persisted.view.last_checked_at = Some(now_epoch());
            runtime.pending = None;
            if verified_cache_exists(&runtime.persisted) {
                runtime.persisted.view.lifecycle = "ready".into();
            } else {
                discard_candidate_files(&runtime.persisted);
                clear_candidate(&mut runtime.persisted, "idle");
            }
        }
        Err(err) => {
            let mut runtime = manager.inner.lock().unwrap();
            let ready = runtime.persisted.view.available_version.is_some()
                && verified_cache_exists(&runtime.persisted);
            runtime.persisted.view.lifecycle = if ready { "ready" } else { "error" }.into();
            runtime.persisted.view.retryable = true;
            runtime.persisted.view.error_code = Some("update_check_failed".into());
            runtime.persisted.view.error = Some(format!(
                "Rove could not check for updates. This never blocks game launch. ({err})"
            ));
        }
    }

    manager.persist();
    manager.finish();
    manager.emit(&app);
    Ok(manager.snapshot())
}

#[tauri::command]
pub async fn download_update(
    app: AppHandle,
    manager: State<'_, UpdateManager>,
) -> Result<UpdateState, String> {
    if manager.snapshot().development {
        return Err("Update downloads are disabled in development builds.".into());
    }
    manager.begin()?;

    let update = manager.inner.lock().unwrap().pending.clone();
    let Some(update) = update else {
        manager.finish();
        return Err("Check for updates again before downloading.".into());
    };

    {
        let mut runtime = manager.inner.lock().unwrap();
        runtime.persisted.view.lifecycle = "downloading".into();
        runtime.persisted.view.downloaded_bytes = 0;
        runtime.persisted.view.total_bytes = None;
        runtime.persisted.view.error = None;
        runtime.persisted.view.error_code = None;
    }
    manager.emit(&app);

    let version = sanitize_version(&update.version);
    let part_path = manager
        .cache_dir
        .join(format!("cloud-saver-{version}.updater.part"));
    let final_path = manager
        .cache_dir
        .join(format!("cloud-saver-{version}.updater"));
    let mut final_result = None;
    for attempt in 0..3_u64 {
        {
            let mut runtime = manager.inner.lock().unwrap();
            runtime.persisted.view.downloaded_bytes = 0;
            runtime.persisted.view.total_bytes = None;
        }
        let result = download_to_staging_file(&update, &part_path, |downloaded, total| {
            let mut runtime = manager.inner.lock().unwrap();
            runtime.persisted.view.downloaded_bytes = downloaded;
            runtime.persisted.view.total_bytes = total;
            drop(runtime);
            manager.emit(&app);
        })
        .await;
        if result.is_ok() || attempt == 2 {
            final_result = Some(result);
            break;
        }
        tokio::time::sleep(Duration::from_secs(1 << attempt)).await;
    }
    let bytes = final_result.expect("bounded download attempts always produce a result");

    match bytes {
        Ok(download) => {
            let result = complete_staged_download(&part_path, &final_path);
            match result {
                Ok(()) => {
                    let mut runtime = manager.inner.lock().unwrap();
                    if let Some(old) = runtime.persisted.cache_path.as_ref() {
                        if old != &final_path {
                            let _ = fs::remove_file(old);
                        }
                    }
                    runtime.persisted.view.lifecycle = "ready".into();
                    runtime.persisted.view.downloaded_bytes = download.size;
                    runtime.persisted.view.total_bytes = Some(download.size);
                    runtime.persisted.view.downloaded_at = Some(now_epoch());
                    runtime.persisted.view.retryable = false;
                    runtime.persisted.cache_path = Some(final_path);
                    runtime.persisted.sha256 = Some(download.sha256);
                }
                Err(err) => set_download_error(&manager, "update_cache_write_failed", err),
            }
        }
        Err(err) => set_download_error(&manager, "update_download_failed", err.to_string()),
    }

    manager.persist();
    manager.finish();
    manager.emit(&app);
    Ok(manager.snapshot())
}

#[tauri::command]
pub fn install_downloaded_update(
    app: AppHandle,
    manager: State<'_, UpdateManager>,
) -> Result<UpdateState, String> {
    if manager.snapshot().development {
        return Err("Update installation is disabled in development builds.".into());
    }
    manager.begin()?;
    let install_result = install_ready_candidate(&app, &manager);
    if let Err(err) = install_result {
        let _ = ensure_backend_running(&app);
        let mut runtime = manager.inner.lock().unwrap();
        runtime.persisted.view.lifecycle = "ready".into();
        runtime.persisted.view.retryable = true;
        runtime.persisted.view.error_code = Some("update_install_failed".into());
        runtime.persisted.view.error = Some(err.clone());
        drop(runtime);
        manager.persist();
        manager.finish();
        manager.emit(&app);
        return Err(err);
    }
    manager.finish();
    Ok(manager.snapshot())
}

#[tauri::command]
pub fn discard_downloaded_update(
    app: AppHandle,
    manager: State<'_, UpdateManager>,
) -> Result<UpdateState, String> {
    manager.begin()?;
    {
        let mut runtime = manager.inner.lock().unwrap();
        discard_candidate_files(&runtime.persisted);
        clear_candidate(&mut runtime.persisted, "idle");
        runtime.pending = None;
    }
    manager.persist();
    manager.finish();
    manager.emit(&app);
    Ok(manager.snapshot())
}

fn install_ready_candidate(app: &AppHandle, manager: &UpdateManager) -> Result<(), String> {
    let candidate = manager.inner.lock().unwrap().persisted.clone();
    if candidate.view.lifecycle != "ready" {
        return Err("No verified update is ready to install.".into());
    }
    let path = candidate
        .cache_path
        .as_deref()
        .ok_or_else(|| "The downloaded update path is missing.".to_string())?;
    let actual_hash =
        sha256_file(path).map_err(|err| format!("Could not read the downloaded update: {err}"))?;
    if candidate.sha256.as_deref() != Some(actual_hash.as_str()) {
        return Err("The downloaded update failed its checksum recheck.".into());
    }
    let file = fs::File::open(path).map_err(|err| err.to_string())?;
    // Reverify directly from a read-only mapping so installation does not
    // allocate an updater-sized buffer after a restart.
    let mapped = unsafe { memmap2::MmapOptions::new().map(&file) }
        .map_err(|err| format!("Could not map the downloaded update: {err}"))?;
    verify_signature(
        &mapped,
        candidate
            .signature
            .as_deref()
            .ok_or_else(|| "The downloaded update signature is missing.".to_string())?,
        updater_public_key(),
    )?;

    {
        let mut runtime = manager.inner.lock().unwrap();
        runtime.persisted.view.lifecycle = "installing".into();
        runtime.persisted.view.error = None;
        runtime.persisted.view.error_code = None;
    }
    manager.persist();
    manager.emit(app);

    drop(mapped);
    drop(file);
    let installer_path = materialize_windows_installer(path)?;
    request_daemon_shutdown()?;
    let started = SystemTime::now();
    while backend_running() {
        if started.elapsed().unwrap_or_default() >= DAEMON_SHUTDOWN_TIMEOUT {
            return Err("The background service did not finish shutting down. Try again after protection work completes.".into());
        }
        std::thread::sleep(Duration::from_millis(150));
    }

    Command::new(installer_path)
        .args(["/P", "/R", "/UPDATE"])
        .spawn()
        .map_err(|err| format!("Could not start the verified installer: {err}"))?;
    app.exit(0);
    Ok(())
}

struct DownloadedFile {
    size: u64,
    sha256: String,
}

async fn download_to_staging_file(
    update: &Update,
    part_path: &Path,
    mut on_progress: impl FnMut(u64, Option<u64>),
) -> Result<DownloadedFile, String> {
    if let Some(parent) = part_path.parent() {
        fs::create_dir_all(parent).map_err(|err| err.to_string())?;
    }
    let _ = fs::remove_file(part_path);
    let client = reqwest::Client::builder()
        .connect_timeout(Duration::from_secs(10))
        .timeout(DOWNLOAD_TIMEOUT)
        .build()
        .map_err(|err| err.to_string())?;
    let response = client
        .get(update.download_url.clone())
        .headers(update.headers.clone())
        .send()
        .await
        .map_err(|err| err.to_string())?;
    if !response.status().is_success() {
        return Err(format!(
            "Update download returned HTTP {}.",
            response.status()
        ));
    }
    let total = response.content_length();
    let mut stream = response.bytes_stream();
    let mut file = fs::File::create(part_path).map_err(|err| err.to_string())?;
    let mut digest = Sha256::new();
    let mut downloaded = 0_u64;
    loop {
        let next = tokio::time::timeout(DOWNLOAD_NO_PROGRESS_TIMEOUT, stream.next())
            .await
            .map_err(|_| "The update download made no progress for 45 seconds.".to_string())?;
        let Some(chunk) = next else { break };
        let chunk = chunk.map_err(|err| err.to_string())?;
        file.write_all(&chunk).map_err(|err| err.to_string())?;
        digest.update(&chunk);
        downloaded = downloaded.saturating_add(chunk.len() as u64);
        on_progress(downloaded, total);
    }
    file.sync_all().map_err(|err| err.to_string())?;
    drop(file);
    if downloaded == 0 {
        return Err("The update download was empty.".into());
    }
    let file = fs::File::open(part_path).map_err(|err| err.to_string())?;
    // The file remains open and immutable while the read-only mapping is
    // verified. Mapping avoids allocating a second installer-sized buffer.
    let mapped = unsafe { memmap2::MmapOptions::new().map(&file) }
        .map_err(|err| format!("Could not map the downloaded update for verification: {err}"))?;
    verify_signature(&mapped, &update.signature, updater_public_key())?;
    Ok(DownloadedFile {
        size: downloaded,
        sha256: format!("{:x}", digest.finalize()),
    })
}

fn complete_staged_download(part_path: &Path, final_path: &Path) -> Result<(), String> {
    if final_path.exists() {
        fs::remove_file(final_path).map_err(|err| err.to_string())?;
    }
    fs::rename(part_path, final_path).map_err(|err| err.to_string())
}

fn materialize_windows_installer(cache_path: &Path) -> Result<PathBuf, String> {
    let mut source = fs::File::open(cache_path).map_err(|err| err.to_string())?;
    let mut magic = [0_u8; 2];
    source
        .read_exact(&mut magic)
        .map_err(|err| err.to_string())?;
    if magic == *b"MZ" {
        return Ok(cache_path.to_path_buf());
    }
    if magic != *b"PK" {
        return Err("The downloaded update is not a valid Windows updater archive.".into());
    }
    drop(source);
    let source = fs::File::open(cache_path).map_err(|err| err.to_string())?;
    let mut archive = zip::ZipArchive::new(source)
        .map_err(|err| format!("Could not open the updater archive: {err}"))?;
    let mut selected = None;
    for index in 0..archive.len() {
        let entry = archive
            .by_index(index)
            .map_err(|err| format!("Could not inspect the updater archive: {err}"))?;
        let safe_name = entry.enclosed_name().map(PathBuf::from);
        if safe_name
            .as_deref()
            .and_then(Path::extension)
            .and_then(|value| value.to_str())
            .is_some_and(|value| value.eq_ignore_ascii_case("exe"))
        {
            selected = Some(index);
            break;
        }
    }
    let index = selected.ok_or_else(|| "The updater archive contains no installer.".to_string())?;
    let mut entry = archive
        .by_index(index)
        .map_err(|err| format!("Could not read the updater archive: {err}"))?;
    if entry.size() > 512 * 1024 * 1024 {
        return Err("The updater installer is unexpectedly large.".into());
    }
    let parent = cache_path
        .parent()
        .ok_or_else(|| "The updater cache path is invalid.".to_string())?;
    let final_path = parent.join("cloud-saver-update-installer.exe");
    let part_path = parent.join("cloud-saver-update-installer.exe.part");
    let _ = fs::remove_file(&part_path);
    let mut output = fs::File::create(&part_path)
        .map_err(|err| format!("Could not stage the updater installer: {err}"))?;
    std::io::copy(&mut entry, &mut output)
        .map_err(|err| format!("Could not extract the updater installer: {err}"))?;
    output.sync_all().map_err(|err| err.to_string())?;
    drop(output);
    let mut header = [0_u8; 2];
    fs::File::open(&part_path)
        .and_then(|mut file| file.read_exact(&mut header))
        .map_err(|err| err.to_string())?;
    if header != *b"MZ" {
        let _ = fs::remove_file(&part_path);
        return Err("The updater archive did not contain a valid Windows installer.".into());
    }
    if final_path.exists() {
        fs::remove_file(&final_path).map_err(|err| err.to_string())?;
    }
    fs::rename(&part_path, &final_path).map_err(|err| err.to_string())?;
    Ok(final_path)
}

fn request_daemon_shutdown() -> Result<(), String> {
    let addr: SocketAddr = BACKEND_API_ADDR
        .parse()
        .map_err(|_| "The background service address is invalid.".to_string())?;
    let mut stream = TcpStream::connect_timeout(&addr, Duration::from_secs(2))
        .map_err(|_| "The background service is not reachable.".to_string())?;
    stream
        .set_read_timeout(Some(Duration::from_secs(3)))
        .map_err(|err| err.to_string())?;
    stream
        .set_write_timeout(Some(Duration::from_secs(3)))
        .map_err(|err| err.to_string())?;
    let request = concat!(
        "POST /lifecycle/prepare-update HTTP/1.1\r\n",
        "Host: 127.0.0.1:7331\r\n",
        "Content-Length: 0\r\n",
        "Connection: close\r\n\r\n"
    );
    stream
        .write_all(request.as_bytes())
        .map_err(|err| format!("Could not prepare the background service: {err}"))?;
    let mut response = String::new();
    stream
        .read_to_string(&mut response)
        .map_err(|err| format!("Could not read the background-service response: {err}"))?;
    let status = response.lines().next().unwrap_or_default();
    if status.contains(" 202 ") {
        return Ok(());
    }
    let detail = response
        .split("\r\n\r\n")
        .nth(1)
        .unwrap_or("The background service refused the update.");
    Err(detail.to_string())
}

fn set_download_error(manager: &UpdateManager, code: &str, error: String) {
    let mut runtime = manager.inner.lock().unwrap();
    runtime.persisted.view.lifecycle = "error".into();
    runtime.persisted.view.retryable = true;
    runtime.persisted.view.error_code = Some(code.into());
    runtime.persisted.view.error = Some(error);
}

fn updater_public_key() -> &'static str {
    include_str!("../updater-public-key.txt").trim()
}

fn verify_signature(data: &[u8], release_signature: &str, public_key: &str) -> Result<(), String> {
    if public_key.trim().is_empty() {
        return Err("This build has no updater public key.".into());
    }
    let public_key = decode_base64_string(public_key)?;
    let signature = decode_base64_string(release_signature)?;
    let public_key = PublicKey::decode(&public_key)
        .map_err(|err| format!("The updater public key is invalid: {err}"))?;
    let signature = Signature::decode(&signature)
        .map_err(|err| format!("The update signature is invalid: {err}"))?;
    public_key
        .verify(data, &signature, true)
        .map_err(|err| format!("The downloaded update signature did not verify: {err}"))
}

fn decode_base64_string(value: &str) -> Result<String, String> {
    let bytes = base64::engine::general_purpose::STANDARD
        .decode(value)
        .map_err(|err| format!("invalid base64 signing value: {err}"))?;
    String::from_utf8(bytes).map_err(|_| "A signing value was not valid UTF-8.".into())
}

fn verified_cache_exists(state: &PersistedUpdateState) -> bool {
    let Some(path) = state.cache_path.as_deref() else {
        return false;
    };
    let Some(expected) = state.sha256.as_deref() else {
        return false;
    };
    sha256_file(path).is_ok_and(|actual| actual == expected)
}

fn sha256_file(path: &Path) -> std::io::Result<String> {
    let mut file = fs::File::open(path)?;
    let mut digest = Sha256::new();
    let mut buffer = [0_u8; 64 * 1024];
    loop {
        let read = file.read(&mut buffer)?;
        if read == 0 {
            break;
        }
        digest.update(&buffer[..read]);
    }
    Ok(format!("{:x}", digest.finalize()))
}

fn candidate_is_not_newer(candidate: Option<&str>, current: &str) -> bool {
    let Some(candidate) = candidate else {
        return false;
    };
    match (
        Version::parse(candidate.trim_start_matches('v')),
        Version::parse(current),
    ) {
        (Ok(candidate), Ok(current)) => candidate <= current,
        _ => candidate == current,
    }
}

fn clear_candidate(state: &mut PersistedUpdateState, lifecycle: &str) {
    state.view.lifecycle = lifecycle.into();
    state.view.available_version = None;
    state.view.notes = None;
    state.view.downloaded_bytes = 0;
    state.view.total_bytes = None;
    state.view.downloaded_at = None;
    state.view.retryable = false;
    state.view.error_code = None;
    state.view.error = None;
    state.signature = None;
    state.download_url = None;
    state.cache_path = None;
    state.sha256 = None;
}

fn discard_candidate_files(state: &PersistedUpdateState) {
    if let Some(path) = state.cache_path.as_deref() {
        let _ = fs::remove_file(path);
    }
}

fn load_persisted_state(path: &Path) -> Option<PersistedUpdateState> {
    let bytes = fs::read(path)
        .or_else(|_| fs::read(path.with_extension("json.bak")))
        .ok()?;
    let state: PersistedUpdateState = serde_json::from_slice(&bytes).ok()?;
    (state.schema_version == UPDATE_STATE_SCHEMA).then_some(state)
}

fn persist_state(path: &Path, state: &PersistedUpdateState) -> Result<(), String> {
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent).map_err(|err| err.to_string())?;
    }
    let temp_path = path.with_extension("json.tmp");
    let backup_path = path.with_extension("json.bak");
    let bytes = serde_json::to_vec_pretty(state).map_err(|err| err.to_string())?;
    {
        let mut file = fs::File::create(&temp_path).map_err(|err| err.to_string())?;
        file.write_all(&bytes).map_err(|err| err.to_string())?;
        file.sync_all().map_err(|err| err.to_string())?;
    }
    if path.exists() {
        let _ = fs::copy(path, &backup_path);
        fs::remove_file(path).map_err(|err| err.to_string())?;
    }
    fs::rename(&temp_path, path).map_err(|err| err.to_string())
}

fn sanitize_version(version: &str) -> String {
    version
        .chars()
        .filter(|ch| ch.is_ascii_alphanumeric() || matches!(ch, '.' | '-' | '_'))
        .collect()
}

fn now_epoch() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_secs()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn version_comparison_rejects_installed_and_older_candidates() {
        assert!(candidate_is_not_newer(Some("1.2.3"), "1.2.3"));
        assert!(candidate_is_not_newer(Some("1.2.2"), "1.2.3"));
        assert!(!candidate_is_not_newer(Some("1.3.0"), "1.2.3"));
    }

    #[test]
    fn version_is_safe_for_cache_filename() {
        assert_eq!(sanitize_version("v1.2.3/../../bad"), "v1.2.3....bad");
    }

    #[test]
    fn updater_uses_tauri_windows_platform_key() {
        assert_eq!(UPDATE_TARGET, "windows-x86_64");
    }

    #[test]
    fn persisted_state_round_trips() {
        let root = std::env::temp_dir().join(format!("cloud-saver-updater-test-{}", now_epoch()));
        let path = root.join("state.json");
        let state = PersistedUpdateState {
            schema_version: UPDATE_STATE_SCHEMA,
            view: UpdateState::initial("1.0.0".into()),
            signature: Some("signature".into()),
            download_url: Some("https://example.invalid/update.exe".into()),
            cache_path: None,
            sha256: None,
        };
        persist_state(&path, &state).unwrap();
        let loaded = load_persisted_state(&path).unwrap();
        assert_eq!(loaded.schema_version, UPDATE_STATE_SCHEMA);
        assert_eq!(loaded.view.current_version, "1.0.0");
        let _ = fs::remove_dir_all(root);
    }

    #[test]
    fn release_signature_matches_committed_key_when_requested() {
        let Ok(artifact_path) = std::env::var("CLOUD_SAVER_VERIFY_UPDATE_ARTIFACT") else {
            return;
        };
        let signature_path = std::env::var("CLOUD_SAVER_VERIFY_UPDATE_SIGNATURE")
            .expect("signature path must accompany updater artifact path");
        let artifact = fs::read(artifact_path).unwrap();
        let signature = fs::read_to_string(signature_path).unwrap();
        verify_signature(&artifact, signature.trim(), updater_public_key()).unwrap();
    }
}
