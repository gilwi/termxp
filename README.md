# TermXP

A Linux-first desktop terminal built with Go, Wails v2, Vue 3, TypeScript, and xterm.js.

## Features

- Tabs with editable names and working-directory labels.
- Nested resizable panes, drag-and-drop moves/swaps, and pane maximization.
- Five themes, font controls, Ctrl+scroll zoom, and optional copy-on-select.
- Scrollback search with Ctrl+F and an action palette with Ctrl+P.
- Shell discovery and a persistent default-shell picker.
- Exited panes retain output and offer Reconnect.
- Automatic settings and workspace persistence; restored panes create fresh processes.
- SSH profiles using the installed OpenSSH client, with key paths and jump hosts.
- Opt-in per-pane logging to a new private file.
- Broadcast typing/paste to running panes in the selected tab, with visible destinations.
- Go-process CPU/RAM metrics and application uptime.

Linux is the supported target. macOS has not been validated; Windows terminals require a future ConPTY adapter.

## Development

Requirements: Go 1.25.1 or newer, Node 22.12 or newer, npm, and Wails CLI **2.17.0**. The dependency scanner uses Go 1.26+ (Go can download that toolchain automatically).

On Ubuntu 24.04:

```sh
sudo apt-get install libgtk-3-dev libwebkit2gtk-4.1-dev
npm --prefix frontend ci
go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0
wails dev -tags webkit2_41
```

Build a native Linux executable:

```sh
wails build -tags webkit2_41
```

Output: `build/bin/termxp`. The CLI and Go module versions must match. Build hooks use `npm ci`; commit the lockfile when updating dependencies.

## Shortcuts

| Shortcut | Action |
|---|---|
| Ctrl+Shift+T / W | New tab / close tab |
| Ctrl+Shift+E / O | Split right / below |
| Ctrl+Shift+X | Toggle maximized pane |
| Ctrl+Shift+C / V | Copy / terminal-aware paste |
| Ctrl+F | Search scrollback |
| Ctrl+P | Command palette |
| Ctrl+scroll | Font size |

Choose the shell for new tabs from the workspace selector. Open the sidebar for SSH profiles and preferences. Each active pane has Search and Log controls; logs may contain sensitive terminal output and are never enabled automatically.

Settings live at `$XDG_CONFIG_HOME/termxp/config.json`, or `~/.config/termxp/config.json`. Saves are atomic, with private files and a private application directory. Workspace restore is enabled by default and can be disabled in preferences. It restores layout, names, shell choices, and directories; terminal output and live sessions are not restored. SSH profiles contain connection metadata and key paths, never private-key contents or passwords.

See [architecture](docs/ARCHITECTURE.md), [validation](docs/TESTING.md), and the [roadmap](ROADMAP.md).
