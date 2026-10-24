# Reliability recovery implementation plan

This follow-up keeps the Go/Wails desktop architecture and closes the failure paths identified in the implementation review. Work is developed on `feature/reliability-recovery` from `e23c1e4`.

## Delivery and commit groups

1. **Backend recovery and service ownership.** Refuse autosaves over unreadable/unsupported config; expose an explicit recovery operation that copies original bytes to a private backup. Add compatible optional preferences and stable SSH profile IDs. Extract launch policy, output transport, and bounded asynchronous logging while keeping lifecycle finalization in TerminalService.
2. **Typed frontend recovery and consistent features.** Introduce domain contracts and a Wails adapter; coordinate autosave, explicit edits and final saves through one queue. Propagate failures to close/retry UI. Make output faults reconnectable, keep logging state controller-owned, and support live split context, same-SSH splits, per-tab focus, stable profile edits/deletion, custom snippets and shortcuts.
3. **Native regressions and documentation.** Exercise failed final saves/retries and explicit config recovery in the packaged webview. Document migration, limits and operational behavior; run the complete validation set before pushing.

## Defined behavior

- Config remains version 1 with additive fields. Missing snippets/shortcuts receive defaults. Legacy SSH profiles receive deterministic IDs and matching inline workspace connections are linked once. New profiles use UUIDs. Editing changes reconnect descriptors without changing running processes. Deletion detaches existing panes and retains connection snapshots.
- One persistence coordinator serializes edits and saves, takes snapshots at execution time and rolls back failed explicit edits. Invalid original config is never silently overwritten. Final-save failure leaves the application open, with retry and explicit close-without-saving actions.
- Logging has a 32-chunk queue (at most 1 MiB queued, plus the active write). Overflow stops logging and reports an error; terminal output continues. File open/write/close never holds a session lifecycle lock. Finalization allows 200 ms to drain, then cancels without waiting on stalled filesystem I/O.
- Protocol sequence/encoding faults and failed or timed-out acknowledgement RPCs move a controller to failed, unsubscribe and cancel the old session. Reconnect uses a new ID while retaining scrollback. Acknowledgement RPC timeout is five seconds.
- Local splits use the most recent reported directory. SSH panes offer an explicit same-connection split; ordinary splits create local shells. Reconnect errors allow a replacement directory and, for local panes, the system default shell.
- Shortcut bindings use canonical Ctrl/Alt/Shift plus a letter or digit, with empty values disabling an action. Clipboard bindings remain reserved. Snippets and shortcuts are editable in the sidebar and persisted with other preferences.

## Acceptance checks

- Unit/race coverage: corruption/unsupported-version preservation, exact backup bytes, legacy migration, stalled/failed/overflowed logging, persistence ordering/rollback/retry, malformed output, missing acknowledgement completion, profile reconnection descriptors, split cwd and remembered focus.
- Native smoke: real shell I/O, splits, reconnect, tabs, palette, final-save fault injection, save retry, workspace restoration and explicit config recovery with backup.
- Production frontend build, Go vet, dependency scans, Linux package build and authenticated IPC origin checks.

Real remote SSH hosts, long filesystem stalls on actual network mounts, full desktop soak tests and non-Linux support remain separate environment-dependent validation.
