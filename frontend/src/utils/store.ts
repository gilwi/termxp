import { reactive } from "vue";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import {
    PaneNode,
    createTerminalNode,
    findNode,
    removeNode,
    splitNode,
    moveNode,
} from "./layout";
import {
    WriteToTerminal,
    KillSession,
} from "../../wailsjs/go/main/TerminalService";
import { EventsOff } from "../../wailsjs/runtime/runtime";

export const themes: Record<string, { name: string; cssClass: string; xterm: any }> = {
    glassmorphic: {
        name: "Glassmorphic",
        cssClass: "theme-glassmorphic",
        xterm: {
            background: "rgba(14, 10, 30, 0.4)",
            foreground: "#f1f5f9",
            cursor: "#9333ea",
            selectionBackground: "rgba(255, 255, 255, 0.1)",
            black: "#0f0a1e",
            red: "#f43f5e",
            green: "#10b981",
            yellow: "#fbbf24",
            blue: "#3b82f6",
            magenta: "#a855f7",
            cyan: "#06b6d4",
            white: "#f1f5f9",
        },
    },
    cyberpunk: {
        name: "Cyberpunk",
        cssClass: "theme-cyberpunk",
        xterm: {
            background: "#040408",
            foreground: "#00ffcc",
            cursor: "#ff007f",
            selectionBackground: "rgba(255, 0, 127, 0.3)",
            black: "#07070e",
            red: "#ff0055",
            green: "#00ffcc",
            yellow: "#ffe600",
            blue: "#0066ff",
            magenta: "#ff00ff",
            cyan: "#00ffff",
            white: "#ffffff",
        },
    },
    dracula: {
        name: "Dracula",
        cssClass: "theme-dracula",
        xterm: {
            background: "#282a36",
            foreground: "#f8f8f2",
            cursor: "#ff79c6",
            selectionBackground: "rgba(255, 255, 255, 0.1)",
            black: "#1e1f29",
            red: "#ff5555",
            green: "#50fa7b",
            yellow: "#f1fa8c",
            blue: "#bd93f9",
            magenta: "#ff79c6",
            cyan: "#8be9fd",
            white: "#f8f8f2",
        },
    },
    matrix: {
        name: "Matrix",
        cssClass: "theme-matrix",
        xterm: {
            background: "#000000",
            foreground: "#00ff00",
            cursor: "#00ff00",
            selectionBackground: "rgba(0, 255, 0, 0.25)",
            black: "#000000",
            red: "#005500",
            green: "#00ff00",
            yellow: "#33cc33",
            blue: "#009900",
            magenta: "#00ff00",
            cyan: "#00ff00",
            white: "#55ff55",
        },
    },
    monokai: {
        name: "Monokai",
        cssClass: "theme-monokai",
        xterm: {
            background: "#272822",
            foreground: "#f8f8f2",
            cursor: "#f92672",
            selectionBackground: "rgba(255, 255, 255, 0.1)",
            black: "#1e1e1e",
            red: "#f92672",
            green: "#a6e22e",
            yellow: "#f4bf75",
            blue: "#66d9ef",
            magenta: "#ae81ff",
            cyan: "#a1efe4",
            white: "#f8f8f2",
        },
    },
};

export interface Tab {
    id: string;
    name: string;
    rootNode: PaneNode;
}

export interface CachedTerminal {
    term: Terminal;
    fitAddon: FitAddon;
    containerWrapper: HTMLDivElement;
    initialized: boolean;
    sId: string;
}

// Global cache for xterm.js instances managed inside the store module
export const instanceCache = new Map<string, CachedTerminal>();

export function disposeCachedTerminal(paneId: string) {
    const cached = instanceCache.get(paneId);
    if (cached) {
        if (cached.sId) {
            EventsOff(`terminal:data:${cached.sId}`);
        }
        cached.term.dispose();
        instanceCache.delete(paneId);
    }
}

// Custom simple UUID generator for client-side session IDs
export function generateUUID(): string {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function (c) {
        const r = (Math.random() * 16) | 0;
        const v = c === 'x' ? r : (r & 0x3) | 0x8;
        return v.toString(16);
    });
}

