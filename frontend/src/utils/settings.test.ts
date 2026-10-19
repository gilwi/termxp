import { describe, expect, it, vi } from "vitest";
vi.mock("./sessions", () => ({
  sessions: new Map(),
  configureInputRouter: vi.fn(),
  disposeSession: vi.fn(),
}));
vi.mock("../../wailsjs/go/main/ConfigService", () => ({
  LoadConfig: vi.fn(),
  SaveConfig: vi.fn(),
}));
vi.mock("../../wailsjs/go/main/TerminalService", () => ({
  ListShells: vi.fn(),
}));
import { store } from "./store";
import { describeNode, restoreNode, snapshotConfig } from "./settings";
import { main } from "../../wailsjs/go/models";
import { createTerminalNode, splitNode, terminals } from "./layout";
describe("workspace persistence", () => {
  it("restores descriptors with fresh identities", () => {
    const root = splitNode(
      createTerminalNode("a", "/bin/sh", "/tmp"),
      "a",
      "b",
      "vertical",
    );
    const descriptor = describeNode(root);
    expect(JSON.stringify(descriptor)).not.toContain('"id"');
    expect(JSON.stringify(descriptor)).not.toContain("sessionId");
    const restored = restoreNode(new main.WorkspaceNode(descriptor));
    expect(terminals(restored).map((n) => n.id)).not.toEqual(["a", "b"]);
    expect(terminals(restored)[0].cwd).toBe("/tmp");
  });
  it("stores names and settings without broadcast, metrics or buffers", () => {
    store.tabs = [];
    store.defaultShell = "/bin/sh";
    store.addTab();
    store.tabs[0].name = "Build";
    store.broadcast = true;
    const snapshot = snapshotConfig();
    expect(snapshot.workspace[0].name).toBe("Build");
    expect(snapshot.defaultShell).toBe("/bin/sh");
    expect(snapshot).not.toHaveProperty("broadcast");
    expect(snapshot).not.toHaveProperty("stats");
  });
});
