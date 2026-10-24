# Validation

## Automated

```sh
npm --prefix frontend ci
npm --prefix frontend test
npm --prefix frontend run build
go test -race ./...
go vet ./...
npm --prefix frontend audit --audit-level=moderate
go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
xvfb-run -a -s '-screen 0 1280x900x24' python3 scripts/dev-origin-smoke.py
wails build -tags webkit2_41 -s -m -nosyncgomod
xvfb-run -a -s '-screen 0 1280x900x24' python3 scripts/native-smoke.py build/bin/termxp
```

Run build steps sequentially: the Go binary embeds the completed frontend bundle. The scanner requires Go 1.26+, which Go's automatic toolchain selection can obtain. Native smoke additionally needs Xvfb, libXtst, Python 3, and ImageMagick (failure screenshots). GTK is forced onto the virtual X display even when the host uses Wayland. Temporary configuration/cache paths and a clean shell isolate the test from user preferences.

Frontend tests cover layout identity/normalization, session subscription lifetime, consumption acknowledgements, retry/reconnect, input ordering, inactive-tab close, and persistence descriptors. Backend tests cover lifecycle races, cleanup, blocked writes, bounded slow-consumer output, Unicode boundaries/final bytes, sustained 8 MiB output, configuration validation/atomic writes/permissions, and SSH arguments. The native smoke covers actual shell input, splits, reconnect, tabs, palette, settings saves, workspace restore, and window close.

## Manual release checks

- Move/swap panes repeatedly while two shells print distinct markers; type, paste, search, resize, maximize, switch tabs, and close. Input/output must stay with the same shell.
- Test `vim`/`less`/`top`, wide/combining characters, emoji, long lines, and rapidly changing dimensions.
- Paste multiline text with bracketed paste enabled and disabled. Test clipboard permission failures and optional copy-on-select.
- Broadcast into two harmless local shells. Verify destination labels and that changing tabs disables broadcast.
- Exercise SSH profiles against controlled hosts: keys with spaces in paths, jump hosts, password prompts, host-key confirmation, connection failure, and reconnect. Remote cwd discovery is not implemented.
- Start/stop logs, test a pre-existing path (must refuse), inaccessible directories, and disk-write failure.
- Run sustained output with slow rendering and repeated process churn; record Go/webview/process-tree RSS and input latency. The protocol bounds outstanding data, but no long-duration whole-app memory plateau claim is made yet.
- Test corrupt/unsupported config, unavailable saved shells/directories, and window close immediately after an edit.

The development smoke starts an isolated Wails/Vite server and verifies missing/foreign origins cannot upgrade the IPC connection. CI uses Go 1.27.1; go.mod retains the minimum Go version.

Linux CI runs unit/race/type/build/dependency checks, builds a package, and runs the native smoke test. It uploads the executable as a build artifact; it does not publish a release.

## Recovery regression coverage

The follow-up tests additionally exercise final-save rejection/retry, configuration recovery guards and exact backup preservation, legacy SSH IDs, shortcut/snippet defaults, shared save ordering and rollback, failed/hung output acknowledgements, malformed packets, input isolation after reconnect, logging error/RPC ordering, blocked disk writes, queue overflow, live split directories, same-SSH/local split choices, profile rename/deletion and remembered tab focus.

Native smoke now temporarily replaces the isolated config directory with a file to force a final-save error. It verifies the application remains open, restores storage and retries close. It then launches with an unsupported-version config, verifies the original bytes survive startup, invokes explicit recovery through the palette and checks the private backup. These operations use only the test's temporary configuration.
