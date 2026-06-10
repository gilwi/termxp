<script lang="ts" setup>
import { ref, onMounted, onBeforeUnmount, computed } from "vue";
import TerminalLayout from "./components/TerminalLayout.vue";
import { GetSystemStats } from "../wailsjs/go/main/App";
import { WriteToTerminal } from "../wailsjs/go/main/TerminalService";

import {
    PaneNode,
    createTerminalNode,
    findNode,
    removeNode,
    splitNode,
    moveNode,
} from "./utils/layout";

interface Tab {
    id: string;
    name: string;
    rootNode: PaneNode;
}

const themes: Record<string, { name: string; cssClass: string; xterm: any }> = {
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

// Application reactive states
const tabs = ref<Tab[]>([]);
const activeTabId = ref<string>("");
const activePaneId = ref<string>("");
const currentTheme = ref<string>("glassmorphic");
const fontSize = ref<number>(14);
const sidebarOpen = ref<boolean>(true);
const editingTabId = ref<string | null>(null);
const editingName = ref<string>("");
const renameInputRef = ref<HTMLInputElement | null>(null);

// Stats metrics
const stats = ref({
    cpu: 0,
    memory: 0,
    uptime: "0m",
});

// Command snippets list
const snippets = [
    { label: "List Files", cmd: "ls -lah" },
    { label: "System Info", cmd: "uname -a" },
    { label: "Disk Space", cmd: "df -h" },
    { label: "Active Network", cmd: "ss -tulpn" },
    { label: "CPU Load Check", cmd: "cat /proc/loadavg" },
    { label: "Who Am I", cmd: "whoami && pwd" },
];

// Computed active tab object
const activeTab = computed(() => {
    return tabs.value.find((t) => t.id === activeTabId.value) || null;
});

// Traverses layout tree to find the first terminal node (leaf)
function getFirstTerminalNode(node: PaneNode): PaneNode | null {
    if (node.type === "terminal") return node;
    if (node.type === "split" && node.children && node.children.length > 0) {
        return getFirstTerminalNode(node.children[0]);
    }
    return null;
}

// Add a new tab/session
function addTab() {
    const id = Date.now().toString();
    const rootPaneId = `pane-${Date.now()}`;
    const index = tabs.value.length + 1;
    const newTab: Tab = {
        id,
        name: `Shell ${index}`,
        rootNode: createTerminalNode(rootPaneId, ""),
    };
    tabs.value.push(newTab);
    activeTabId.value = id;
    activePaneId.value = rootPaneId;
}

// Select specific tab
function selectTab(id: string) {
    activeTabId.value = id;
    const tab = tabs.value.find((t) => t.id === id);
    if (tab) {
        const firstTerm = getFirstTerminalNode(tab.rootNode);
        if (firstTerm) {
            activePaneId.value = firstTerm.id;
        }
    }
}

// Close active tab
function closeTab(id: string) {
    const index = tabs.value.findIndex((t) => t.id === id);
    if (index === -1) return;

    tabs.value.splice(index, 1);

    if (activeTabId.value === id) {
        if (tabs.value.length > 0) {
            const nextActiveIndex = Math.min(index, tabs.value.length - 1);
            selectTab(tabs.value[nextActiveIndex].id);
        } else {
            activeTabId.value = "";
            activePaneId.value = "";
        }
    }
}

// Splits the target pane inside the current tab
function handleSplitPane(
    paneId: string,
    orientation: "horizontal" | "vertical",
) {
    const tab = activeTab.value;
    if (!tab) return;

    const newPaneId = `pane-${Date.now()}`;
    tab.rootNode = splitNode(tab.rootNode, paneId, newPaneId, orientation);
    activePaneId.value = newPaneId;
}

// Closes a terminal pane inside the current tab
function handleClosePane(paneId: string) {
    const tab = activeTab.value;
    if (!tab) return;

    // If this is the absolute only pane left, close the entire tab
    if (tab.rootNode.type === "terminal" && tab.rootNode.id === paneId) {
        closeTab(tab.id);
        return;
    }

    const updatedRoot = removeNode(tab.rootNode, paneId);
    if (updatedRoot) {
        tab.rootNode = updatedRoot;
        // Set focus to the first terminal we can find
        const firstTerm = getFirstTerminalNode(updatedRoot);
        if (firstTerm) {
            activePaneId.value = firstTerm.id;
        }
    }
}

// Backend terminal PTY initialization callback
function handlePaneInitialized(paneId: string, sessionId: string) {
    // Seek across all tabs for safety
    for (const t of tabs.value) {
        const found = findNode(t.rootNode, paneId);
        if (found) {
            found.node.sessionId = sessionId;
            break;
        }
    }
}

// Resizes a split node's children ratios
function handleUpdateSizes(nodeId: string, newSizes: number[]) {
    const tab = activeTab.value;
    if (!tab) return;
    const found = findNode(tab.rootNode, nodeId);
    if (found) {
        found.node.sizes = newSizes;
    }
}

// Rearrange terminal panes via Drag & Drop
function handleMovePane(
    sourceId: string,
    targetId: string,
    position: "left" | "right" | "top" | "bottom" | "swap",
) {
    const tab = activeTab.value;
    if (!tab) return;

    const updatedRoot = moveNode(tab.rootNode, sourceId, targetId, position);
    if (updatedRoot) {
        tab.rootNode = updatedRoot;
        activePaneId.value = sourceId; // Refocus the dragged pane
    }
}

// Seeks the active pane's session ID to route snippet commands
const activeSessionId = computed(() => {
    const tab = activeTab.value;
    if (!tab || !activePaneId.value) return null;
    const found = findNode(tab.rootNode, activePaneId.value);
    return found?.node.sessionId || null;
});

// Run predefined snippet command
function runSnippet(cmd: string) {
    const sessionId = activeSessionId.value;
    if (sessionId) {
        WriteToTerminal(sessionId, cmd + "\n").catch((err) => {
            console.error("Failed to run snippet:", err);
        });
    }
}

// Edit tab naming
function startRenameTab(tab: Tab) {
    editingTabId.value = tab.id;
    editingName.value = tab.name;
    setTimeout(() => {
        renameInputRef.value?.focus();
        renameInputRef.value?.select();
    }, 50);
}

function saveRenameTab(tab: Tab) {
    if (editingName.value.trim()) {
        tab.name = editingName.value.trim();
    }
    editingTabId.value = null;
}

function cancelRenameTab() {
    editingTabId.value = null;
}

// System stats poller
let statsInterval: number | null = null;
async function fetchStats() {
    try {
        const data = await GetSystemStats();
        if (data) {
            stats.value.cpu = Math.round(data.cpu);
            stats.value.memory = Math.round(data.memory);
            stats.value.uptime = data.uptime;
        }
    } catch (err) {
        console.error("Failed to poll system stats:", err);
    }
}

onMounted(() => {
    addTab();
    fetchStats();
    statsInterval = window.setInterval(fetchStats, 2500);
});

onBeforeUnmount(() => {
    if (statsInterval) {
        clearInterval(statsInterval);
    }
});
</script>

<template>
    <div :class="['app-container', themes[currentTheme].cssClass]">
        <!-- Background glow design for glassmorphism -->
        <div class="bg-glow"></div>

        <!-- Sidebar Layout -->
        <aside :class="['sidebar', { 'sidebar-closed': !sidebarOpen }]">
            <div class="sidebar-header">
                <div class="logo-area">
                    <span class="logo-icon">🚀</span>
                    <h2>TermXP</h2>
                    <span class="badge">PRO</span>
                </div>
            </div>

            <div class="sidebar-content custom-scrollbar">
                <!-- System Status metrics -->
                <section class="section">
                    <h3>System Metrics</h3>
                    <div class="metrics-grid">
                        <div class="metric-card">
                            <div class="metric-info">
                                <span>CPU Load</span>
                                <span class="metric-value"
                                    >{{ stats.cpu }}%</span
                                >
                            </div>
                            <div class="progress-bar-container">
                                <div
                                    class="progress-bar cpu-bar"
                                    :style="{ width: stats.cpu + '%' }"
                                ></div>
                            </div>
                        </div>

                        <div class="metric-card">
                            <div class="metric-info">
                                <span>RAM Usage</span>
                                <span class="metric-value"
                                    >{{ stats.memory }}%</span
                                >
                            </div>
                            <div class="progress-bar-container">
                                <div
                                    class="progress-bar memory-bar"
                                    :style="{ width: stats.memory + '%' }"
                                ></div>
                            </div>
                        </div>

                        <div class="metric-card single-metric">
                            <span class="metric-label">Uptime:</span>
                            <span class="metric-text font-mono">{{
                                stats.uptime
                            }}</span>
                        </div>
                    </div>
                </section>

                <!-- Predefined Themes -->
                <section class="section">
                    <h3>Visual Themes</h3>
                    <div class="themes-grid">
                        <button
                            v-for="(themeConfig, themeKey) in themes"
                            :key="themeKey"
                            :class="[
                                'theme-btn',
                                { active: currentTheme === themeKey },
                            ]"
                            @click="currentTheme = themeKey"
                        >
                            <span
                                class="theme-color-dot"
                                :style="{
                                    background: themeConfig.xterm.background,
                                }"
                            ></span>
                            {{ themeConfig.name }}
                        </button>
                    </div>
                </section>

                <!-- FontSize control -->
                <section class="section">
                    <h3>Font Control</h3>
                    <div class="font-controls">
                        <button
                            @click="fontSize = Math.max(10, fontSize - 1)"
                            class="font-btn"
                        >
                            <svg
                                width="16"
                                height="16"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2"
                            >
                                <line x1="5" y1="12" x2="19" y2="12"></line>
                            </svg>
                        </button>
                        <span class="font-display">{{ fontSize }}px</span>
                        <button
                            @click="fontSize = Math.min(24, fontSize + 1)"
                            class="font-btn"
                        >
                            <svg
                                width="16"
                                height="16"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2"
                            >
                                <line x1="12" y1="5" x2="12" y2="19"></line>
                                <line x1="5" y1="12" x2="19" y2="12"></line>
                            </svg>
                        </button>
                    </div>
                </section>

                <!-- Quick actions / commands -->
                <section class="section">
                    <h3>Quick Actions</h3>
                    <div class="snippets-list">
                        <button
                            v-for="s in snippets"
                            :key="s.label"
                            class="snippet-btn"
                            @click="runSnippet(s.cmd)"
                            :disabled="!activeSessionId"
                            :title="s.cmd"
                        >
                            <span class="cmd-symbol">$</span>
                            <span class="cmd-label">{{ s.label }}</span>
                        </button>
                    </div>
                </section>
            </div>
        </aside>

        <!-- Main Workspace -->
        <main class="workspace">
            <!-- Top header bar -->
            <header class="top-header">
                <div class="tabs-scroll-container custom-scrollbar">
                    <!-- Toggle sidebar button -->
                    <button
                        class="icon-toggle-btn"
                        @click="sidebarOpen = !sidebarOpen"
                        title="Toggle Sidebar"
                    >
                        <svg
                            width="18"
                            height="18"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="2"
                        >
                            <line x1="3" y1="12" x2="21" y2="12"></line>
                            <line x1="3" y1="6" x2="21" y2="6"></line>
                            <line x1="3" y1="18" x2="21" y2="18"></line>
                        </svg>
                    </button>

                    <!-- Tab list -->
                    <div class="tabs-list">
                        <div
                            v-for="tab in tabs"
                            :key="tab.id"
                            :class="[
                                'tab-item',
                                { active: activeTabId === tab.id },
                            ]"
                            @click="selectTab(tab.id)"
                        >
                            <!-- Editing name input -->
                            <input
                                v-if="editingTabId === tab.id"
                                ref="renameInputRef"
                                type="text"
                                v-model="editingName"
                                @blur="saveRenameTab(tab)"
                                @keydown.enter="saveRenameTab(tab)"
                                @keydown.esc="cancelRenameTab"
                                class="tab-rename-input"
                            />
                            <!-- Default text label -->
                            <span
                                v-else
                                class="tab-label"
                                @dblclick="startRenameTab(tab)"
                                title="Double click to rename"
                            >
                                {{ tab.name }}
                            </span>

                            <!-- Close button -->
                            <button
                                class="tab-close-btn"
                                @click.stop="closeTab(tab.id)"
                            >
                                <svg
                                    width="12"
                                    height="12"
                                    viewBox="0 0 24 24"
                                    fill="none"
                                    stroke="currentColor"
                                    stroke-width="2.5"
                                >
                                    <line x1="18" y1="6" x2="6" y2="18"></line>
                                    <line x1="6" y1="6" x2="18" y2="18"></line>
                                </svg>
                            </button>
                        </div>

                        <!-- Add Tab Button -->
                        <button
                            class="add-tab-btn"
                            @click="addTab"
                            title="Open new shell session"
                        >
                            <svg
                                width="14"
                                height="14"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2.5"
                            >
                                <line x1="12" y1="5" x2="12" y2="19"></line>
                                <line x1="5" y1="12" x2="19" y2="12"></line>
                            </svg>
                            <span>New Shell</span>
                        </button>
                    </div>
                </div>
            </header>

            <!-- Terminal content pane -->
            <div class="terminal-workspace-container">
                <!-- Render recursive split layouts for each tab. v-show keeps running terminals alive. -->
                <TerminalLayout
                    v-for="tab in tabs"
                    :key="tab.id"
                    v-show="activeTabId === tab.id"
                    :node="tab.rootNode"
                    :active-pane-id="activePaneId"
                    :theme="themes[currentTheme].xterm"
                    :theme-class="themes[currentTheme].cssClass"
                    :font-size="fontSize"
                    @split-pane="handleSplitPane"
                    @close-pane="handleClosePane"
                    @pane-initialized="handlePaneInitialized"
                    @focus-pane="(pId) => (activePaneId = pId)"
                    @move-pane="handleMovePane"
                    @update-sizes="handleUpdateSizes"
                />

                <!-- Empty state layout -->
                <div v-if="tabs.length === 0" class="empty-state">
                    <div class="empty-glow"></div>
                    <div class="empty-info">
                        <span class="empty-icon">📟</span>
                        <h1>No Active Shell Sessions</h1>
                        <p>
                            Spawns local terminal processes (e.g. bash) on your
                            host machine to run CLI actions.
                        </p>
                        <button @click="addTab" class="action-btn">
                            <span>Start New Shell</span>
                        </button>
                    </div>
                </div>
            </div>
        </main>
    </div>
