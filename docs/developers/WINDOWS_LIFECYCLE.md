# Windows startup, daemon, and tray

## Startup

Tauri initializes `tauri-plugin-autostart` and registers `Cloud Saver.exe --background`. On a background invocation, the bootstrap ensures the Go daemon is running, keeps the main window hidden, and exits. Settings queries native autostart state so a change made in Windows Startup Apps is reflected in the UI.

The daemon acquires a per-user named mutex before service construction. A second daemon exits without creating another tray or worker set.

## Game monitoring

The monitor enumerates every 10 seconds while idle and every two seconds while a tracked game is launching/running. It compares executable basenames before resolving full paths. Running-state changes update watcher roots, exit scheduling, tray state, and daemon priority.

## Connectivity

The Windows network watcher uses `NotifyIpInterfaceChange`. It is a retry signal, not proof that Google Drive is reachable. Provider responses remain authoritative.

## Tray

Status updates travel over the engine event channel. The tray does not poll every two seconds.

Implemented items:

- open;
- explicit state text;
- provider and pending count;
- protect now;
- Recovery Point creation for running games;
- retry connection;
- review conflicts;
- pause uploads;
- pause all;
- pause the only currently running tracked game;
- resume;
- quit.

Timed/per-game pauses are currently configured in the desktop UI. Pending jobs are durable on quit, but the planned Wait/Exit-and-resume/Cancel confirmation window is not yet implemented.

## Shutdown

The normal application stop cancels contexts, stops monitor/watcher/API components, and closes persistent storage. Jobs, revisions, dirty metadata, provider health, and pause state are already persisted during operation; an interrupted provider transfer resumes idempotently from its revision.
