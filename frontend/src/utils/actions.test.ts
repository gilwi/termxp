import { beforeEach, describe, expect, it, vi } from "vitest";
vi.mock("./sessions", () => ({
  sessions: new Map(),
  configureInputRouter: vi.fn(),
  disposeSession: vi.fn(),
}));
import { store } from "./store";
import { actions, fuzzyMatch } from "./actions";
import { terminals } from "./layout";
import { disposeSession, sessions } from "./sessions";

beforeEach(() => {
  store.tabs = [];
  store.activeTabId = "";
  store.activePaneId = "";
  store.maximizedPaneId = null;
  store.broadcast = false;
  vi.clearAllMocks();
  sessions.clear();
});

describe("actions and shortcuts", () => {
  it("close action only closes current pane, preserving the tab when multiple panes exist", () => {
    store.addTab();
    const tabId = store.activeTabId;
    store.splitPane(store.activePaneId, "vertical");
    expect(terminals(store.tabs[0].rootNode)).toHaveLength(2);

    const closedPaneId = store.activePaneId;
    const closeAction = actions().find((a) => a.id === "close");
    expect(closeAction).toBeDefined();
    expect(closeAction?.label).toBe("Close current pane");

    closeAction?.run();

    expect(store.tabs).toHaveLength(1);
    expect(store.tabs[0].id).toBe(tabId);
    expect(terminals(store.tabs[0].rootNode)).toHaveLength(1);
    expect(disposeSession).toHaveBeenCalledWith(closedPaneId);
    expect(store.activePaneId).not.toBe(closedPaneId);
  });

  it("close action closes the tab when closing the only pane", () => {
    store.addTab();
    expect(store.tabs).toHaveLength(1);
    const closeAction = actions().find((a) => a.id === "close");
    closeAction?.run();
    expect(store.tabs).toHaveLength(0);
  });

  it("close-tab action closes the entire tab including all panes", () => {
    store.addTab();
    store.splitPane(store.activePaneId, "vertical");
    expect(store.tabs).toHaveLength(1);
    expect(terminals(store.tabs[0].rootNode)).toHaveLength(2);

    const closeTabAction = actions().find((a) => a.id === "close-tab");
    expect(closeTabAction).toBeDefined();
    expect(closeTabAction?.label).toBe("Close current tab");

    closeTabAction?.run();
    expect(store.tabs).toHaveLength(0);
  });

  it("fuzzy matches close actions in palette", () => {
    expect(fuzzyMatch("close pane", "Close current pane")).toBe(true);
    expect(fuzzyMatch("close tab", "Close current tab")).toBe(true);
  });
});
