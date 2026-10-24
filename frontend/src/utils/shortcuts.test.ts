import { describe, it, expect } from "vitest";
import { shortcutAction, shortcutForEvent } from "./shortcuts";
const event = (
  key: string,
  ctrlKey = false,
  altKey = false,
  shiftKey = false,
  metaKey = false,
) => ({ key, ctrlKey, altKey, shiftKey, metaKey });
describe("shared action shortcuts", () => {
  it("recognizes custom bindings consistently in the terminal and app", () => {
    const keys = { new: "Ctrl+Alt+N", close: "" };
    expect(shortcutAction(event("n", true, true), keys)).toBe("new");
    expect(shortcutAction(event("T", true, false, true), keys)).toBeUndefined();
    expect(shortcutAction(event("W", true, false, true), keys)).toBeUndefined();
  });
  it("canonicalizes modifiers and ignores unsupported keys", () => {
    expect(shortcutForEvent(event("K", true, true, true))).toBe(
      "Ctrl+Alt+Shift+K",
    );
    expect(shortcutForEvent(event("Control", true))).toBe("");
    expect(shortcutForEvent(event("p", false, false, false, true))).toBe("");
  });
});
