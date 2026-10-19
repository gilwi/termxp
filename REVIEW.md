# TermXP architecture, security, and reliability review

Reviewed 2026-10-19 against the current working tree, including existing uncommitted frontend changes. Application source was not modified. Findings below are from source inspection unless a check is explicitly reported. No exploit was run, no GUI session was exercised, and no complete dependency vulnerability scan was performed.

## Architecture and trust boundaries

This is a small desktop monolith: main.go embeds the built Vue application and binds App and TerminalService through Wails. App supplies process metrics. TerminalService owns a map of shell processes and PTYs, forwards input/resizes through method calls, and streams output through per-session Wails events. Vue holds tabs and recursive layout state in a global reactive store; a separate global cache retains xterm instances and DOM wrappers across component remounts.

The framework and xterm choices fit the product. The main architectural weakness is ownership: layout nodes, mounted components, cached terminal instances, event subscriptions, and Go sessions each hold parts of a terminal's identity and lifetime. They can disagree. The frontend bridge has the user's shell privileges, so protecting it is a security boundary. Shell output is untrusted even when the shell is local (SSH, downloaded files, build output). Current output goes through xterm.write rather than HTML insertion; no direct output-to-HTML execution sink was found in the inspected source.

Do not introduce microservices. Keep a small Go application, give a session manager explicit lifecycle ownership, and give the frontend one session-controller registry independent of Vue component mounts. Make layout operations operate on stable pane identities. Separate pure layout functions, session effects, settings, themes, and metrics.

## Findings, ordered by priority

### P0 — Upgrade Wails before further development use

go.mod pins Wails v2.12.0. Its installed internal/frontend/devserver/devserver.go:37 has CheckOrigin returning true. The upstream advisory GHSA-qg59-r6hv-pvm6 affects versions below 2.17.0 and describes arbitrary-origin access to /wails/ipc while wails dev runs. This app exposes StartSession and WriteToTerminal, so unauthorized bound-method invocation can plausibly become command execution as the developer. This application-specific consequence is inferred from the methods; it was not exploited here. Loopback binding alone does not address cross-origin browser access. This finding concerns development mode, not proof of the same path in a packaged production application.

Action: upgrade the module and the CLI to a compatible patched release (at least 2.17.0 for this advisory), regenerate bindings, and verify an untrusted browser Origin cannot connect or invoke a method. Keep development servers stopped until patched. Review the other current Wails advisories for platform and feature applicability.

Source: https://github.com/wailsapp/wails/security/advisories/GHSA-qg59-r6hv-pvm6

### P1 — Pane swap can route input to the wrong shell

frontend/src/utils/layout.ts:144 swaps only sessionId values. TerminalInstance.vue binds output listeners once to the initial sId, while input handlers read the current reactive sessionId. The xterm cache is keyed by paneId. After swap, a pane can still display shell A while writing to shell B. This risks commands or secrets going to the wrong destination.

Action: move complete pane/session identities, or explicitly rebind both input and output atomically. Have handlers refer to a stable controller, not props captured by the first mounted component. Test swap, move, split, tab switching, typing, paste, and close with two clearly distinguishable shells.

### P1 — Process lifecycle leaks and shutdown can block

terminal.go never calls cmd.Wait after pty.Start. Exited/killed children can remain unreaped, and exit status is lost. Failure to set initial size closes the PTY without an explicit process cleanup/wait. Duplicate StartSession IDs overwrite live entries without closing the old session; an old reader can subsequently KillSession by ID and remove a replacement. CleanupAllSessions can race a new StartSession because there is no shutting-down gate.

WriteToTerminal holds session.mu during a potentially blocking PTY write; KillSession and shutdown need that same mutex before closing the descriptor. A shell that does not consume input and a sufficiently large paste can therefore prevent cancellation. Killing only the shell process also provides no explicit policy for foreground jobs or descendants; PTY hangup is not a guarantee all descendants exit.

Action: one lifecycle owner per session; reserve IDs before spawning, reject duplicates, roll back every partial failure, call Wait exactly once, and remove entries only when they still refer to the same session. Make close able to interrupt blocked I/O. Add a shutdown gate and bounded joins. Define process-group/job termination behavior per supported OS, with graceful termination followed by a timeout and forceful cleanup where appropriate.

### P1 — Session state race and exit/output ordering bug

