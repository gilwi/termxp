# TermXP development notes

Read README.md, docs/ARCHITECTURE.md, and docs/TESTING.md for the current implementation and checks. ROADMAP.md records feature status and remaining validation.

Keep Wails CLI/module versions aligned at 2.17.0. Use npm ci and commit dependency lockfile changes. Regenerate frontend bindings with `wails generate module`; do not hand-edit generated files.

Go owns PTYs/processes; frontend session controllers own xterm/input/subscription lifetimes independently of Vue mounts. Layout operations preserve stable pane identities. Output is sequenced base64 bytes with consumption acknowledgements. Treat shell output as untrusted and keep it out of HTML rendering. Lifecycle methods stay unexported from bound services.

The supported target is Linux. Build with `-tags webkit2_41` on systems using WebKitGTK 4.1. Keep frontend generation/build and native compilation sequential because Go embeds the completed bundle. Do not claim macOS or Windows support without native adapters and validation.