</template>

<style>
/* Reset and core layout */
.app-container {
    display: flex;
    width: 100vw;
    height: 100vh;
    overflow: hidden;
    position: relative;
    background: var(--bg-gradient);
    color: var(--text-main);
    transition:
        background var(--transition-speed),
        color var(--transition-speed);
}

.bg-glow {
    position: absolute;
    top: -20%;
    left: -20%;
    width: 60%;
    height: 60%;
    background: radial-gradient(
        circle,
        var(--active-accent-glow) 0%,
        transparent 70%
    );
    z-index: 0;
    pointer-events: none;
    filter: blur(80px);
}

/* Monospace font variables utility */
.font-mono {
    font-family: var(--font-mono);
}

/* Sidebar Styling */
.sidebar {
    width: 250px;
    height: 100%;
    background: var(--sidebar-bg);
    backdrop-filter: var(--backdrop-blur);
    -webkit-backdrop-filter: var(--backdrop-blur);
    border-right: 1px solid var(--border-color);
    display: flex;
    flex-direction: column;
    z-index: 10;
    transition:
        width var(--transition-speed),
        transform var(--transition-speed);
    flex-shrink: 0;
}

.sidebar-closed {
    width: 0px;
    transform: translateX(-250px);
    border-right: none;
}

.sidebar-header {
    padding: 16px;
    border-bottom: 1px solid var(--border-color);
}

