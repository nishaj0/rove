# Rove brand implementation

The supplied Rove brand kit is the source of truth for the logo and icon. The desktop app and website use a dark-first desktop theme built around that orange: cool blue-slate neutrals, one orange accent for primary actions and the selected item, clear borders, compact controls, keyboard-visible focus, and explicit status text. A light theme is available.

## Tagline

The approved product tagline is: **Game cloud saves for everyone.** Use it exactly in the app identity area, README, product listings, and release-facing copy. Do not add a second competing tagline.

## Approved desktop assets

Only the assets needed by the Windows desktop app are stored in the repository:

- `app/src/assets/rove-primary-horizontal.svg` for light UI surfaces.
- `app/src/assets/rove-reversed-horizontal.svg` for dark UI surfaces.
- `app/src-tauri/icons/icon.ico` for the Windows executable and installer.
- `app/src-tauri/resources/backend/icon.ico` for the bundled background service.
- `app/src-tauri/icons/rove-app-icon-light-1024px.png` as the retained source used to generate the Windows icon.

The original kit is not copied wholesale. Its licence note permits use and modification for Rove. Font files are intentionally not bundled.

## Tokens and typography

Use these values for new product UI. Keep the semantic CSS variables in `app/src/styles.css` rather than hard-coding colours in components.

| Role | Light | Dark |
| --- | --- | --- |
| Paper | `#F8F3EB` | `#1C140E` |
| Surface | `#FDFAF4` | `#281E17` |
| Ink | `#291C14` | `#EEE7DB` |
| Border | `#C4B4A3` | `#5C4C3F` |
| Brand orange | `#DF6100` | `#F68622` |
| Strong orange | `#AB3E00` | `#F4B768` |

- Display headings: `Trebuchet MS`, `Arial Rounded MT Bold`, Arial, sans-serif.
- Product and body UI: `Segoe UI`, Tahoma, Arial, sans-serif.
- Keep corners at 4px or 8px, borders visible, and shadows as a restrained 3px offset.
- Use the primary logo on paper/surface backgrounds and the reversed logo only on dark surfaces.
- Do not add gradients, glow, bevels, oversized pill shapes, or decorative status dots without text.

## Rebrand compatibility

The visible application name, window title, installer name, tray text, and product documentation are **Rove**. The internal Tauri identifier (`com.cloudsaver.app`), Go module path, `CloudSaver` application-data directory, provider layout, and background daemon filename intentionally remain unchanged in this release. They are persisted compatibility identifiers: changing them would split existing saves, credentials, autostart registration, and update history into a second installation. The daemon also keeps a legacy `Cloud Saver.exe` lookup only as an update-transition fallback when opening an already installed desktop app.

Do not rename those internal identifiers as part of a visual change. A future identity migration, if ever required, must copy data atomically and preserve the existing update path.

## Product terminology

Use **Cloud Sync** for the player-facing feature name and **sync** for its actions: *Cloud Sync health*, *Sync now*, *Pause all sync*, and *last local sync*. The compatibility API, persisted settings, and database still use `protection` identifiers such as `protectionTarget` and `ProtectionTarget`; these are implementation details and must not leak into new UI copy.

## Asset changes

Use the Tauri icon generator only when replacing the 1024px source icon:

```powershell
cd app
node node_modules/@tauri-apps/cli/tauri.js icon src-tauri/icons/rove-app-icon-light-1024px.png
```

Keep only the regenerated `src-tauri/icons/icon.ico` for this Windows-focused app, then copy that same file to `src-tauri/resources/backend/icon.ico`. Do not commit generated Android, iOS, Appx, or macOS icon variants unless the project begins shipping those platforms.

## Desktop UI theme

Tokens live in `app/src/styles.css` (app) and `rove-website/src/styles/tokens.css` (website) and use OKLCH.

| Role | Dark | Light |
| --- | --- | --- |
| Content surface | `oklch(0.185 0.014 265)` | `oklch(0.985 0.003 265)` |
| Chrome (toolbar, sidebar, status bar) | `oklch(0.145 0.014 265)` | `oklch(0.935 0.007 265)` |
| Raised surface | `oklch(0.225 0.016 265)` | `oklch(1 0 0)` |
| Text | `oklch(0.95 0.008 265)` | `oklch(0.22 0.02 265)` |
| Accent | `oklch(0.72 0.17 48)` | `oklch(0.55 0.18 45)` |

Rules: the accent is used only for primary actions, the selected item and state indicators; status uses green, amber and red together with text; the UI font is the system UI font (Segoe UI Variable on Windows). The logo has no baked-in background, so it sits on either theme.
