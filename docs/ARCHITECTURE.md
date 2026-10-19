# Architecture

TermXP is one desktop application. Wails embeds the built Vue bundle and binds three small services: `App` (typed metrics and window-close confirmation), `TerminalService` (terminal operations), and `ConfigService` (validated configuration). Context setup and session shutdown are unexported and unavailable through frontend bindings.

## Session ownership

The Go session registry serializes spawn/registration with shutdown and duplicate detection. It allows up to 64 sessions. Each session owns its PTY, process waiter, cancellation signal, completion signal, acknowledgement channel, and optional log file. `cmd.Wait` runs exactly once. Completion closes the PTY/log and removes only the matching registry entry. Shutdown cancels all sessions and joins for up to three seconds. The PTY master is rewrapped as a nonblocking, pollable descriptor; resize uses SyscallConn so it cannot silently revert I/O to blocking. Cancellation closes I/O before signalling the process group, escalating from TERM to KILL after 200 ms. PTY hangup also affects foreground jobs; deliberately detached processes that create new sessions are outside this cleanup contract.

The frontend registry is keyed by stable pane ID. A `SessionController` owns subscriptions, input serialization, process startup, exit/error state, and disposal. A terminal view owns xterm, its addons, and the reusable DOM wrapper. Vue components attach/detach that wrapper and own only view observers/listeners. Moving a pane preserves the controller and terminal history. Closing a pane disposes its controller; late startup completion triggers another kill so it cannot leave an orphan. Reconnect uses a fresh backend session ID and keeps scrollback.

## Output and input

PTY reads use a fixed 32 KiB buffer. Each `terminal:data:<id>` event carries `{sequence, data}`, where `data` is base64 bytes. xterm receives `Uint8Array`, preserving UTF-8 sequences split across packets. Only one packet is outstanding per session. `AcknowledgeOutput(id, sequence)` is called from xterm's consumption callback; until then Go stops reading. This bounds application buffering and pushes pressure back to the process. Cancellation releases an unacknowledged packet. Final output is acknowledged before `terminal:exit:<id>` reports the process exit code.

Input is serialized per controller with a bounded queue. Custom clipboard paste goes through xterm's paste API, including bracketed-paste handling. Broadcast routes the resulting input to running controllers in the selected tab, and turns off when tab selection or membership changes. Snippets deliberately execute in the focused pane only.

`TerminalService` exposes start/start-with-options, write, resize, kill, acknowledge, list-shells, and logging. Dimensions and payload/session counts are limited in Go. Shells must be absolute executable paths. SSH launches an argument array through the installed client; no command string or shell interpolation is used.

## Workspace and configuration

Layout nodes form a discriminated union: terminal leaves carry stable pane identity and launch descriptors; split nodes carry orientation, children, and percentages. Pure operations preserve terminal identity and collapse empty/single-child splits. Workspace state, themes, controllers, persistence, and action definitions live in separate frontend modules.

Config version 1 stores preferences, tab descriptors, and SSH connection metadata. It excludes runtime IDs, buffers, metrics, broadcast state, log enablement, and credentials. Saves use a same-directory temporary file, fsync, rename, and directory fsync. New directories use 0700 and files 0600. Invalid/unsupported config falls back to defaults with a visible warning. Workspace descriptors are restored with new pane and session IDs. A debounced, serialized save handles edits; window close requests a final save before confirming Wails shutdown.

Linux working directories are polled from `/proc/<shell-pid>/cwd` every two seconds. SSH labels describe the local SSH process directory, not the remote shell's directory. CPU is normalized so one fully utilized core is approximately 100%; RAM covers only the Go process, excluding webview and child shells. Unavailable metrics use null values.

## Trust boundary and builds

The Wails bridge has the user's shell privileges. Shell output is rendered through xterm, never HTML. Production CSP disallows external scripts, frames, objects, and form submissions. Inline styles remain allowed for Vue/xterm positioning. Development serves on loopback with Vite CORS disabled; Wails 2.17.0 checks IPC host/origin and authorization. Generated bindings are regenerated with the pinned CLI. Frontend and Go output-event changes must ship together.