terminal.go:166 reads s.closed under ts.mu, but KillSession and CleanupAllSessions write it under s.mu: these locks do not synchronize that field. At line 163, closing done requests a final flush but does not wait for it. The exit event can arrive before final data, and the frontend closes/disposes the pane on exit, losing trailing output.

Action: synchronize session state consistently. Join the flusher before reporting exit; emit a structured exit status once. Test normal exit, close during output, concurrent write/resize/kill, and final output under go test -race.

### P1 — Output pipeline has no real bound or backpressure

terminal.go's maxPending is a notification threshold, not a bound: the reader continues appending while the flusher is delayed. There is also no acknowledgement from xterm consumption to Go. A fast producer can build memory pressure in pending data, IPC, and xterm's write queue.

The backend converts arbitrary PTY byte chunks to Go strings before JSON transport. A multibyte UTF-8 character split across flushes can be replaced/corrupted during serialization.

Action: carry bytes losslessly (for example base64 to Uint8Array), use bounded queues and consumption acknowledgements, and apply backpressure rather than silently discarding terminal data. Measure memory and responsiveness under sustained output and slow rendering. Test Unicode across every transport boundary and verify the last bytes precede exit.

### P1 — Custom paste bypasses terminal paste handling

TerminalInstance.vue:128 sends clipboard text directly to WriteToTerminal. This bypasses xterm's paste path, including bracketed-paste handling when enabled by the shell/application. Multiline clipboard text can execute immediately as ordinary input. The custom path also bypasses the onData broadcast behavior, so paste and typing have inconsistent destinations.

Action: use term.paste(text) and one intentional input-routing path. Make multiline execution behavior explicit and broadcast destinations visible. Handle clipboard permission/read/write errors. Test a shell with bracketed paste enabled and disabled.

### P1 — Frontend toolchain includes known vulnerable development tooling

The locked/build-used Vite version is 3.2.11, covered by GHSA-vg6x-rcgg-rjx6 (affected <=4.5.5). Its development server can expose content to malicious browser origins even on a local machine. esbuild is locked at 0.15.18; GHSA-67mh-4wv8-2f99 applies to its own serve functionality, whose use by this app was not established. These are development exposure findings; their presence does not establish a production bundle exploit.

Action: upgrade Vite and its Vue plugin together to a maintained compatible toolchain, update TypeScript/vue-tsc as needed, and audit both npm and Go dependency graphs. Keep explicit trusted dev origins/hosts. Pin the CLI and use npm ci for reproducible builds instead of the current npm install build hook.

Sources:
- https://github.com/vitejs/vite/security/advisories/GHSA-vg6x-rcgg-rjx6
- https://github.com/evanw/esbuild/security/advisories/GHSA-67mh-4wv8-2f99

### P2 — Frontend lifetime and inactive-tab bugs

- store.ts:239 searches other tabs only when there is no active tab. An exit from an inactive tab normally calls performClosePane on the wrong tab, leaving stale state. Find the owning tab independently of selection.
- TerminalInstance.vue's deferred initialization timer is not cancelled on unmount/disposal. Closing a pane immediately can allow its delayed callback to start an orphan session afterward. Async StartSession completion also needs cancellation/ownership checks.
- Cached input/exit callbacks capture the original component's props and emit function. Reattaching the cached DOM after tree restructuring does not replace those callbacks. Make subscriptions and callbacks controller-owned.
- removeNode has split collapsing commented out. Removing the last terminal from a nested branch can leave an empty split occupying space, and getFirstTerminalNode only descends through the first child. Normalize empty/single-child splits and search for the first actual leaf.
- Tab and pane IDs use Date.now, allowing collisions within the same millisecond. Use a reliable unique ID generator. IDs are identities, not an authorization mechanism.
- A global maximizedPaneId can refer to a different tab; switching to a split tab can hide all children. Store maximization per tab or reset it on selection.
- TerminalLayout installs document-level resize/context-menu listeners without unmount cleanup; timed listener installation can outlive a component. Own and dispose listener handles and timers.
- Initialization failures set cached.initialized before success and leave no retry state. Represent starting/running/exited/failed/closing explicitly and provide a recoverable error view. Preserve exit output rather than automatically destroying the pane.

### P2 — Backend API boundary is broader than necessary

Generated TerminalService bindings expose SetContext and CleanupAllSessions alongside normal terminal methods. Lifecycle controls should not be callable from the web UI. StartSession/ResizeTerminal cast arbitrary integers to uint16 without bounds checks; negative and oversized values wrap. There are no service-level session or input-size limits.

