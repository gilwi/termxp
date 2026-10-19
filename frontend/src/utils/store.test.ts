import { beforeEach, describe, expect, it, vi } from "vitest";
vi.mock("./sessions", () => ({
  sessions: new Map(),
  configureInputRouter: vi.fn(),
  disposeSession: vi.fn(),
}));
import { store } from "./store";
import { disposeSession } from "./sessions";
import { terminals } from "./layout";
beforeEach(() => {
  store.tabs = [];
  store.activeTabId = "";
  store.activePaneId = "";
  store.maximizedPaneId = null;
  store.broadcast = false;
  vi.clearAllMocks();
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
    store.addTab(); store.toggleMaximize(); store.splitPane(store.activePaneId, "vertical");
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
