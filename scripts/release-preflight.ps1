[CmdletBinding()]
param(
  [Parameter(Mandatory = $true)]
  [ValidatePattern("^v\d+\.\d+\.\d+$")]
  [string]$Version,

  [switch]$Bundle,

  [string]$Repository = "nishaj0/rove"
)

$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
$previousGoCache = $env:GOCACHE
$previousCargoTargetDir = $env:CARGO_TARGET_DIR

function Invoke-Checked {
  param(
    [string]$Label,
    [scriptblock]$Command
  )

  Write-Host "==> $Label"
  & $Command
  if ($LASTEXITCODE -ne 0) {
    throw "$Label failed with exit code $LASTEXITCODE"
  }
}

Push-Location $repoRoot
try {
  $nodeVersion = (& node --version).Trim()
  if ($LASTEXITCODE -ne 0) {
    throw "Unable to determine the active Node.js version"
  }
  $nodeMajor = [int](($nodeVersion -replace "^v", "").Split(".")[0])
  if ($nodeMajor -lt 24) {
    throw "Release CI uses Node.js 24; found $nodeVersion. Select Node.js 24 before running the preflight."
  }

  Invoke-Checked "validate release version targets" { node scripts/sync-version.mjs $Version --dry-run }

  Push-Location app
  try {
    Invoke-Checked "type-check frontend" { npm run type-check }
    Invoke-Checked "build frontend" { npm run build }
  }
  finally {
    Pop-Location
  }

  if ([string]::IsNullOrWhiteSpace($env:GOCACHE)) {
    $env:GOCACHE = Join-Path $repoRoot ".cache\go-build"
  }
  Push-Location backend
  try {
    Invoke-Checked "build backend" { go build ./... }
  }
  finally {
    Pop-Location
  }

  Push-Location app\src-tauri
  try {
    $env:CARGO_TARGET_DIR = Join-Path $repoRoot ".cache\release-preflight\cargo-target"
    Invoke-Checked "check native shell" { cargo check }
  }
  finally {
    Pop-Location
  }

  if ([string]::IsNullOrWhiteSpace($previousCargoTargetDir)) {
    Remove-Item Env:CARGO_TARGET_DIR -ErrorAction SilentlyContinue
  }
  else {
    $env:CARGO_TARGET_DIR = $previousCargoTargetDir
  }

  if (-not $Bundle) {
    Write-Host "Release preflight passed. Use -Bundle with local release credentials to build and verify a signed installer."
    return
  }

  Invoke-Checked "confirm release version metadata" { node scripts/sync-version.mjs $Version --check }

  foreach ($name in "CLOUD_SAVER_GOOGLE_CLIENT_ID", "CLOUD_SAVER_GOOGLE_CLIENT_SECRET", "TAURI_SIGNING_PRIVATE_KEY", "TAURI_SIGNING_PRIVATE_KEY_PASSWORD") {
    if ([string]::IsNullOrWhiteSpace([Environment]::GetEnvironmentVariable($name))) {
      throw "$name is required when -Bundle is specified"
    }
  }

  $publicKey = Get-Content -Raw app/src-tauri/updater-public-key.txt
  $configuredKey = (Get-Content -Raw app/src-tauri/tauri.conf.json | ConvertFrom-Json).plugins.updater.pubkey
  if ([string]::IsNullOrWhiteSpace($publicKey) -or $publicKey.Trim() -ne $configuredKey.Trim()) {
    throw "The committed updater public key does not match app/src-tauri/tauri.conf.json"
  }

  Push-Location app
  try {
    Invoke-Checked "build signed Tauri installer" { npm run tauri:build }
  }
  finally {
    Pop-Location
  }

  Invoke-Checked "generate updater manifest" {
    node scripts/generate-update-manifest.mjs "app/src-tauri/target/release/bundle/nsis" $Version $Repository
  }
  Invoke-Checked "validate updater manifest" {
    node scripts/generate-update-manifest.mjs "app/src-tauri/target/release/bundle/nsis" $Version $Repository --check
  }

  $archive = Get-ChildItem "app/src-tauri/target/release/bundle/nsis/*" -File |
    Where-Object { ($_.Name -like "*.nsis.zip" -or $_.Name -like "*-setup.exe") -and (Test-Path "$($_.FullName).sig") } |
    Select-Object -First 1
  if (-not $archive) {
    throw "No signed NSIS updater payload found"
  }
  $env:CLOUD_SAVER_VERIFY_UPDATE_ARTIFACT = $archive.FullName
  $env:CLOUD_SAVER_VERIFY_UPDATE_SIGNATURE = "$($archive.FullName).sig"
  Invoke-Checked "verify updater signature" {
    cargo test --manifest-path app/src-tauri/Cargo.toml release_signature_matches_committed_key_when_requested
  }

  $bundleRoot = "app/src-tauri/target/release/bundle"
  $files = Get-ChildItem -Path $bundleRoot -File -Recurse |
    Where-Object { $_.Extension -in ".exe", ".msi", ".zip", ".sig" }
  if (-not $files) {
    throw "No release artifacts found under $bundleRoot"
  }
  $checksums = foreach ($file in $files) {
    $hash = (Get-FileHash -Algorithm SHA256 -Path $file.FullName).Hash.ToLower()
    $relativePath = Resolve-Path -Relative -Path $file.FullName
    "$hash  $($relativePath -replace '^\.\\', '' -replace '\\', '/')"
  }
  Set-Content -Path release-checksums.txt -Value $checksums -Encoding ascii
  Write-Host "Signed bundle preflight passed. latest.json and release-checksums.txt were generated locally."
}
finally {
  $env:GOCACHE = $previousGoCache
  if ([string]::IsNullOrWhiteSpace($previousCargoTargetDir)) {
    Remove-Item Env:CARGO_TARGET_DIR -ErrorAction SilentlyContinue
  }
  else {
    $env:CARGO_TARGET_DIR = $previousCargoTargetDir
  }
  Pop-Location
}
