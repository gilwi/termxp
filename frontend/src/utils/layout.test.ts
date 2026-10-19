import { describe, it, expect } from "vitest";
import {
  createTerminalNode,
  findNode,
  moveNode,
  removeNode,
  splitNode,
  terminals,
} from "./layout";
describe("layout identities", () => {
  it("preserves session owners through split, swap, move and close", () => {
    const a = createTerminalNode("a", "/bin/sh", "/tmp");
    let root = splitNode(a, "a", "b", "vertical");
    const b = findNode(root, "b")!.node;
    root = splitNode(root, "b", "c", "horizontal");
    root = moveNode(root, "a", "c", "swap");
    expect(findNode(root, "a")!.node).toBe(a);
    expect(findNode(root, "b")!.node).toBe(b);
    root = moveNode(root, "a", "b", "bottom");
    expect(
      terminals(root)
        .map((n) => n.id)
        .sort(),
    ).toEqual(["a", "b", "c"]);
    expect(findNode(root, "a")!.node).toBe(a);
    root = removeNode(removeNode(root, "c")!, "b")!;
    expect(root).toBe(a);
    expect(removeNode(root, "a")).toBeNull();
  });
  it("ignores invalid moves and self drops", () => {
    const root = splitNode(createTerminalNode("a"), "a", "b", "vertical");
    expect(moveNode(root, "a", "missing", "left")).toBe(root);
    expect(moveNode(root, "a", "a", "swap")).toBe(root);
  });
  it("collapses empty nested branches", () => {
    let root = splitNode(createTerminalNode("a"), "a", "b", "vertical");
    root = splitNode(root, "b", "c", "horizontal");
    root = removeNode(root, findNode(root, "b")!.parent!.id)!;
    expect(root.id).toBe("a");
  });
});
