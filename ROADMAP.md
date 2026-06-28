TermXP — Final Feature Plan
============================
14 features · 4 milestones · ~4–6 weeks total


MILESTONE 1 — Foundation fixes (~1–2 days)
-------------------------------------------
Do these first. Everything else depends on them.

1. Settings persistence
   Store theme, preferences, and future feature config in a JSON file via
   the Wails runtime path. Add SaveConfig / LoadConfig Go methods.
   Files: Go (app.go), Vue (App.vue)

2. Smarter shell detection
   Scan /etc/shells and check $SHELL env var before falling back.
   Fixes silent breakage on macOS (zsh default) and NixOS.
   Files: Go (terminal.go)

3. Session reconnect on crash
   When terminal:exit fires unexpectedly, show a "Reconnect" button in the
   pane instead of leaving a dead terminal. Reuse the existing StartSession call.
   Files: Go (terminal.go), Vue (TerminalInstance.vue)


MILESTONE 2 — Daily driver polish (~1 week, features mostly independent)
-------------------------------------------------------------------------

4. Shell picker on startup
   Dropdown in the new tab dialog, populated from the shell list detected
   in step 2. Persist the default choice to config (step 1).
   Depends on: 1, 2
   Files: Go, Vue

5. Font size control
   Pass fontSize to xterm constructor. Wire Ctrl+scroll and +/− UI controls.
   Persist to config.
   Depends on: 1
   Files: Vue (TerminalInstance.vue), xterm

6. Tab rename
   Double-click a tab label to edit it inline. Store the custom name alongside
   the session ID in the layout tree.
   Files: Vue (App.vue)

7. Copy on select
   Wire xterm's onSelectionChange event to auto-copy to clipboard.
   Files: Vue (TerminalInstance.vue)

8. Working directory in tab label
   Poll /proc/<pid>/cwd (Linux) or lsof (macOS) from Go every few seconds,
   emit via Wails event, display in the tab alongside the custom name.
   Depends on: 6
   Files: Go (terminal.go), Vue

9. Scrollback search
   Load @xterm/addon-search. Ctrl+F opens a floating bar with match
   highlighting and forward/back navigation. No Go changes needed.
   Files: Vue (TerminalInstance.vue), @xterm/addon-search


MILESTONE 3 — Workspace features (~1–2 weeks, sequential)
----------------------------------------------------------

10. Session logging
    Add a LogSession(id, path string) Go method that tees PTY output to a
    file via the existing readLoop. Per-pane toggle in the UI.
    Files: Go (terminal.go), Vue

11. Layout save and restore
    Serialize the layout.ts tree (splits, tab names, shell paths, cwds) to
    config JSON on close. Offer to restore on startup. Add SaveLayout /
    LoadLayout Go method pair.
    Depends on: 1, 6, 8
    Files: Go, Vue (layout.ts)

12. Pane broadcast (sync input)
    A toolbar toggle. When active, keystrokes are written to all session IDs
    in the layout tree via WriteToTerminal, not just the focused pane.
    Pure frontend — no new Go needed.
    Depends on: 11 (best after layout is stable)
    Files: Vue (App.vue)


MILESTONE 4 — Power features (~2–3 weeks, one new Go service)
-------------------------------------------------------------

13. Command palette
    Ctrl+P opens a fuzzy-search overlay listing all actions: new tab,
    split right/below, switch theme, connect SSH, toggle broadcast, run snippet.
    Static action registry in Vue; actions call existing Wails bindings.
    Best after steps 11 and 14 are stable.
    Files: Vue (new component)

14. SSH profile manager
    New SSHService in Go storing profiles (host, user, key path, jump host)
    in the config file. Connecting spawns a PTY session running:
      ssh -i <key> <user>@<host>
    No SSH library needed. Vue sidebar panel lists and launches profiles.
    Depends on: 1, 2, 4
    Files: Go (new ssh.go service), Vue (new sidebar component)


KEY DEPENDENCY NOTE
-------------------
Settings persistence (step 1) must land before anything else.
Steps 4, 5, 11, and 14 all write to config — doing it once cleanly
means every subsequent feature gets persistence for free.

Only one genuinely new Go service is introduced in the entire plan
(SSH profile manager, step 14). Everything else extends terminal.go
or app.go, or lives entirely in the Vue frontend.