.logo-area {
    display: flex;
    align-items: center;
    gap: 8px;
}

.logo-icon {
    font-size: 20px;
}

.logo-area h2 {
    margin: 0;
    font-size: 20px;
    font-weight: 700;
    letter-spacing: -0.5px;
}

.badge {
    font-size: 9px;
    font-weight: 700;
    padding: 2px 5px;
    border-radius: 4px;
    background: var(--active-accent);
    color: #ffffff;
    box-shadow: 0 0 8px var(--active-accent-glow);
}

.sidebar-content {
    flex: 1;
    padding: 16px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 24px;
}

.section h3 {
    margin: 0 0 12px 0;
    font-size: 11px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 1px;
    color: var(--text-muted);
}

/* System metrics UI */
.metrics-grid {
    display: flex;
    flex-direction: column;
    gap: 12px;
}

.metric-card {
    background: var(--card-bg);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    padding: 10px 12px;
}

.metric-info {
    display: flex;
    justify-content: space-between;
    font-size: 12px;
    margin-bottom: 6px;
    color: var(--text-main);
    font-weight: 500;
}

.metric-value {
    font-family: var(--font-mono);
}

.progress-bar-container {
    width: 100%;
    height: 5px;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 10px;
    overflow: hidden;
}

.progress-bar {
    height: 100%;
    border-radius: 10px;
    width: 0;
    transition: width 0.8s cubic-bezier(0.1, 0.8, 0.2, 1);
}

