import { beforeEach, describe, expect, it, vi } from "vitest";
vi.mock("./sessions", () => ({
  sessions: new Map(),
  configureInputRouter: vi.fn(),
  disposeSession: vi.fn(),
}));
import { store } from "./store";
import { disposeSession, sessions } from "./sessions";
import { terminals } from "./layout";
beforeEach(() => {
  store.tabs = [];
  store.activeTabId = "";
  store.activePaneId = "";
  store.maximizedPaneId = null;
  store.broadcast = false;
  vi.clearAllMocks();
  sessions.clear();
});
describe("workspace state", () => {
  it("closes inactive panes without selecting their tab", () => {
    store.addTab();
    const first = store.tabs[0];
    store.splitPane(store.activePaneId, "vertical");
    const pane = store.activePaneId;
    store.addTab();
    const selected = store.activeTabId;
    store.closePane(pane);
    expect(store.activeTabId).toBe(selected);
    expect(terminals(first.rootNode)).toHaveLength(1);
    expect(disposeSession).toHaveBeenCalledWith(pane);
  });
  it("reveals the new pane when splitting a maximized terminal", () => {
    store.addTab();
    store.toggleMaximize();
    store.splitPane(store.activePaneId, "vertical");
    expect(store.maximizedPaneId).toBeNull();
  });
  it("resets cross-tab maximization and broadcast", () => {
    store.addTab();
    const first = store.tabs[0].id;
    store.addTab();
    store.toggleMaximize();
    store.broadcast = true;
    store.selectTab(first);
    expect(store.maximizedPaneId).toBeNull();
    expect(store.broadcast).toBe(false);
  });
});

describe("launch context and profile identities", () => {
  it("uses the latest controller directory for a split", () => {
    store.addTab();
    const id = store.activePaneId;
    sessions.set(id, {
      controller: { state: { cwd: "/new-directory" } },
    } as never);
    store.splitPane(id, "vertical");
    expect(store.getActiveNode()?.cwd).toBe("/new-directory");
    sessions.clear();
  });
  it("offers local or same-profile splits without sharing mutable launch options", () => {
    const profile = {
      id: "ssh-1",
      name: "build",
      host: "example.org",
      user: "",
      port: 22,
      keyPath: "",
      jumpHost: "",
    };
    store.addTab(profile);
    const first = store.activePaneId;
    store.splitPane(first, "vertical", "same");
    expect(store.getActiveNode()?.sshProfileId).toBe("ssh-1");
    expect(store.getActiveNode()?.ssh).toEqual(profile);
    store.splitPane(first, "horizontal", "local");
    expect(store.getActiveNode()?.ssh).toBeUndefined();
  });
  it("returns to each tab's last focused pane", () => {
    store.addTab();
    const first = store.activeTabId;
    store.splitPane(store.activePaneId, "vertical");
    const focused = store.activePaneId;
    store.addTab();
    const second = store.activeTabId;
    store.selectTab(first);
    expect(store.activePaneId).toBe(focused);
    store.selectTab(second);
    store.selectTab(first);
    expect(store.activePaneId).toBe(focused);
  });
  it("updates reconnect descriptors by stable profile ID and detaches on deletion", () => {
    const profile = {
      id: "ssh-1",
      name: "build",
      host: "old.example",
      user: "",
      port: 22,
      keyPath: "",
      jumpHost: "",
    };
    store.sshProfiles = [profile];
    store.addTab(profile);
    const id = store.activePaneId,
      updateSSH = vi.fn();
    sessions.set(id, {
      controller: { updateSSH, state: { cwd: "/tmp" } },
    } as never);
    store.updateSSHProfile({
      ...profile,
      name: "renamed",
      host: "new.example",
    });
    expect(store.getActiveNode()?.ssh?.host).toBe("new.example");
    expect(updateSSH).toHaveBeenCalled();
    store.deleteSSHProfile("ssh-1");
    expect(store.getActiveNode()?.sshProfileId).toBeUndefined();
    expect(store.getActiveNode()?.ssh?.host).toBe("new.example");
    sessions.clear();
  });
});
