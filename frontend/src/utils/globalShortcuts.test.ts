import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
vi.mock("./sessions", () => ({
  sessions: new Map(), configureInputRouter: vi.fn(), disposeSession: vi.fn(),
}));
import { handleGlobalKeyDown } from "./globalShortcuts";
import { store } from "./store";
import { defaultShortcuts } from "./domain";
import { terminals } from "./layout";
import { disposeSession } from "./sessions";

// Minimal DOM target for the Node test environment; dispatchEvent supplies
// event.target, and closest walks from a textarea to its owning pane wrapper.
class TestElement extends EventTarget {
  constructor(readonly paneId: string | null = null, readonly parent: TestElement | null = null) { super(); }
  closest(selector: string): TestElement | null {
    expect(selector).toBe("[data-pane-id]");
    return this.paneId !== null ? this : this.parent?.closest(selector) || null;
  }
  getAttribute(name: string) {
    expect(name).toBe("data-pane-id");
    return this.paneId;
  }
}
function press(target: TestElement, key = "W", repeat = false, shiftKey = true) {
  const event = Object.assign(new Event("keydown", { cancelable: true }), {
    key, ctrlKey: true, shiftKey, altKey: false, metaKey: false, repeat,
  });
  target.addEventListener("keydown", handleGlobalKeyDown as EventListener, { once: true });
  target.dispatchEvent(event);
  return event;
}
function textarea(id: string) { return new TestElement(null, new TestElement(null, new TestElement(id))); }
function split() {
  store.addTab();
  const older = store.activePaneId;
  store.splitPane(older, "vertical");
  return { older, newer: store.activePaneId, tab: store.getActiveTab()! };
}
beforeEach(() => {
  vi.clearAllMocks();
  vi.stubGlobal("Element", TestElement);
  store.tabs = [];
  store.activeTabId = "";
  store.activePaneId = "";
  store.maximizedPaneId = null;
  store.paletteOpen = false;
  store.ready = true;
  store.shortcuts = { ...defaultShortcuts };
});
afterEach(() => vi.unstubAllGlobals());

describe("global shortcut target", () => {
  it("closes the event's older pane despite stale active-pane state", () => {
    const { older, newer, tab } = split();
    expect(store.activePaneId).toBe(newer);
    expect(press(textarea(older)).defaultPrevented).toBe(true);
    expect(terminals(tab.rootNode).map(n => n.id)).toEqual([newer]);
    expect(disposeSession).toHaveBeenCalledExactlyOnceWith(older);
    expect(store.activePaneId).toBe(newer);
    expect(tab.focusedPaneId).toBe(newer);
  });
  it("resolves an older pane inside nested splits", () => {
    const { older, newer, tab } = split();
    store.splitPane(newer, "horizontal");
    const newest = store.activePaneId;
    press(textarea(older));
    expect(terminals(tab.rootNode).map(n => n.id)).toEqual([newer, newest]);
    expect(disposeSession).toHaveBeenCalledExactlyOnceWith(older);
  });
  it("remembers the event's focused pane across tab switches", () => {
    const { older, tab } = split();
    press(textarea(older), "P", false, false); // Opening the palette preserves the pane.
    expect(store.paletteOpen).toBe(true);
    expect(tab.focusedPaneId).toBe(older);
    store.paletteOpen = false;
    store.addTab();
    store.selectTab(tab.id);
    expect(store.activePaneId).toBe(older);
    press(textarea(older));
    expect(disposeSession).toHaveBeenCalledWith(older);
  });
  it("keeps the active target when invoked outside a terminal", () => {
    const { older, newer, tab } = split();
    press(new TestElement());
    expect(terminals(tab.rootNode).map(n => n.id)).toEqual([older]);
    expect(disposeSession).toHaveBeenCalledExactlyOnceWith(newer);
  });
  it("rejects a pane from an inactive tab", () => {
    const { older, tab } = split();
    store.addTab();
    const active = store.activePaneId;
    press(textarea(older));
    expect(store.activePaneId).toBe(active);
    expect(terminals(tab.rootNode)).toHaveLength(2);
    expect(disposeSession).not.toHaveBeenCalled();
  });
  it("does not close a different pane for a detached or invalid target", () => {
    split();
    press(textarea("missing-pane"));
    expect(disposeSession).not.toHaveBeenCalled();
  });
  it("ignores held close-pane and close-tab shortcuts", () => {
    const { older, newer, tab } = split();
    press(textarea(older));
    expect(press(textarea(newer), "W", true).defaultPrevented).toBe(true);
    store.shortcuts["close-tab"] = "Ctrl+Shift+Q";
    press(textarea(newer), "Q", true);
    expect(store.tabs).toHaveLength(1);
    expect(terminals(tab.rootNode).map(n => n.id)).toEqual([newer]);
    expect(disposeSession).toHaveBeenCalledExactlyOnceWith(older);
  });
  it("leaves shortcuts alone during startup or while the palette is open", () => {
    const { older } = split();
    store.ready = false;
    expect(press(textarea(older)).defaultPrevented).toBe(false);
    store.ready = true;
    store.paletteOpen = true;
    expect(press(textarea(older)).defaultPrevented).toBe(false);
    expect(disposeSession).not.toHaveBeenCalled();
  });
});
