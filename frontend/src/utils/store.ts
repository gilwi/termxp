import { reactive } from "vue";
import { defaultShortcuts, defaultSnippets } from "./domain";
import type { SSHProfile } from "./domain";
import {
  PaneNode,
  Position,
  TerminalNode,
  createTerminalNode,
  findNode,
  generateUUID,
  terminals,
  removeNode,
  splitNode,
  moveNode,
} from "./layout";
import { configureInputRouter, disposeSession, sessions } from "./sessions";
export { themes } from "./themes";
export interface Tab {
  id: string;
  name: string;
  rootNode: PaneNode;
  focusedPaneId?: string;
}
export const store = reactive({
  tabs: [] as Tab[],
  activeTabId: "",
  activePaneId: "",
  maximizedPaneId: null as string | null,
  currentTheme: "glassmorphic",
  fontSize: 14,
  defaultShell: "",
  shells: [] as string[],
  copyOnSelect: false,
  restoreWorkspace: true,
  recoveryRequired: false,
  snippets: defaultSnippets.map((s) => ({ ...s })),
  shortcuts: { ...defaultShortcuts },
  sshProfiles: [] as SSHProfile[],
  sidebarOpen: false,
  editingTabId: null as string | null,
  editingName: "",
  broadcast: false,
  paletteOpen: false,
  notice: "",
  ready: false,
  stats: {
    cpu: null as number | null,
    memory: null as number | null,
    memoryRaw: "Unavailable",
    uptime: "0m",
  },
  getActiveTab(): Tab | null {
    return this.tabs.find((t) => t.id === this.activeTabId) || null;
  },
  getActiveNode(): TerminalNode | null {
    const tab = this.getActiveTab();
    const node = tab ? findNode(tab.rootNode, this.activePaneId)?.node : null;
    return node?.type === "terminal" ? node : null;
  },
  getActiveSessionId(): string | null {
    const controller = sessions.get(this.activePaneId)?.controller;
    return controller?.state.status === "running" ? controller.sessionId : null;
  },
  addTab(ssh?: SSHProfile) {
    if (
      this.tabs.reduce((count, t) => count + terminals(t.rootNode).length, 0) >=
      64
    ) {
      this.notice = "Workspace limit reached (64 terminals)";
      return;
    }
    const root = createTerminalNode(generateUUID(), this.defaultShell);
    if (ssh) {
      root.ssh = { ...ssh };
      root.sshProfileId = ssh.id;
    }
    const tab = {
      id: generateUUID(),
      name: ssh?.name || `Shell ${this.tabs.length + 1}`,
      rootNode: root,
    };
    this.tabs.push(tab);
    this.selectTab(tab.id);
  },
  selectTab(id: string) {
    const tab = this.tabs.find((t) => t.id === id);
    if (!tab) return;
    const previous = this.getActiveTab();
    if (previous) previous.focusedPaneId = this.activePaneId;
    this.activeTabId = id;
    this.activePaneId =
      (tab.focusedPaneId && findNode(tab.rootNode, tab.focusedPaneId)
        ? tab.focusedPaneId
        : terminals(tab.rootNode)[0]?.id) || "";
    if (this.maximizedPaneId && !findNode(tab.rootNode, this.maximizedPaneId))
      this.maximizedPaneId = null;
    this.broadcast = false;
  },
  closeTab(id: string) {
    const index = this.tabs.findIndex((t) => t.id === id);
    if (index < 0) return;
    terminals(this.tabs[index].rootNode).forEach((node) =>
      disposeSession(node.id),
    );
    this.tabs.splice(index, 1);
    this.broadcast = false;
    if (this.activeTabId === id) {
      const next = this.tabs[Math.min(index, this.tabs.length - 1)];
      if (next) this.selectTab(next.id);
      else {
        this.activeTabId = "";
        this.activePaneId = "";
        this.maximizedPaneId = null;
      }
    }
  },
  closePane(id: string) {
    const tab = this.tabs.find((t) => findNode(t.rootNode, id));
    if (!tab) return;
    if (tab.rootNode.id === id) {
      this.closeTab(tab.id);
      return;
    }
    disposeSession(id);
    const root = removeNode(tab.rootNode, id);
    if (!root) {
      this.closeTab(tab.id);
      return;
    }
    tab.rootNode = root;
    this.broadcast = false;
    if (this.maximizedPaneId === id) this.maximizedPaneId = null;
    if (this.activePaneId === id)
      this.activePaneId = terminals(root)[0]?.id || "";
  },
  splitPane(
    id: string,
    orientation: "horizontal" | "vertical",
    connection: "local" | "same" = "local",
  ) {
    const tab = this.tabs.find((t) => findNode(t.rootNode, id));
    if (!tab) return;
    if (
      this.tabs.reduce((count, t) => count + terminals(t.rootNode).length, 0) >=
      64
    ) {
      this.notice = "Workspace limit reached (64 terminals)";
      return;
    }
    const depth = (n: PaneNode, level = 0): number =>
      n.id === id
        ? level
        : n.type === "split"
          ? Math.max(-1, ...n.children.map((c) => depth(c, level + 1)))
          : -1;
    if (depth(tab.rootNode) >= 16) {
      this.notice = "Split nesting limit reached (16)";
      return;
    }
    const source = findNode(tab.rootNode, id)?.node;
    if (source?.type !== "terminal") return;
    const liveCwd = sessions.get(id)?.controller.state.cwd || source.cwd;
    const newId = generateUUID();
    tab.rootNode = splitNode(
      tab.rootNode,
      id,
      newId,
      orientation,
      connection === "same" ? source.shell : this.defaultShell,
    );
    const created = findNode(tab.rootNode, newId)?.node;
    if (created?.type === "terminal") {
      created.cwd = liveCwd;
      if (connection === "same" && source.ssh) {
        created.ssh = { ...source.ssh };
        created.sshProfileId = source.sshProfileId;
      }
    }
    this.activePaneId = newId;
    this.maximizedPaneId = null;
    this.broadcast = false;
  },
  movePane(source: string, target: string, position: Position) {
    const tab = this.getActiveTab();
    if (!tab) return;
    const root = moveNode(tab.rootNode, source, target, position);
    const depth = (node: PaneNode): number =>
      node.type === "terminal" ? 0 : 1 + Math.max(...node.children.map(depth));
    if (depth(root) > 16) {
      this.notice = "Split nesting limit reached (16)";
      return;
    }
    tab.rootNode = root;
    this.activePaneId = source;
  },
  updateSizes(id: string, sizes: number[]) {
    const tab = this.tabs.find((t) => findNode(t.rootNode, id));
    const node = tab && findNode(tab.rootNode, id)?.node;
    if (node?.type === "split") node.sizes = sizes;
  },
  toggleMaximize(id?: string) {
    this.maximizedPaneId = this.maximizedPaneId
      ? null
      : id || this.activePaneId;
  },
  runSnippet(command: string) {
    sessions.get(this.activePaneId)?.controller.send(command + "\n");
  },
  updateSSHProfile(profile: SSHProfile): () => void {
    const previous = this.sshProfiles;
    const changed: {
      node: TerminalNode;
      ssh: SSHProfile | undefined;
      id: string | undefined;
    }[] = [];
    this.sshProfiles = [
      ...previous.filter((p) => p.id !== profile.id),
      { ...profile },
    ];
    for (const tab of this.tabs)
      for (const node of terminals(tab.rootNode))
        if (node.sshProfileId === profile.id) {
          changed.push({ node, ssh: node.ssh, id: node.sshProfileId });
          node.ssh = { ...profile };
          sessions.get(node.id)?.controller.updateSSH(profile);
        }
    return () => {
      this.sshProfiles = previous;
      for (const item of changed) {
        item.node.ssh = item.ssh;
        item.node.sshProfileId = item.id;
        sessions.get(item.node.id)?.controller.updateSSH(item.ssh);
      }
    };
  },
  deleteSSHProfile(id: string): () => void {
    const previous = this.sshProfiles;
    const changed: TerminalNode[] = [];
    this.sshProfiles = previous.filter((p) => p.id !== id);
    for (const tab of this.tabs)
      for (const node of terminals(tab.rootNode))
        if (node.sshProfileId === id) {
          changed.push(node);
          node.sshProfileId = undefined;
        }
    return () => {
      this.sshProfiles = previous;
      for (const node of changed) node.sshProfileId = id;
    };
  },
  startRenameTab(tab: Tab) {
    this.editingTabId = tab.id;
    this.editingName = tab.name;
  },
  saveRenameTab(tab: Tab) {
    if (this.editingName.trim())
      tab.name = this.editingName.trim().slice(0, 256);
    this.editingTabId = null;
  },
  cancelRenameTab() {
    this.editingTabId = null;
  },
  tabDirectory(tab: Tab): string {
    const node =
      terminals(tab.rootNode).find((n) => n.id === this.activePaneId) ||
      terminals(tab.rootNode)[0];
    return node ? sessions.get(node.id)?.controller.state.cwd || node.cwd : "";
  },
});
configureInputRouter((source, data) => {
  const tab = store.getActiveTab();
  // Broadcast is limited to the selected tab, and only originates in that tab.
  const targets = tab
    ? terminals(tab.rootNode)
        .map((n) => sessions.get(n.id)?.controller)
        .filter((c) => c?.state.status === "running")
    : [];
  if (store.broadcast && targets.includes(source))
    targets.forEach((c) => c?.send(data));
  else source.send(data);
});