.cpu-bar {
    background: var(--active-accent);
    box-shadow: 0 0 6px var(--active-accent-glow);
}

.memory-bar {
    background: #3b82f6;
    box-shadow: 0 0 6px rgba(59, 130, 246, 0.5);
}

.single-metric {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 12px;
}

.metric-label {
    color: var(--text-muted);
}

.metric-text {
    font-weight: 600;
    color: var(--text-main);
}

/* Visual Themes Selector */
.themes-grid {
    display: grid;
    grid-template-columns: 1fr;
    gap: 6px;
}

.theme-btn {
    background: var(--card-bg);
    border: 1px solid var(--border-color);
    color: var(--text-main);
    padding: 8px 12px;
    border-radius: 6px;
    font-size: 13px;
    font-weight: 500;
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    text-align: left;
    transition:
        background 0.2s,
        border-color 0.2s,
        box-shadow 0.2s;
}

.theme-btn:hover {
    background: var(--card-hover);
    border-color: var(--active-accent);
}

.theme-btn.active {
    background: var(--card-hover);
    border-color: var(--active-accent);
    box-shadow: 0 0 10px var(--active-accent-glow);
    font-weight: 600;
}

.theme-color-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    border: 1px solid rgba(255, 255, 255, 0.2);
}

/* Font controls styling */
.font-controls {
    display: flex;
    align-items: center;
    justify-content: space-between;
    background: var(--card-bg);
    border: 1px solid var(--border-color);
    padding: 6px;
    border-radius: 6px;
}

