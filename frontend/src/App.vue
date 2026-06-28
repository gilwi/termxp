<script lang="ts" setup>
import { ref, onMounted, onBeforeUnmount, computed, watch } from "vue";
import TerminalLayout from "./components/TerminalLayout.vue";
import CustomTitleBar from "./components/CustomTitleBar.vue";
import { GetSystemStats, LoadConfig, SaveConfig } from "../wailsjs/go/main/App";
import {
    EventsOn,
    WindowIsMaximised,
} from "../wailsjs/runtime/runtime";
import { store, themes } from "./utils/store";

const isMaximised = ref<boolean>(false);
const renameInputRef = ref<HTMLInputElement[]>([]);
const configLoaded = ref<boolean>(false);

// Command snippets list
const snippets = [
    { label: "List Files", cmd: "ls -lah" },
    { label: "System Info", cmd: "uname -a" },
    { label: "Disk Space", cmd: "df -h" },
    { label: "Active Network", cmd: "ss -tulpn" },
    { label: "CPU Load Check", cmd: "cat /proc/loadavg" },
    { label: "Who Am I", cmd: "whoami && pwd" },
];

// Edit tab naming helper to set focus
function startRenameTab(tab: any) {
    store.startRenameTab(tab);
    setTimeout(() => {
        if (renameInputRef.value && renameInputRef.value.length > 0) {
            renameInputRef.value[0].focus();
            renameInputRef.value[0].select();
        }
    }, 50);
}

// System stats poller
let statsInterval: number | null = null;

// Global Keyboard Shortcuts
function handleGlobalKeyDown(e: KeyboardEvent) {
    // Ctrl+Shift+T: New Tab
    if (e.ctrlKey && e.shiftKey && e.key.toLowerCase() === "t") {
        e.preventDefault();
        store.addTab();
    }
    // Ctrl+Shift+W: Close Current Tab
    else if (e.ctrlKey && e.shiftKey && e.key.toLowerCase() === "w") {
        if (store.activeTabId) {
            e.preventDefault();
            store.closeTab(store.activeTabId);
        }
    }
    // Ctrl+Shift+X: Toggle Maximize Pane
    else if (e.ctrlKey && e.shiftKey && e.key.toLowerCase() === "x") {
        e.preventDefault();
        store.toggleMaximize();
    }
    // Ctrl+Shift+E: Split Vertically (Right)
    else if (e.ctrlKey && e.shiftKey && e.key.toLowerCase() === "e") {
        if (store.activePaneId) {
            e.preventDefault();
            store.splitPane(store.activePaneId, "vertical");
        }
    }
    // Ctrl+Shift+O: Split Horizontally (Bottom)
    else if (e.ctrlKey && e.shiftKey && e.key.toLowerCase() === "o") {
        if (store.activePaneId) {
            e.preventDefault();
            store.splitPane(store.activePaneId, "horizontal");
        }
    }
}

async function fetchStats() {
    try {
        const data = await GetSystemStats();
        if (data) {
            store.stats.cpu = Math.round(data.cpu);
            store.stats.memory = Math.round(data.memory);
            store.stats.memoryRaw = data.memoryRaw;
            store.stats.uptime = data.uptime;
        }
    } catch (err) {
        console.error("Failed to poll system stats:", err);
    }
}

onMounted(async () => {
    try {
        const config = await LoadConfig();
        if (config) {
            if (config.theme) {
                store.currentTheme = config.theme;
            }
            if (config.fontSize) {
                store.fontSize = Number(config.fontSize);
            }
        }
    } catch (err) {
        console.error("Failed to load config:", err);
    } finally {
        configLoaded.value = true;
    }

    store.addTab();
    fetchStats();
    statsInterval = window.setInterval(fetchStats, 2500);
    window.addEventListener("keydown", handleGlobalKeyDown);

    // Listen for window state changes to handle border radius
    try {
        if (typeof EventsOn === "function") {
            EventsOn("wails:window-maximise", () => {
                isMaximised.value = true;
            });
            EventsOn("wails:window-unmaximise", () => {
                isMaximised.value = false;
            });
        }

        if (typeof WindowIsMaximised === "function") {
            WindowIsMaximised()
                .then((m) => {
                    isMaximised.value = m;
                })
                .catch(() => {});
        }
    } catch (err) {
        console.warn("Wails runtime not fully available yet:", err);
    }
});

watch(
    [() => store.currentTheme, () => store.fontSize],
    async () => {
        if (!configLoaded.value) return;
        try {
            await SaveConfig({
                theme: store.currentTheme,
                fontSize: store.fontSize,
            });
        } catch (err) {
            console.error("Failed to save config:", err);
        }
    }
);

