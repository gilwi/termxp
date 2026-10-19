import type { main } from "../../wailsjs/go/models";

export interface TerminalNode {
  id: string;
  type: "terminal";
  shell: string;
  cwd: string;
  ssh?: main.SSHProfile;
}
export interface SplitNode {
  id: string;
  type: "split";
  orientation: "horizontal" | "vertical";
  children: PaneNode[];
  sizes: number[];
}
export type PaneNode = TerminalNode | SplitNode;
export type Position = "left" | "right" | "top" | "bottom" | "swap";
export const generateUUID = (): string => crypto.randomUUID();
export function createTerminalNode(
  id = generateUUID(),
  shell = "",
  cwd = "",
): TerminalNode {
  return { id, type: "terminal", shell, cwd };
}
export function findNode(
  root: PaneNode,
  id: string,
  parent: SplitNode | null = null,
  index = -1,
): { node: PaneNode; parent: SplitNode | null; index: number } | null {
  if (root.id === id) return { node: root, parent, index };
  if (root.type === "split") {
    for (let i = 0; i < root.children.length; i++) {
      const found = findNode(root.children[i], id, root, i);
      if (found) return found;
    }
  }
  return null;
}
export function terminals(root: PaneNode): TerminalNode[] {
  return root.type === "terminal" ? [root] : root.children.flatMap(terminals);
}
function normalize(
  root: SplitNode,
  children: PaneNode[],
  sizes: number[],
): PaneNode | null {
  if (!children.length) return null;
  if (children.length === 1) return children[0];
  const total = sizes.reduce((sum, size) => sum + size, 0);
  return {
    ...root,
    children,
    sizes: sizes.map((s) =>
      total > 0 ? (s / total) * 100 : 100 / children.length,
    ),
  };
}
export function removeNode(root: PaneNode, id: string): PaneNode | null {
  if (root.id === id) return null;
  if (root.type === "terminal") return root;
  const children: PaneNode[] = [],
    sizes: number[] = [];
  root.children.forEach((child, i) => {
    const next = removeNode(child, id);
    if (next) {
      children.push(next);
      sizes.push(root.sizes[i] ?? 1);
    }
  });
  return normalize(root, children, sizes);
}
function replaceNode(
  root: PaneNode,
  id: string,
  replacement: PaneNode,
): PaneNode {
  if (root.id === id) return replacement;
  if (root.type === "terminal") return root;
  return {
    ...root,
    children: root.children.map((c) => replaceNode(c, id, replacement)),
  };
}
function split(
  first: PaneNode,
  second: PaneNode,
  orientation: SplitNode["orientation"],
): SplitNode {
  return {
    id: generateUUID(),
    type: "split",
    orientation,
    children: [first, second],
    sizes: [50, 50],
  };
}
export function splitNode(
  root: PaneNode,
  targetId: string,
  newPaneId: string,
  orientation: SplitNode["orientation"],
  shell = "",
): PaneNode {
  const target = findNode(root, targetId)?.node;
  return target?.type === "terminal"
    ? replaceNode(
        root,
        targetId,
        split(
          target,
          createTerminalNode(newPaneId, shell, target.cwd),
          orientation,
        ),
      )
    : root;
}
export function moveNode(
  root: PaneNode,
  sourceId: string,
  targetId: string,
  position: Position,
): PaneNode {
  if (sourceId === targetId) return root;
  const source = findNode(root, sourceId)?.node,
    target = findNode(root, targetId)?.node;
  if (source?.type !== "terminal" || target?.type !== "terminal") return root;
  if (position === "swap") {
    const swap = (n: PaneNode): PaneNode =>
      n.id === sourceId
        ? target
        : n.id === targetId
          ? source
          : n.type === "split"
            ? { ...n, children: n.children.map(swap) }
            : n;
    return swap(root);
  }
  const remaining = removeNode(root, sourceId);
  if (!remaining) return root;
  const before = position === "left" || position === "top";
  return replaceNode(
    remaining,
    targetId,
    split(
      before ? source : target,
      before ? target : source,
      position === "left" || position === "right" ? "vertical" : "horizontal",
    ),
  );
}
