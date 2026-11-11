import { afterEach, describe, expect, it, vi } from "vitest";
import { clipboardBridge } from "./bridge";
afterEach(() => vi.unstubAllGlobals());
describe("native clipboard bridge", () => {
  it("reads and writes through Wails", async () => {
    const runtime = { ClipboardGetText: vi.fn().mockResolvedValue("native text"), ClipboardSetText: vi.fn().mockResolvedValue(true) };
    vi.stubGlobal("window", { runtime });
    expect(await clipboardBridge.readText()).toBe("native text");
    await expect(clipboardBridge.writeText("copied text")).resolves.toBeUndefined();
    expect(runtime.ClipboardSetText).toHaveBeenCalledWith("copied text");
  });
  it("rejects unsuccessful writes", async () => {
    vi.stubGlobal("window", { runtime: { ClipboardSetText: vi.fn().mockResolvedValue(false) } });
    await expect(clipboardBridge.writeText("text")).rejects.toThrow("Native clipboard write was unsuccessful");
  });
  it("rejects when runtime is unavailable", async () => {
    vi.stubGlobal("window", {});
    await expect(clipboardBridge.readText()).rejects.toThrow();
    await expect(clipboardBridge.writeText("text")).rejects.toThrow();
  });
});
