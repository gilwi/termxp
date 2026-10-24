import { describe, expect, it, vi } from "vitest";
vi.mock("./sessions", () => ({
  sessions: new Map(),
  configureInputRouter: vi.fn(),
  disposeSession: vi.fn(),
}));
vi.mock("./bridge", () => ({
  settingsBridge: {
    load: vi.fn(),
    save: vi.fn().mockResolvedValue(undefined),
    recover: vi.fn(),
    shells: vi.fn(),
  },
}));
import { store } from "./store";
import {
  describeNode,
  restoreNode,
  snapshotConfig,
  initializeSettings,
  persistence,
} from "./settings";

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
    const restored = restoreNode(descriptor);
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

import { settingsBridge } from "./bridge";
import { defaultShortcuts, defaultSnippets } from "./domain";
describe("settings initialization and shutdown", () => {
  it("loads preferences even when shell discovery fails, and propagates close-save failure", async () => {
    store.tabs = [];
    vi.mocked(settingsBridge.load).mockResolvedValue({
      config: {
        version: 1,
        theme: "dracula",
        fontSize: 18,
        defaultShell: "",
        copyOnSelect: false,
        restoreWorkspace: true,
        workspace: [],
        sshProfiles: [],
        snippets: defaultSnippets,
        shortcuts: defaultShortcuts,
      },
      warning: "",
      recoveryRequired: false,
    });
    vi.mocked(settingsBridge.shells).mockRejectedValue(
      new Error("discovery failed"),
    );
    const session = await initializeSettings();
    expect(store.currentTheme).toBe("dracula");
    vi.mocked(settingsBridge.save).mockRejectedValueOnce(
      new Error("disk full"),
    );
    await expect(session.flush()).rejects.toThrow("disk full");
    await expect(session.flush()).resolves.toBeUndefined();
    session.dispose();
  });
  it("refuses autosave of fallback preferences while recovery is pending", async () => {
    vi.mocked(settingsBridge.load).mockResolvedValue({
      config: snapshotConfig(),
      warning: "unsupported version",
      recoveryRequired: true,
    });
    vi.mocked(settingsBridge.shells).mockResolvedValue([]);
    const session = await initializeSettings();
    await expect(session.flush()).rejects.toThrow("recovery");
    expect(persistence.blocked).toBe(true);
    session.dispose();
    persistence.blocked = false;
  });
});
