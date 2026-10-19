# TermXP roadmap

Linux-first; prioritize reliable daily use. The original 14-feature backlog is implemented in this working tree, with the validation limits below.

| Original feature | State |
|---|---|
| Settings persistence | Implemented: versioned atomic private JSON saves |
| Shell detection | Implemented: validated `$SHELL`, fallbacks, `/etc/shells` discovery |
| Session reconnect | Implemented: retained output, status, fresh session IDs |
| Shell picker | Implemented: workspace selector and persisted default |
| Font size | Implemented: controls, Ctrl+scroll, persistence |
| Tab rename | Implemented and persisted |
| Copy on select | Implemented as opt-in preference |
| Working directory labels | Implemented on Linux; SSH remote cwd is not detected |
| Scrollback search | Implemented: Ctrl+F, highlighting, next/previous |
| Session logging | Implemented: explicit new-file path, private permissions |
| Layout save/restore | Implemented: fresh sessions from descriptors |
| Pane broadcast | Implemented: selected-tab scope, visible destinations |
| Command palette | Implemented: fuzzy action search, Ctrl+P |
| SSH profiles | Implemented: host/user/port/key path/jump host; local validation only |

## Reliability foundation delivered

- Atomic duplicate-session rejection, process reaping, explicit PTY cleanup, cancellable I/O, bounded shutdown.
- Component-independent frontend controllers, cancellable startup, stable identity across layout changes, retained exited panes.
- Sequenced byte transport with consumption acknowledgements and bounded outstanding output.
- Typed layout/config/metrics, reproducible dependency installs, upgraded frontend tooling, Linux CI/package/native smoke checks.

## Follow-up validation and development

1. Test real SSH hosts, jump hosts, authentication prompts, and disconnected networks in a controlled environment.
2. Run longer desktop soak tests, including high output, interactive full-screen programs, multiline paste, selection, and drag-and-drop during output. Measure full Go/webview/process-tree memory and latency on representative hardware.
3. Evaluate larger bounded output windows only if measured throughput requires them; preserve acknowledgement/cancellation guarantees.
4. Add Linux distribution packaging/signing and release automation once native validation is stable.
5. Add macOS native PTY/metrics validation, then a Windows ConPTY adapter before claiming support there.

No calendar estimate is assigned to these follow-ups until their environments and acceptance criteria are established. See `docs/TESTING.md` for automated checks and remaining manual scenarios.