export const store = reactive({
    tabs: [] as Tab[],
    activeTabId: "",
    activePaneId: "",
    maximizedPaneId: null as string | null,
    currentTheme: "glassmorphic",
    fontSize: 14,
    sidebarOpen: false,
    editingTabId: null as string | null,
    editingName: "",

    stats: {
        cpu: 0,
        memory: 0,
        memoryRaw: "0 MB",
        uptime: "0m",
    },

    // Getters
    getActiveTab(): Tab | null {
        return this.tabs.find((t) => t.id === this.activeTabId) || null;
    },

    getActiveSessionId(): string | null {
        const tab = this.getActiveTab();
        if (!tab || !this.activePaneId) return null;
        const found = findNode(tab.rootNode, this.activePaneId);
        return found?.node.sessionId || null;
    },

    // Actions
    addTab() {
        const id = Date.now().toString();
        const rootPaneId = `pane-${Date.now()}`;
        const index = this.tabs.length + 1;
        const newTab: Tab = {
            id,
            name: `Shell ${index}`,
            rootNode: createTerminalNode(rootPaneId, ""),
        };
        this.tabs.push(newTab);
        this.activeTabId = id;
        this.activePaneId = rootPaneId;
    },

    selectTab(id: string) {
        this.activeTabId = id;
        const tab = this.tabs.find((t) => t.id === id);
        if (tab) {
            const firstTerm = getFirstTerminalNode(tab.rootNode);
            if (firstTerm) {
                this.activePaneId = firstTerm.id;
            }
        }
    },

    closeTab(id: string) {
        const index = this.tabs.findIndex((t) => t.id === id);
        if (index === -1) return;

        const tabToClose = this.tabs[index];
        this.killSessionsInNode(tabToClose.rootNode);

        if (this.maximizedPaneId && findNode(tabToClose.rootNode, this.maximizedPaneId)) {
            this.maximizedPaneId = null;
        }

        this.tabs.splice(index, 1);

        if (this.activeTabId === id) {
            if (this.tabs.length > 0) {
                const nextActiveIndex = Math.min(index, this.tabs.length - 1);
                this.selectTab(this.tabs[nextActiveIndex].id);
            } else {
                this.activeTabId = "";
                this.activePaneId = "";
            }
        }
    },

    splitPane(paneId: string, orientation: "horizontal" | "vertical") {
        const tab = this.getActiveTab();
        if (!tab) return;

        const newPaneId = `pane-${Date.now()}`;
        tab.rootNode = splitNode(tab.rootNode, paneId, newPaneId, orientation);
        this.activePaneId = newPaneId;
    },

    closePane(paneId: string) {
        const tab = this.getActiveTab();
        if (!tab) {
            for (const t of this.tabs) {
                const found = findNode(t.rootNode, paneId);
                if (found) {
                    this.performClosePane(t, paneId);
                    return;
                }
            }
            return;
        }
        this.performClosePane(tab, paneId);
    },

    performClosePane(tab: Tab, paneId: string) {
        if (this.maximizedPaneId === paneId) {
            this.maximizedPaneId = null;
        }

        if (tab.rootNode.type === "terminal" && tab.rootNode.id === paneId) {
            this.closeTab(tab.id);
            return;
        }

        const found = findNode(tab.rootNode, paneId);
        if (found && found.node.sessionId) {
            EventsOff(`terminal:exit:${found.node.sessionId}`);
            KillSession(found.node.sessionId).catch((err) => {
                console.error("Failed to kill session:", err);
            });
        }

        disposeCachedTerminal(paneId);

        const updatedRoot = removeNode(tab.rootNode, paneId);
        if (updatedRoot) {
            if (countTerminals(updatedRoot) === 0) {
                this.closeTab(tab.id);
                return;
            }
            tab.rootNode = updatedRoot;

            if (tab.id === this.activeTabId) {
                const firstTerm = getFirstTerminalNode(updatedRoot);
                if (firstTerm) {
                    this.activePaneId = firstTerm.id;
                }
            }
        }
    },

    paneInitialized(paneId: string, sessionId: string) {
        for (const t of this.tabs) {
            const found = findNode(t.rootNode, paneId);
            if (found) {
                found.node.sessionId = sessionId;
                break;
            }
        }
    },

    updateSizes(nodeId: string, newSizes: number[]) {
        const tab = this.getActiveTab();
        if (!tab) return;
        const found = findNode(tab.rootNode, nodeId);
        if (found) {
            found.node.sizes = newSizes;
        }
    },

    toggleMaximize(paneId?: string) {
        if (this.maximizedPaneId) {
            this.maximizedPaneId = null;
        } else {
            const targetId = paneId || this.activePaneId;
            if (targetId) {
                this.maximizedPaneId = targetId;
            }
        }
    },

    movePane(sourceId: string, targetId: string, position: "left" | "right" | "top" | "bottom" | "swap") {
        const tab = this.getActiveTab();
        if (!tab) return;

        const updatedRoot = moveNode(tab.rootNode, sourceId, targetId, position);
        if (updatedRoot) {
            tab.rootNode = updatedRoot;
            this.activePaneId = sourceId;
        }
    },

    runSnippet(cmd: string) {
        const sessionId = this.getActiveSessionId();
        if (sessionId) {
            WriteToTerminal(sessionId, cmd + "\n").catch((err) => {
                console.error("Failed to run snippet:", err);
            });
        }
    },

    startRenameTab(tab: Tab) {
        this.editingTabId = tab.id;
        this.editingName = tab.name;
    },

    saveRenameTab(tab: Tab) {
        if (this.editingName.trim()) {
            tab.name = this.editingName.trim();
        }
        this.editingTabId = null;
    },

    cancelRenameTab() {
        this.editingTabId = null;
    },

    // Session helpers
    killSessionsInNode(node: PaneNode) {
        if (node.type === "terminal") {
            if (node.sessionId) {
                EventsOff(`terminal:exit:${node.sessionId}`);
                KillSession(node.sessionId).catch((err) => {
                    console.error("Failed to kill session:", err);
                });
            }
            disposeCachedTerminal(node.id);
        } else if (node.type === "split" && node.children) {
            for (const child of node.children) {
                this.killSessionsInNode(child);
            }
        }
    }
});

// Traversal helper functions
function getFirstTerminalNode(node: PaneNode): PaneNode | null {
    if (node.type === "terminal") return node;
    if (node.type === "split" && node.children && node.children.length > 0) {
        return getFirstTerminalNode(node.children[0]);
    }
    return null;
}

function countTerminals(node: PaneNode): number {
    if (node.type === "terminal") return 1;
    if (node.type === "split" && node.children) {
        return node.children.reduce(
            (sum, child) => sum + countTerminals(child),
            0,
        );
    }
    return 0;
}
