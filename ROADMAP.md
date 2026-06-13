# Roadmap: Migrating TermXP to a Frameless, Transparent Architecture

This roadmap outlines the steps required to transition **TermXP** from a standard native window layout into a premium, frameless application featuring custom HTML/Vue window decorations and system-level transparency.

---

## 🗺️ Implementation Overview

```
+-------------------------------------------------------------+
| [=] TermXP  (Custom Drag Zone Title Bar)          [-] [[]] [X] | -> Stage 3: Vue Window Controls
+-------------------------------------------------------------+
|                                                             |
|   >_ Terminal Session Layer                                 | -> Stage 4: Layout & Fit Tuning
|                                                             |
|                                                             |
+-------------------------------------------------------------+
 \___________________________________________________________/  -> Stage 2: Webview Corners & Filters

```

---

## 🛠️ Phase 1: Go Backend Configuration

*Goal: Remove native OS borders and configure the window subsystems to allow translucency layers.*

### 1. Update `main.go` Application Options

Modify the initialization block for `options.App` to hand layout control over to the webview:

* **Enable Frameless Flag**: Set `Frameless: true` to strip standard native title bars and borders.
* **Enable Translucency**: Set `WindowIsTranslucent: true`.

### 2. Add Platform-Specific Tuning

Configure native backdrop engines within `options.App` for smooth rendering:

* **Windows Subsystem**: Populate `windows.Options` with `WebviewIsTransparent: true` and set `BackdropType` to `windows.Mica` (or `windows.Acrylic`).
* **macOS Subsystem**: Populate `mac.Options` utilizing `mac.TitleBarHiddenOrTransparent()` alongside `WebviewIsTransparent: true`.

---

## 🎨 Phase 2: Core CSS Layout & Geometry

*Goal: Prevent default HTML boundaries from breaking webview opacity and set up application framing.*

### 1. Root & HTML Level Overrides (`frontend/src/style.css`)

* Force `html` and `body` backgrounds to `transparent !important` to let the OS-level backdrop filter shine through.
* Lock overflow to `hidden` to suppress unnecessary native window scrollbars.

### 2. Set Up the Outer Wrapper Application Layer

* Target `#app` to stretch exactly `100vw` and `100vh` using a flex direction of `column`.
* Apply a modern border radius (e.g., `border-radius: 12px`) and set `overflow: hidden` to neatly clip child components to the rounded window bounds.
* **Glassmorphic Theme Check**: Update variables within your theme configurations to use alpha-channel values (`rgba`) and `backdrop-filter: blur()` instead of solid hex codes.

---

## 🧩 Phase 3: Custom Title Bar & Control Handlers (Vue 3)

*Goal: Replace the lost native OS dragging features and close/minimize/maximize buttons.*

### 1. Create a `CustomTitleBar.vue` Component

* **Implement Drag Zone**: Add the inline style attribute `style="--wails-drag-zone: drag"` to the header container element. This tells the Wails platform architecture which area should move the native window.
* **Implement Interactive Safe Zones**: Ensure any button or interactive element inside the header explicitly declares `style="--wails-drag-zone: no-drag"`. This prevents dragging logic from swallowing user mouse clicks.

### 2. Wire Up Window Lifecycle Actions

Import runtime hooks directly from the Wails runtime framework:

* Bind your UI minimize icon trigger to call `WindowMinimise()`.
* Bind your UI maximize icon trigger to call `WindowToggleMaximise()`.
* Bind your UI close icon trigger to call `Quit()`.

---

## 📐 Phase 4: Layout Calculations & Edge Cases

*Goal: Polish interactions so that dynamic resizing doesn't break terminal terminal scaling.*

### 1. Handle Window Maximization Adjustments

* **The Problem**: When a window maximizes, a rounded corner (`border-radius: 12px`) leaves awkward screen gaps at the corners of the monitor.
* **The Fix**: Listen to the Wails event engine using `WindowIsMaximised()`. Bind a dynamic reactive class to the `#app` frame component to strip the `border-radius` and outer borders cleanly whenever the application is full-screen.

### 2. Recalculate Terminal Fit Layouts

* **The Problem**: Introducing a custom title bar shifts the height available for terminal outputs, which can cut off lines or cause overflow issues.
* **The Fix**: Ensure that any logic utilizing `@xterm/addon-fit` within terminal layout files is triggered *after* the custom title bar finishes rendering to prevent text calculation mismatches.