.font-btn {
    background: transparent;
    border: none;
    color: var(--text-main);
    cursor: pointer;
    width: 28px;
    height: 28px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 4px;
    transition: background 0.2s;
}

.font-btn:hover {
    background: var(--card-hover);
    color: var(--active-accent);
}

.font-display {
    font-size: 13px;
    font-family: var(--font-mono);
    font-weight: 600;
}

/* Snippets panel */
.snippets-list {
    display: grid;
    grid-template-columns: 1fr;
    gap: 6px;
}

.snippet-btn {
    background: var(--card-bg);
    border: 1px solid var(--border-color);
    color: var(--text-main);
    padding: 8px 12px;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 500;
    display: flex;
    align-items: center;
    gap: 6px;
    cursor: pointer;
    text-align: left;
    transition: all 0.2s;
}

.snippet-btn:hover:not(:disabled) {
    background: var(--card-hover);
    border-color: var(--active-accent);
    transform: translateX(2px);
}

.snippet-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
}

.cmd-symbol {
    color: var(--active-accent);
    font-family: var(--font-mono);
    font-weight: 700;
}

/* Workspace workspace */
.workspace {
    flex: 1;
    height: 100%;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    z-index: 1;
}

/* Header bar */
.top-header {
    height: 48px;
    background: var(--header-bg);
    border-bottom: 1px solid var(--border-color);
    display: flex;
    align-items: center;
    padding: 0 12px;
    flex-shrink: 0;
}

