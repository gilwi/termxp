import { afterEach, beforeEach, expect, it, vi } from "vitest";
const m = vi.hoisted(() => ({ selection: "selected", select: () => {}, key: (_e: KeyboardEvent): boolean => true, paste: vi.fn(), report: vi.fn(), state: { status: "running" } }));
vi.mock("@xterm/xterm", () => ({ Terminal: class {
  loadAddon() {}
  getSelection() { return m.selection; }
  hasSelection() { return !!m.selection; }
  paste = m.paste;
  attachCustomKeyEventHandler(h: typeof m.key) { m.key = h; }
  onSelectionChange(h: typeof m.select) { m.select = h; }
} }));
vi.mock("@xterm/addon-fit", () => ({ FitAddon: class {} }));
vi.mock("@xterm/addon-search", () => ({ SearchAddon: class {} }));
vi.mock("./sessionController", () => ({ SessionController: class { state = m.state; report = m.report; } }));
vi.mock("./bridge", () => ({ terminalBridge: {}, clipboardBridge: { readText: vi.fn(), writeText: vi.fn() } }));
import { clipboardBridge } from "./bridge";
import { acquireSession, sessions } from "./sessions";
function key(key: string, options: Partial<KeyboardEvent> = {}) {
  const e = { key, ctrlKey: true, shiftKey: true, type: "keydown", repeat: false, preventDefault: vi.fn(), ...options } as unknown as KeyboardEvent;
  expect(m.key(e)).toBe(false);
  expect(e.preventDefault).toHaveBeenCalled();
}
function view() { return acquireSession({ id: "pane", type: "terminal", shell: "", cwd: "" }, {}, 14); }
beforeEach(() => {
  vi.resetAllMocks();
  m.selection = "selected";
  m.state.status = "running";
  sessions.clear();
  vi.stubGlobal("document", { createElement: () => ({ style: {} }) });
  vi.mocked(clipboardBridge.writeText).mockResolvedValue(undefined);
  vi.mocked(clipboardBridge.readText).mockResolvedValue("pasted");
});
afterEach(() => vi.unstubAllGlobals());
it("copies only enabled, nonempty selections", () => {
  const v = view();
  m.select();
  expect(clipboardBridge.writeText).not.toHaveBeenCalled();
  v.copyOnSelect = true;
  m.select();
  expect(clipboardBridge.writeText).toHaveBeenCalledWith("selected");
  vi.mocked(clipboardBridge.writeText).mockClear();
  m.selection = "";
  m.select();
  key("c");
  expect(clipboardBridge.writeText).not.toHaveBeenCalled();
});
it("copies by shortcut once per keypress", () => {
  view(); key("C"); key("c", { type: "keyup" }); key("c", { repeat: true });
  expect(clipboardBridge.writeText).toHaveBeenCalledExactlyOnceWith("selected");
});
it("pastes through terminal paste", async () => {
  view(); key("v");
  await vi.waitFor(() => expect(m.paste).toHaveBeenCalledWith("pasted"));
});
it("does not paste if terminal stops before read completes", async () => {
  let resolve!: (text: string) => void;
  vi.mocked(clipboardBridge.readText).mockReturnValue(new Promise(r => { resolve = r; }));
  view(); key("v"); m.state.status = "exited"; resolve("late");
  await Promise.resolve();
  expect(m.paste).not.toHaveBeenCalled();
});
it("reports copy and paste failures", async () => {
  vi.mocked(clipboardBridge.writeText).mockRejectedValue(new Error("write failed"));
  vi.mocked(clipboardBridge.readText).mockRejectedValue(new Error("read failed"));
  const v = view(); v.copyOnSelect = true; m.select(); key("v");
  await vi.waitFor(() => {
    expect(m.report).toHaveBeenCalledWith("Could not copy to clipboard: Error: write failed");
    expect(m.report).toHaveBeenCalledWith("Could not paste from clipboard: Error: read failed");
  });
});
