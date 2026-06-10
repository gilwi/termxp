# TermXP - Premium Terminal Application

TermXP is a modern, visually rich terminal emulator built using the [Wails](https://wails.io/) framework. It combines a Go-powered backend for low-level system interactions (like PTY management) with a Vue 3 and TypeScript frontend for a highly customizable and interactive user interface.

## Core Architecture

### Backend (Go)
The Go backend is responsible for process management, system monitoring, and bridging terminal data between the OS and the frontend.

- **`main.go`**: The entry point of the application. It configures the Wails application, sets up the asset server, and binds the `App` and `TerminalService` structures to the frontend.
- **`app.go`**: Implements the `App` service, which provides real-time system metrics (CPU usage, RAM usage, and uptime). It specifically reads from `/proc/` files, making it Linux-oriented.
- **`terminal.go`**: Implements the `TerminalService`, which manages terminal sessions. It uses `github.com/creack/pty` to spawn and control shell processes (like `bash` or `sh`) in a pseudo-terminal (PTY). It uses Wails events (`terminal:data:<id>`, `terminal:exit:<id>`) to stream output to the frontend.

### Frontend (Vue 3 / TypeScript)
The frontend provides the terminal UI, session management (tabs/panes), and theme engine.

- **`frontend/src/App.vue`**: The main application component. It manages tabs, themes, system metrics polling, and the layout tree for terminal panes.
- **`frontend/src/components/TerminalInstance.vue`**: Wraps the `@xterm/xterm` library. It initializes the terminal, handles input/output streaming via Wails events, and manages resizing through the `FitAddon` and `ResizeObserver`.
- **`frontend/src/utils/layout.ts`**: Contains the logic for managing the recursive split-pane layout system.
- **Themes**: Supports multiple visual themes (Glassmorphic, Cyberpunk, Dracula, Matrix, Monokai) defined in `App.vue`.

## Tech Stack

- **Backend**:
    - Go 1.23+
    - Wails v2 (Application Framework)
    - `creack/pty` (PTY Support)
    - `google/uuid` (Session ID generation)
- **Frontend**:
    - Vue 3 (Composition API)
    - TypeScript
    - Vite (Build Tool)
    - Xterm.js (Terminal Rendering)
    - Tailwind-like CSS variables and glassmorphism.

## Building and Running

### Prerequisites
- [Go](https://go.dev/doc/install)
- [Node.js & npm](https://nodejs.org/en/download/)
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

### Development
To run the application in development mode with hot-reloading:
```bash
wails dev
```

### Production Build
To build a redistributable production package:
```bash
wails build
```

### Frontend Dependencies
If you need to manually install or update frontend dependencies:
```bash
cd frontend
npm install
```

## Development Conventions

- **Wails Bindings**: All methods in `App` and `TerminalService` that start with an uppercase letter are automatically bound to the frontend and available in `frontend/wailsjs/go/main/`.
- **Event Streaming**: Use `runtime.EventsEmit` in Go and `EventsOn` in TypeScript for high-frequency data like terminal output.
- **System Metrics**: Polling is handled by the frontend in `App.vue` using a `setInterval` that calls `GetSystemStats` every 2500ms.
- **PTY Cleanup**: Ensure `CleanupAllSessions` is called on application shutdown (handled in `main.go`) to prevent orphaned shell processes.
- **Styling**: Uses CSS variables for theming. Themes are applied by switching a class on the `.app-container` in `App.vue`.
