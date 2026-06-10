# 🚀 TermXP

**TermXP** is a premium, high-performance terminal emulator designed for power users who value both aesthetics and functionality. Built on the cutting-edge **Wails v2** framework, it bridges a high-speed Go backend with a modern, glassmorphic Vue 3 frontend to deliver a terminal experience that feels like the future.

![TermXP Preview](build/appicon.png)

## ✨ Features

- **💎 Stunning Glassmorphism:** A beautiful, semi-transparent UI with customizable blur effects and vibrant accent glows.
- **⚡ Pro Split-Pane Layout:** Deeply nested recursive split-panes. Organize your workspace with horizontal and vertical splits, drag-and-drop repositioning, and fluid resizing.
- **🎨 Dynamic Theming:** Switch instantly between premium themes like *Cyberpunk*, *Dracula*, *Matrix*, and *Monokai*.
- **📊 Real-Time App Metrics:** Monitor TermXP's own footprint with process-specific CPU usage, human-readable RAM consumption (MB/GB), and application uptime.
- **⌨️ Power-User Shortcuts:** 
    - `Ctrl + Shift + T`: Open new tab
    - `Ctrl + Shift + W`: Close active tab
    - `Ctrl + Shift + C`: Copy selection
    - `Ctrl + Shift + V`: Paste from clipboard
    - High-performance PTY management powered by `creack/pty`.
- **🚀 Snippet Library:** Run common commands instantly from a customizable quick-action sidebar.

## 🛠️ Technical Architecture

TermXP is engineered for stability and speed:

- **Backend (Go):** Handles low-level PTY (Pseudo-Terminal) management, process lifecycle, and system telemetry. It uses Wails events for zero-latency data streaming.
- **Frontend (Vue 3 + TypeScript):** A reactive, type-safe interface utilizing `@xterm/xterm` for high-fidelity terminal rendering.
- **IPC:** Seamless bridge between Go and JS/TS using Wails bindings, ensuring high-frequency updates (like terminal output) remain smooth.

## 🚀 Getting Started

### Prerequisites

- **Go:** 1.23+
- **Node.js:** 18+ (with npm)
- **Wails CLI:** `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### Development Mode

Run the app with hot-reloading for both backend and frontend:

```bash
wails dev
```

### Production Build

Generate a platform-native redistributable binary:

```bash
wails build
```

## 📁 Project Structure

- `/app.go`: Application-level logic and telemetry.
- `/terminal.go`: PTY session management and I/O bridging.
- `/frontend/src/`: Vue 3 source code, including the recursive layout engine.
- `/frontend/src/components/`: Core UI components (TerminalInstance, TerminalLayout).
- `/frontend/src/utils/layout.ts`: The recursive tree logic for split-pane management.

---

Built with ❤️ for the terminal enthusiasts.
