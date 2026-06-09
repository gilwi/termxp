export interface PaneNode {
  id: string;
  type: 'terminal' | 'split';
  orientation?: 'horizontal' | 'vertical';
  children?: PaneNode[];
  sizes?: number[];
  sessionId?: string;
}

export function createTerminalNode(id: string, sessionId: string = ''): PaneNode {
  return {
    id,
    type: 'terminal',
    sessionId,
  };
}

/**
 * Searches for a node inside a PaneNode tree by ID.
 * Returns the node, its parent, and its index in the parent's children.
 */
export function findNode(
  root: PaneNode,
  id: string,
  parent: PaneNode | null = null,
  index: number = -1
): { parent: PaneNode | null; node: PaneNode; index: number } | null {
  if (root.id === id) {
    return { parent, node: root, index };
  }
  if (root.type === 'split' && root.children) {
    for (let i = 0; i < root.children.length; i++) {
      const found = findNode(root.children[i], id, root, i);
      if (found) return found;
    }
  }
  return null;
}

/**
 * Removes a terminal or split pane by ID from the layout tree.
 * Collapses any parent split node if it only has 1 remaining child.
 * Returns the new root node.
 */
export function removeNode(root: PaneNode, id: string): PaneNode | null {
  if (root.id === id) {
    return null;
  }

  const found = findNode(root, id);
  if (!found || !found.parent || found.index === -1) {
    return root;
  }

  const parent = found.parent;
  parent.children?.splice(found.index, 1);
  parent.sizes?.splice(found.index, 1);

  // Distribute size percentages equally or proportionally
  if (parent.children && parent.children.length > 0 && parent.sizes) {
    const total = parent.sizes.reduce((sum, s) => sum + s, 0);
    if (total > 0) {
      parent.sizes = parent.sizes.map((s) => (s / total) * 100);
    } else {
      parent.sizes = parent.children.map(() => 100 / parent.children!.length);
    }
  }

  // Collapse parent split if it is down to a single child
  if (parent.children && parent.children.length === 1) {
    const singleChild = parent.children[0];
    const grandparentInfo = findNode(root, parent.id);
    if (grandparentInfo && grandparentInfo.parent && grandparentInfo.index !== -1) {
      grandparentInfo.parent.children![grandparentInfo.index] = singleChild;
    } else {
      // Parent was the root node of the tree
      return singleChild;
    }
  }

  return root;
}

/**
 * Splits a terminal pane into two panes (the original and a new blank one).
 * Returns the root of the tree.
 */
export function splitNode(
  root: PaneNode,
  targetId: string,
  newPaneId: string,
  orientation: 'horizontal' | 'vertical'
): PaneNode {
  const found = findNode(root, targetId);
  if (!found) return root;

  const targetNode = found.node;
  // Clone current node to put as child
  const originalCopy = { ...targetNode };
  const newNode = createTerminalNode(newPaneId, '');

  targetNode.type = 'split';
  targetNode.orientation = orientation;
  targetNode.children = [originalCopy, newNode];
  targetNode.sizes = [50, 50];
  delete targetNode.sessionId;

  return root;
}

/**
 * Moves a source terminal pane to a target terminal pane.
 * position can be: 'left', 'right', 'top', 'bottom', or 'swap'.
 * Returns the root of the tree.
 */
export function moveNode(
  root: PaneNode,
  sourceId: string,
  targetId: string,
  position: 'left' | 'right' | 'top' | 'bottom' | 'swap'
): PaneNode {
  if (sourceId === targetId) return root;

  const sourceInfo = findNode(root, sourceId);
  const targetInfo = findNode(root, targetId);
  if (!sourceInfo || !targetInfo) return root;

  if (position === 'swap') {
    // Swap the session ID and keep layout tree shape unchanged
    const tempSession = sourceInfo.node.sessionId;
    sourceInfo.node.sessionId = targetInfo.node.sessionId;
    targetInfo.node.sessionId = tempSession;
    return root;
  }

  // Keep a copy of the source pane before removal
  const sourceNodeCopy = JSON.parse(JSON.stringify(sourceInfo.node));

  // Extract source pane from the tree
  let newRoot = removeNode(root, sourceId);
  if (!newRoot) return root;

  // Search for the target pane in the now-clean tree
  const cleanTargetInfo = findNode(newRoot, targetId);
  if (!cleanTargetInfo) return newRoot;

  const targetNode = cleanTargetInfo.node;
  const targetNodeCopy = JSON.parse(JSON.stringify(targetNode));

  const isHorizontal = position === 'left' || position === 'right';
  const orientation = isHorizontal ? 'horizontal' : 'vertical';

  const children =
    position === 'left' || position === 'top'
      ? [sourceNodeCopy, targetNodeCopy]
      : [targetNodeCopy, sourceNodeCopy];

  // Convert target node into a split node
  targetNode.type = 'split';
  targetNode.orientation = orientation;
  targetNode.children = children;
  targetNode.sizes = [50, 50];
  delete targetNode.sessionId;

  return newRoot;
}