Action: bind a narrow facade and keep initialization/shutdown unexported or outside the bound type. Validate identifiers and sensible dimension ranges before spawning; enforce resource limits in Go. A random session ID alone cannot protect a compromised bridge. Add a production CSP compatible with Wails and prevent untrusted navigation/frames from gaining bridge access; verify this across platform webviews. No exploitable app-level XSS was demonstrated.

### P2 — Platform support and metrics need honest contracts

terminal.go hardcodes /bin/bash then /bin/sh, always with --login; /bin/sh implementations need not accept that flag. It ignores the user's preferred shell, can select a non-executable path, and cannot provide Windows terminals through this Unix PTY implementation. macOS/Windows metrics silently report zero. Decide which OSes are supported, use platform adapters/build constraints, and show unavailable metrics honestly. Prefer a validated user shell with shell-specific login arguments.

app.go splits /proc/self/stat on whitespace even though the parenthesized process name may contain spaces, misaligning CPU fields. readCPUStats checks len(fields)>4 but indexes through fields[8]. Numeric parse errors are ignored. CPU usage is divided by aggregate machine time, so document that scale or normalize to the expected per-core convention. Reported Go RSS excludes the separate webview and shell processes; label the scope accurately.

### P3 — Persistence and maintainability after correctness

Settings and layout remain memory-only; ROADMAP.md places persistence first, but bridge security and session correctness should precede it. Then add versioned, validated config with atomic writes and restrictive file permissions. Restore layout descriptors by creating fresh sessions, not persisting runtime session IDs. Keep command history, credentials, and raw session logs out of default config.

Use discriminated TypeScript unions for terminal/split nodes and typed metrics/themes instead of optional fields and any. Keep generated bindings reproducible and avoid exposing context.Context (the generated declaration currently imports a models file absent from the listed tree). Update README's Go prerequisite: it says 1.23+, while go.mod requires 1.25.1. Add CI before expanding SSH, logging, or broadcast features.

## Delivery plan and acceptance criteria

1. **Immediate security patch:** upgrade Wails module and CLI, upgrade the frontend toolchain, regenerate bindings, scan dependencies, and verify cross-origin IPC denial. Acceptance: malicious-origin connection rejected, normal dev/build works, no unexplained reachable high-severity advisories.
2. **Session foundation:** implement lifecycle ownership, process reaping, cancellable I/O, duplicate-ID protection, consistent locking, and shutdown gating. Acceptance: repeated start/exit/close leaves no unreaped children or tracked sessions; blocked writes cannot stall shutdown; concurrent lifecycle tests pass under the race detector.
3. **Identity and input correctness:** stable frontend session controllers, atomic pane movement, owning-tab lookup, cancellable initialization, normalized trees, and terminal-aware paste. Acceptance: output/input/close always target the same shell through swap/move/split/remount; inactive exits and rapid close/start work; paste honors bracketed mode.
4. **Streaming resilience:** lossless byte transport, bounded buffering and render acknowledgements, ordered output/exit events. Acceptance: sustained-output memory plateaus under a defined workload, input remains responsive, Unicode and final output remain intact.
5. **Platform and API hardening:** narrow bound facade, backend validation/limits, CSP/navigation checks, shell adapters, explicit support matrix, metrics fixes. Acceptance: native smoke tests on each claimed OS; invalid requests create no processes and mutate no sessions.
6. **Product improvements:** versioned settings/layout persistence, reconnect/exited-pane UI, typed modules, accessibility/keyboard polish, then SSH/logging. Add logs only with explicit paths, restrictive permissions, and clear handling of secret-bearing output; launch SSH with argument arrays and option validation.

Add regression tests with each fix, focused on the behavior above. Establish CI for npm ci/build, Go test/vet/race, dependency checks, and platform packaging. Do not interpret a green empty test suite as lifecycle coverage.

## Checks performed

- npm run build: passed (vue-tsc and Vite production bundle).
- go test ./...: passed, reports [no test files].
- go vet ./...: passed.
- Reviewed application Go/TypeScript/Vue, generated bindings, build/dependency configuration, README and roadmap; checked primary upstream advisories and installed Wails dev-server code.
- Not performed: interactive packaged-app tests, process-leak reproduction, load tests, cross-platform builds, full npm audit/govulncheck, or exploit testing. The locally available Go toolchain is newer than the declared minimum.
