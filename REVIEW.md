# TermXP reliability review — implementation follow-up

The earlier review described an older tree. This document supersedes that snapshot; it is not a comprehensive security audit.

## Implemented

- Wails module/CLI pinned to 2.17.0; frontend toolchain upgraded and lockfile maintained. Build hooks use npm ci.
- Startup/duplicate/shutdown ownership is serialized. Every child has one waiter; PTY/log cleanup and registry removal have explicit completion. Cancellation interrupts PTY I/O and output acknowledgement waits.
- Output is transported as sequenced bytes, one 32 KiB packet at a time, acknowledged after xterm consumes it. Exit follows final output.
- Frontend session controllers survive component remounts. Close cancels pending startup; reconnect creates a new ID. Layout operations preserve terminal identity.
- Paste uses xterm; broadcast destinations are visible and restricted to the selected tab. Exited panes retain output.
- Backend lifecycle controls are unexported. Session/input/dimension limits, executable-shell checks, SSH argument validation, and production CSP are in place.
- Metrics are typed and unavailable values are explicit. Config/workspace persistence is versioned, validated, atomic, and private.

## Evidence and limits

Go race tests/vet, frontend tests/type checking/production build, Linux packaging, npm audit, and govulncheck are part of validation. The native smoke script exercises actual PTYs through the packaged webview using isolated configuration. See `docs/TESTING.md` for commands and scenarios.

Automated regression coverage includes duplicate starts, natural-exit descriptor closure, concurrent shutdown, blocked input, slow consumers, byte boundaries/final output, sustained 8 MiB output, config corruption/permissions, SSH argument construction, layout identity, late startup cancellation, retry, input order, and restore descriptors.

Real SSH connections, long interactive desktop soak tests, distribution signing, and non-Linux platforms remain unvalidated. Process-group cleanup does not promise termination of intentionally detached processes. Production CSP is verified through the Linux native smoke path; other webviews need their own checks. Dependency scans are point-in-time evidence, not proof that the application is vulnerability-free.