onBeforeUnmount(() => {
    if (statsInterval) {
        clearInterval(statsInterval);
    }
    window.removeEventListener("keydown", handleGlobalKeyDown);
});
</script>

<template>
    <div
        :class="[
            'app-container',
            themes[store.currentTheme].cssClass,
            { 'is-maximised': isMaximised },
        ]"
    >
        <CustomTitleBar title="TermXP" />
        <div class="main-content">
            <!-- Background glow design for glassmorphism -->
            <div class="bg-glow"></div>

            <!-- Sidebar Layout -->
            <aside :class="['sidebar', { 'sidebar-closed': !store.sidebarOpen }]">
                <div class="sidebar-header">
                    <div class="logo-area">
                        <span class="logo-icon">🚀</span>
                        <h2>TermXP</h2>
                        <span class="badge">PRO</span>
                    </div>
                </div>

                <div class="sidebar-content custom-scrollbar">
                    <!-- App Status metrics -->
                    <section class="section">
                        <h3>App Metrics</h3>
                        <div class="metrics-grid">
                            <div class="metric-card">
                                <div class="metric-info">
                                    <span>App CPU</span>
                                    <span class="metric-value">{{ store.stats.cpu }}%</span>
                                </div>
                                <div class="progress-bar-container">
                                    <div
                                        class="progress-bar cpu-bar"
                                        :style="{ width: store.stats.cpu + '%' }"
                                    ></div>
                                </div>
                            </div>

                            <div class="metric-card">
                                <div class="metric-info">
                                    <span>App RAM</span>
                                    <span class="metric-value">{{ store.stats.memoryRaw }}</span>
                                </div>
                                <div class="progress-bar-container">
                                    <div
                                        class="progress-bar memory-bar"
                                        :style="{ width: store.stats.memory + '%' }"
                                    ></div>
                                </div>
                            </div>

                            <div class="metric-card single-metric">
                                <span class="metric-label">Uptime:</span>
                                <span class="metric-text font-mono">{{ store.stats.uptime }}</span>
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
                                    { active: store.currentTheme === themeKey },
                                ]"
                                @click="store.currentTheme = themeKey"
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
                                @click="store.fontSize = Math.max(10, store.fontSize - 1)"
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
                            <span class="font-display">{{ store.fontSize }}px</span>
                            <button
                                @click="store.fontSize = Math.min(24, store.fontSize + 1)"
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
                                @click="store.runSnippet(s.cmd)"
                                :disabled="!store.getActiveSessionId()"
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
                            @click="store.sidebarOpen = !store.sidebarOpen"
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
                                v-for="tab in store.tabs"
                                :key="tab.id"
                                :class="[
                                    'tab-item',
                                    { active: store.activeTabId === tab.id },
                                ]"
                                @click="store.selectTab(tab.id)"
                            >
                                <!-- Editing name input -->
                                <input
                                    v-if="store.editingTabId === tab.id"
                                    ref="renameInputRef"
                                    type="text"
                                    v-model="store.editingName"
                                    @blur="store.saveRenameTab(tab)"
                                    @keydown.enter="store.saveRenameTab(tab)"
                                    @keydown.esc="store.cancelRenameTab"
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
                                    @click.stop="store.closeTab(tab.id)"
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
                                @click="store.addTab"
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
                        v-for="tab in store.tabs"
                        :key="tab.id"
                        v-show="store.activeTabId === tab.id"
                        :node="tab.rootNode"
                    />

                    <!-- Empty state layout -->
                    <div v-if="store.tabs.length === 0" class="empty-state">
                        <div class="empty-glow"></div>
                        <div class="empty-info">
                            <span class="empty-icon">📟</span>
                            <h1>No Active Shell Sessions</h1>
                            <p>
                                Spawns local terminal processes (e.g. bash) on
                                your host machine to run CLI actions.
                            </p>
                            <button @click="store.addTab" class="action-btn">
                                <span>Start New Shell</span>
                            </button>
                        </div>
                    </div>
                </div>
            </main>
        </div>
    </div>
</template>

<style>
/* Reset and core layout */
.app-container {
    display: flex;
    flex-direction: column;
    width: 100vw;
    height: 100vh;
    overflow: hidden;
    position: relative;
    background: var(--bg-gradient);
    color: var(--text-main);
    transition:
        background var(--transition-speed),
        color var(--transition-speed);
    border: 1px solid var(--border-color);
    box-sizing: border-box;
    border-radius: 12px;
}

.app-container.is-maximised {
    border-radius: 0;
    border: none;
}

.main-content {
    display: flex;
    flex: 1;
    width: 100%;
    height: calc(100% - 32px); /* Title bar is 32px */
    overflow: hidden;
    position: relative;
}

.bg-glow {
    pointer-events: none;
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
    min-width: 0;
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