.tabs-scroll-container {
    display: flex;
    align-items: center;
    width: 100%;
    height: 100%;
    overflow-x: auto;
    overflow-y: hidden;
}

.icon-toggle-btn {
    background: transparent;
    border: none;
    color: var(--text-main);
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    border-radius: 4px;
    margin-right: 8px;
    transition:
        background 0.2s,
        color 0.2s;
    flex-shrink: 0;
}

.icon-toggle-btn:hover {
    background: var(--card-hover);
    color: var(--active-accent);
}

.tabs-list {
    display: flex;
    align-items: center;
    gap: 6px;
    height: 100%;
}

.tab-item {
    height: 32px;
    padding: 0 12px;
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid var(--border-color);
    border-radius: 6px;
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    font-size: 13px;
    font-weight: 500;
    transition: all 0.2s;
    user-select: none;
    max-width: 140px;
    flex-shrink: 0;
}

.tab-item:hover {
    background: var(--card-hover);
    border-color: var(--active-accent);
}

.tab-item.active {
    background: var(--card-hover);
    border-color: var(--active-accent);
    box-shadow: 0 0 8px var(--active-accent-glow);
    font-weight: 600;
}

.tab-label {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

.tab-rename-input {
    background: transparent;
    border: 1px solid var(--active-accent);
    color: var(--text-main);
    outline: none;
    font-size: 12px;
    padding: 1px 4px;
    border-radius: 3px;
    width: 80px;
}

.tab-close-btn {
    background: transparent;
    border: none;
    color: var(--text-muted);
    width: 16px;
    height: 16px;
    border-radius: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    padding: 0;
    transition:
        background 0.2s,
        color 0.2s;
}

.tab-close-btn:hover {
    background: rgba(255, 255, 255, 0.1);
    color: var(--text-main);
}

.add-tab-btn {
    height: 32px;
    padding: 0 10px;
    background: transparent;
    border: 1px dashed var(--border-color);
    color: var(--text-muted);
    border-radius: 6px;
    display: flex;
    align-items: center;
    gap: 4px;
    cursor: pointer;
    font-size: 12px;
    font-weight: 500;
    transition: all 0.2s;
    flex-shrink: 0;
}

.add-tab-btn:hover {
    border-color: var(--active-accent);
    color: var(--active-accent);
    background: var(--card-hover);
}

/* Terminal wrapper pane */
.terminal-workspace-container {
    flex: 1;
    position: relative;
    overflow: hidden;
    background: rgba(0, 0, 0, 0.15);
}

/* Empty state styling */
.empty-state {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    box-sizing: border-box;
    padding: 20px;
}

.empty-glow {
    position: absolute;
    width: 250px;
    height: 250px;
    background: radial-gradient(
        circle,
        var(--active-accent-glow) 0%,
        transparent 70%
    );
    filter: blur(50px);
    z-index: 0;
    pointer-events: none;
}

.empty-info {
    z-index: 1;
    text-align: center;
    max-width: 420px;
}

.empty-icon {
    font-size: 64px;
    display: block;
    margin-bottom: 16px;
    filter: drop-shadow(0 0 12px var(--active-accent-glow));
}

.empty-info h1 {
    margin: 0 0 10px 0;
    font-size: 26px;
    font-weight: 700;
    letter-spacing: -0.5px;
}

.empty-info p {
    margin: 0 0 24px 0;
    font-size: 14px;
    line-height: 1.6;
    color: var(--text-muted);
}

.action-btn {
    background: var(--active-accent);
    color: #ffffff;
    border: none;
    padding: 10px 20px;
    font-size: 14px;
    font-weight: 600;
    border-radius: 8px;
    cursor: pointer;
    transition:
        background 0.2s,
        box-shadow 0.2s;
    box-shadow: 0 4px 12px var(--active-accent-glow);
}

.action-btn:hover {
    background: var(--active-accent-hover);
    box-shadow: 0 6px 16px var(--active-accent-glow);
}
</style>
