<script lang="ts" setup>
import { ref, watch, onMounted } from "vue";

interface SnippetConfig {
    label: string;
    cmd: string;
}

interface Settings {
    theme: string;
    fontSize: number;
    defaultShell: string;
    sidebarOpenDefault: boolean;
    snippets: SnippetConfig[];
}

const props = defineProps<{
    modelValue: boolean;
    themes: any;
    initialSettings: Settings;
}>();

const emit = defineEmits<{
    (e: "update:modelValue", value: boolean): void;
    (e: "save", settings: Settings): void;
    (e: "preview-theme", theme: string): void;
}>();

const activeTab = ref<"appearance" | "terminal" | "snippets" | "shortcuts">("appearance");

// Local copy of settings
const settingsCopy = ref<Settings>({
    theme: "glassmorphic",
    fontSize: 14,
    defaultShell: "",
    sidebarOpenDefault: false,
    snippets: [],
});

// Sync copy when modal opens
watch(
    () => props.modelValue,
    (isOpen) => {
        if (isOpen) {
            settingsCopy.value = JSON.parse(JSON.stringify(props.initialSettings));
        }
    }
);

// Preview theme instantly
watch(
    () => settingsCopy.value.theme,
    (newTheme) => {
        emit("preview-theme", newTheme);
    }
);

// Snippet manager state
const newLabel = ref("");
const newCmd = ref("");
const editingSnippetIndex = ref<number | null>(null);
const editLabel = ref("");
const editCmd = ref("");

function addSnippet() {
    if (newLabel.value.trim() && newCmd.value.trim()) {
        settingsCopy.value.snippets.push({
            label: newLabel.value.trim(),
            cmd: newCmd.value.trim(),
        });
        newLabel.value = "";
        newCmd.value = "";
    }
}

function removeSnippet(index: number) {
    settingsCopy.value.snippets.splice(index, 1);
    if (editingSnippetIndex.value === index) {
        editingSnippetIndex.value = null;
    }
}

function startEditSnippet(index: number) {
    editingSnippetIndex.value = index;
    editLabel.value = settingsCopy.value.snippets[index].label;
    editCmd.value = settingsCopy.value.snippets[index].cmd;
}

function saveEditSnippet(index: number) {
    if (editLabel.value.trim() && editCmd.value.trim()) {
        settingsCopy.value.snippets[index] = {
            label: editLabel.value.trim(),
            cmd: editCmd.value.trim(),
        };
        editingSnippetIndex.value = null;
    }
}

function cancelEditSnippet() {
    editingSnippetIndex.value = null;
}

function handleSave() {
    emit("save", JSON.parse(JSON.stringify(settingsCopy.value)));
    emit("update:modelValue", false);
}

function handleClose() {
    // Revert preview theme to initial settings
    emit("preview-theme", props.initialSettings.theme);
    emit("update:modelValue", false);
}
</script>

<template>
    <div v-if="modelValue" class="settings-overlay" @click.self="handleClose">
        <div class="settings-modal">
            <!-- Modal Header -->
            <header class="modal-header">
                <div class="header-title">
                    <span class="header-icon">⚙️</span>
                    <h2>Application Settings</h2>
                </div>
                <button class="close-btn" @click="handleClose" title="Close settings">
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                        <line x1="18" y1="6" x2="6" y2="18"></line>
                        <line x1="6" y1="6" x2="18" y2="18"></line>
                    </svg>
                </button>
            </header>

            <!-- Modal Body -->
            <div class="modal-body">
                <!-- Tabs Sidebar -->
                <aside class="modal-sidebar">
                    <button 
                        :class="['tab-btn', { active: activeTab === 'appearance' }]" 
                        @click="activeTab = 'appearance'"
                    >
                        🎨 Appearance
                    </button>
                    <button 
                        :class="['tab-btn', { active: activeTab === 'terminal' }]" 
                        @click="activeTab = 'terminal'"
                    >
                        💻 Shell & Term
                    </button>
                    <button 
                        :class="['tab-btn', { active: activeTab === 'snippets' }]" 
                        @click="activeTab = 'snippets'"
                    >
                        ⚡ Quick Actions
                    </button>
                    <button 
                        :class="['tab-btn', { active: activeTab === 'shortcuts' }]" 
                        @click="activeTab = 'shortcuts'"
                    >
                        ⌨️ Shortcuts
                    </button>
                </aside>

                <!-- Tab Content Area -->
                <main class="modal-content custom-scrollbar">
                    <!-- Appearance Tab -->
                    <section v-if="activeTab === 'appearance'" class="settings-section">
                        <h3>Theme & Styles</h3>
                        <p class="section-desc">Choose the primary visual style for the dashboard and terminal panes.</p>
                        
                        <div class="themes-grid">
                            <button
                                v-for="(themeConfig, themeKey) in themes"
                                :key="themeKey"
                                :class="['theme-card', { active: settingsCopy.theme === String(themeKey) }]"
                                @click="settingsCopy.theme = String(themeKey)"
                            >
                                <div class="theme-card-preview" :style="{ background: themeConfig.xterm.background }">
                                    <div class="colors">
                                        <span :style="{ background: themeConfig.xterm.red }"></span>
                                        <span :style="{ background: themeConfig.xterm.green }"></span>
                                        <span :style="{ background: themeConfig.xterm.yellow }"></span>
                                        <span :style="{ background: themeConfig.xterm.blue }"></span>
                                    </div>
                                </div>
                                <span class="theme-card-name">{{ themeConfig.name }}</span>
                            </button>
                        </div>

                        <div class="setting-divider"></div>

                        <h3>Font Settings</h3>
                        <p class="section-desc">Adjust the typography size within all terminal shells.</p>
                        <div class="font-controls">
                            <button
                                @click="settingsCopy.fontSize = Math.max(10, settingsCopy.fontSize - 1)"
                                class="font-btn"
                            >
                                -
                            </button>
                            <span class="font-display">{{ settingsCopy.fontSize }}px</span>
                            <button
                                @click="settingsCopy.fontSize = Math.min(24, settingsCopy.fontSize + 1)"
                                class="font-btn"
                            >
                                +
                            </button>
                        </div>
                    </section>

                    <!-- Shell & Terminal Tab -->
                    <section v-if="activeTab === 'terminal'" class="settings-section">
                        <h3>PTY Shell Configurations</h3>
                        <p class="section-desc">Define the shell spawned when opening new sessions.</p>
                        
                        <div class="field-group">
                            <label for="defaultShell">Default Shell Path</label>
                            <input 
                                id="defaultShell"
                                type="text"
                                v-model="settingsCopy.defaultShell"
                                placeholder="e.g. /bin/bash, /bin/zsh, /bin/sh (leave blank for system default)"
                                class="settings-input"
                            />
                            <small class="field-help">Defaults to your user account shell ($SHELL environment variable) if left empty.</small>
                        </div>

                        <div class="setting-divider"></div>

                        <h3>Layout & Panel Defaults</h3>
                        <p class="section-desc">Configure dashboard display behaviors on startup.</p>

                        <div class="checkbox-group">
                            <label class="settings-checkbox-label">
                                <input 
                                    type="checkbox"
                                    v-model="settingsCopy.sidebarOpenDefault"
                                    class="settings-checkbox"
                                />
                                <span class="checkbox-custom"></span>
                                Open sidebar automatically on startup
                            </label>
                        </div>
                    </section>

                    <!-- Snippets Tab -->
                    <section v-if="activeTab === 'snippets'" class="settings-section">
                        <h3>Quick Command Snippets</h3>
                        <p class="section-desc">Add, edit, or remove pre-configured scripts that can be sent directly to terminal sessions.</p>
                        
                        <!-- Snippet Creator Form -->
                        <div class="add-snippet-form">
                            <h4>Add New Snippet</h4>
                            <div class="form-inputs">
                                <input 
                                    type="text" 
                                    v-model="newLabel" 
                                    placeholder="Label (e.g. Docker Status)" 
                                    class="settings-input"
                                />
                                <input 
                                    type="text" 
                                    v-model="newCmd" 
                                    placeholder="Command (e.g. docker ps)" 
                                    class="settings-input font-mono"
                                    @keydown.enter="addSnippet"
                                />
                                <button @click="addSnippet" class="add-btn">Add</button>
                            </div>
                        </div>

                        <!-- Snippets List -->
                        <div class="snippets-manager-list">
                            <h4>Configured Snippets</h4>
                            <div v-if="settingsCopy.snippets.length === 0" class="no-snippets">
                                No quick actions configured. Create one above!
                            </div>
                            <div 
                                v-for="(snippet, index) in settingsCopy.snippets" 
                                :key="index"
                                class="snippet-manager-item"
                            >
                                <!-- Edit Mode -->
                                <div v-if="editingSnippetIndex === index" class="snippet-edit-row">
                                    <input type="text" v-model="editLabel" class="settings-input sm" />
                                    <input type="text" v-model="editCmd" class="settings-input sm font-mono" />
                                    <div class="snippet-edit-actions">
                                        <button @click="saveEditSnippet(index)" class="action-btn-sm save">✓</button>
                                        <button @click="cancelEditSnippet" class="action-btn-sm cancel">✕</button>
                                    </div>
                                </div>
                                <!-- Display Mode -->
                                <template v-else>
                                    <div class="snippet-info">
                                        <strong class="snippet-label">{{ snippet.label }}</strong>
                                        <code class="snippet-code font-mono">{{ snippet.cmd }}</code>
                                    </div>
                                    <div class="snippet-actions">
                                        <button @click="startEditSnippet(index)" class="action-btn-sm edit" title="Edit">✏️</button>
                                        <button @click="removeSnippet(index)" class="action-btn-sm delete" title="Delete">🗑️</button>
                                    </div>
                                </template>
                            </div>
                        </div>
                    </section>

                    <!-- Shortcuts Tab -->
                    <section v-if="activeTab === 'shortcuts'" class="settings-section">
                        <h3>Keyboard Shortcuts</h3>
                        <p class="section-desc">Reference keybind bindings configured for faster application handling.</p>
                        
                        <table class="shortcuts-table">
                            <thead>
                                <tr>
                                    <th>Shortcut</th>
                                    <th>Action</th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr>
                                    <td><kbd>Ctrl</kbd> + <kbd>Shift</kbd> + <kbd>T</kbd></td>
                                    <td>Open New Shell Tab</td>
                                </tr>
                                <tr>
                                    <td><kbd>Ctrl</kbd> + <kbd>Shift</kbd> + <kbd>W</kbd></td>
                                    <td>Close Current Tab</td>
                                </tr>
                                <tr>
                                    <td><kbd>Ctrl</kbd> + <kbd>Shift</kbd> + <kbd>E</kbd></td>
                                    <td>Split Pane Vertically (Right)</td>
                                </tr>
                                <tr>
                                    <td><kbd>Ctrl</kbd> + <kbd>Shift</kbd> + <kbd>O</kbd></td>
                                    <td>Split Pane Horizontally (Bottom)</td>
                                </tr>
                                <tr>
                                    <td><kbd>Ctrl</kbd> + <kbd>Shift</kbd> + <kbd>X</kbd></td>
                                    <td>Toggle Maximize Active Pane</td>
                                </tr>
                                <tr>
                                    <td><kbd>Ctrl</kbd> + <kbd>Shift</kbd> + <kbd>C</kbd></td>
                                    <td>Copy Selected Text (inside Terminal)</td>
                                </tr>
                                <tr>
                                    <td><kbd>Ctrl</kbd> + <kbd>Shift</kbd> + <kbd>V</kbd></td>
                                    <td>Paste Clipboard Text (inside Terminal)</td>
                                </tr>
                            </tbody>
                        </table>
                    </section>
                </main>
            </div>

            <!-- Modal Footer -->
            <footer class="modal-footer">
                <button class="footer-btn secondary" @click="handleClose">Cancel</button>
                <button class="footer-btn primary" @click="handleSave">Save & Apply</button>
            </footer>
        </div>
    </div>
</template>

<style scoped>
.settings-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background: rgba(0, 0, 0, 0.5);
    backdrop-filter: blur(10px);
    -webkit-backdrop-filter: blur(10px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 20000;
}

.settings-modal {
    width: 680px;
    height: 500px;
    background: var(--sidebar-bg);
    border: 1px solid var(--border-color);
    border-radius: 12px;
    box-shadow: 0 20px 40px rgba(0, 0, 0, 0.5), 0 0 20px var(--active-accent-glow);
    display: flex;
    flex-direction: column;
    overflow: hidden;
    animation: scaleUp 0.25s cubic-bezier(0.16, 1, 0.3, 1);
}

@keyframes scaleUp {
    from {
        opacity: 0;
        transform: scale(0.95);
    }
    to {
        opacity: 1;
        transform: scale(1);
    }
}

.modal-header {
    height: 48px;
    border-bottom: 1px solid var(--border-color);
    padding: 0 16px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    background: rgba(0, 0, 0, 0.2);
}

.header-title {
    display: flex;
    align-items: center;
    gap: 8px;
}

.header-title h2 {
    margin: 0;
    font-size: 15px;
    font-weight: 600;
    color: var(--text-main);
}

.header-icon {
    font-size: 16px;
}

.close-btn {
    background: transparent;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border-radius: 6px;
    transition: all 0.2s;
}

.close-btn:hover {
    background: rgba(255, 255, 255, 0.08);
    color: var(--text-main);
}

.modal-body {
    flex: 1;
    display: flex;
    overflow: hidden;
}

.modal-sidebar {
    width: 170px;
    border-right: 1px solid var(--border-color);
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 6px;
    background: rgba(0, 0, 0, 0.1);
}

.tab-btn {
    background: transparent;
    border: none;
    color: var(--text-muted);
    padding: 10px 12px;
    border-radius: 6px;
    text-align: left;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
}

.tab-btn:hover {
    background: rgba(255, 255, 255, 0.04);
    color: var(--text-main);
}

.tab-btn.active {
    background: var(--active-accent-glow);
    color: var(--text-main);
    box-shadow: 0 0 10px rgba(147, 51, 234, 0.2);
}

.modal-content {
    flex: 1;
    padding: 18px;
    overflow-y: auto;
}

.settings-section {
    display: flex;
    flex-direction: column;
    gap: 12px;
}

.settings-section h3 {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--text-main);
}

.section-desc {
    margin: -4px 0 8px 0;
    font-size: 12px;
    color: var(--text-muted);
    line-height: 1.4;
}

.setting-divider {
    height: 1px;
    background: var(--border-color);
    margin: 8px 0;
}

/* Themes grid */
.themes-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 10px;
}

.theme-card {
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    padding: 8px;
    cursor: pointer;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    transition: all 0.2s;
}

.theme-card:hover {
    background: rgba(255, 255, 255, 0.05);
    border-color: rgba(255, 255, 255, 0.2);
}

.theme-card.active {
    border-color: var(--active-accent);
    background: rgba(147, 51, 234, 0.05);
    box-shadow: 0 0 8px var(--active-accent-glow);
}

.theme-card-preview {
    width: 100%;
    height: 48px;
    border-radius: 4px;
    display: flex;
    align-items: flex-end;
    justify-content: flex-start;
    padding: 6px;
    box-sizing: border-box;
}

.theme-card-preview .colors {
    display: flex;
    gap: 3px;
}

.theme-card-preview .colors span {
    width: 8px;
    height: 8px;
    border-radius: 50%;
}

.theme-card-name {
    font-size: 11px;
    font-weight: 500;
    color: var(--text-main);
}

/* Font control */
.font-controls {
    display: flex;
    align-items: center;
    gap: 12px;
}

.font-btn {
    width: 32px;
    height: 32px;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid var(--border-color);
    border-radius: 6px;
    color: var(--text-main);
    font-size: 16px;
    font-weight: 600;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s;
}

.font-btn:hover {
    background: rgba(255, 255, 255, 0.1);
    border-color: rgba(255, 255, 255, 0.3);
}

.font-display {
    font-size: 13px;
    font-weight: 600;
    color: var(--text-main);
    min-width: 48px;
    text-align: center;
}

/* Inputs & Form Groups */
.field-group {
    display: flex;
    flex-direction: column;
    gap: 6px;
}

.field-group label {
    font-size: 12px;
    font-weight: 600;
    color: var(--text-main);
}

.settings-input {
    background: rgba(0, 0, 0, 0.3);
    border: 1px solid var(--border-color);
    border-radius: 6px;
    color: var(--text-main);
    padding: 8px 12px;
    font-size: 12px;
    outline: none;
    transition: all 0.2s;
    font-family: inherit;
}

.settings-input:focus {
    border-color: var(--active-accent);
    box-shadow: 0 0 6px var(--active-accent-glow);
}

.settings-input.sm {
    padding: 6px 10px;
    font-size: 11px;
}

.field-help {
    font-size: 11px;
    color: var(--text-muted);
}

/* Checkbox design */
.checkbox-group {
    margin: 6px 0;
}

.settings-checkbox-label {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    cursor: pointer;
    color: var(--text-main);
    user-select: none;
}

.settings-checkbox {
    position: absolute;
    opacity: 0;
    cursor: pointer;
    height: 0;
    width: 0;
}

.checkbox-custom {
    height: 18px;
    width: 18px;
    background: rgba(0, 0, 0, 0.3);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    position: relative;
    transition: all 0.2s;
}

.settings-checkbox:checked ~ .checkbox-custom {
    background: var(--active-accent);
    border-color: var(--active-accent);
}

.checkbox-custom:after {
    content: "";
    position: absolute;
    display: none;
    left: 5px;
    top: 2px;
    width: 4px;
    height: 8px;
    border: solid white;
    border-width: 0 2px 2px 0;
    transform: rotate(45deg);
}

.settings-checkbox:checked ~ .checkbox-custom:after {
    display: block;
}

/* Snippets CRUD manager */
.add-snippet-form {
    background: rgba(0, 0, 0, 0.15);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 8px;
}

.add-snippet-form h4 {
    margin: 0;
    font-size: 12px;
    font-weight: 600;
    color: var(--text-main);
}

.form-inputs {
    display: grid;
    grid-template-columns: 140px 1fr 60px;
    gap: 8px;
}

.add-btn {
    background: var(--active-accent);
    border: none;
    color: white;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.2s;
}

.add-btn:hover {
    background: var(--active-accent-hover);
}

.snippets-manager-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
}

.snippets-manager-list h4 {
    margin: 12px 0 4px 0;
    font-size: 12px;
    font-weight: 600;
    color: var(--text-main);
}

.no-snippets {
    font-size: 11px;
    color: var(--text-muted);
    text-align: center;
    padding: 16px;
    background: rgba(255, 255, 255, 0.01);
    border: 1px dashed var(--border-color);
    border-radius: 8px;
}

.snippet-manager-item {
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    padding: 8px 12px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    height: 48px;
    box-sizing: border-box;
}

.snippet-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
    overflow: hidden;
}

.snippet-label {
    font-size: 12px;
    color: var(--text-main);
    text-overflow: ellipsis;
    white-space: nowrap;
    overflow: hidden;
}

.snippet-code {
    font-size: 11px;
    color: var(--text-muted);
    text-overflow: ellipsis;
    white-space: nowrap;
    overflow: hidden;
}

.snippet-actions {
    display: flex;
    gap: 6px;
}

.action-btn-sm {
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    color: var(--text-main);
    width: 24px;
    height: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    font-size: 11px;
    transition: all 0.2s;
}

.action-btn-sm:hover {
    background: rgba(255, 255, 255, 0.1);
}

.action-btn-sm.delete:hover {
    background: rgba(244, 63, 94, 0.15);
    border-color: #f43f5e;
    color: #f43f5e;
}

.snippet-edit-row {
    display: flex;
    width: 100%;
    gap: 8px;
}

.snippet-edit-row .settings-input {
    flex: 1;
}

.snippet-edit-row .settings-input:first-child {
    max-width: 120px;
}

.snippet-edit-actions {
    display: flex;
    gap: 4px;
}

.action-btn-sm.save {
    background: rgba(16, 185, 129, 0.1);
    border-color: #10b981;
    color: #10b981;
}

.action-btn-sm.cancel {
    background: rgba(244, 63, 94, 0.1);
    border-color: #f43f5e;
    color: #f43f5e;
}

/* Shortcuts Table */
.shortcuts-table {
    width: 100%;
    border-collapse: collapse;
    margin-top: 4px;
}

.shortcuts-table th, 
.shortcuts-table td {
    padding: 10px 12px;
    text-align: left;
    font-size: 12px;
}

.shortcuts-table th {
    font-weight: 600;
    color: var(--text-muted);
    border-bottom: 2px solid var(--border-color);
}

.shortcuts-table td {
    color: var(--text-main);
    border-bottom: 1px solid var(--border-color);
}

kbd {
    background: rgba(255, 255, 255, 0.08);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    padding: 2px 5px;
    font-size: 10px;
    font-family: inherit;
    box-shadow: 0 1px 0 rgba(0, 0, 0, 0.2);
}

/* Modal Footer */
.modal-footer {
    height: 52px;
    border-top: 1px solid var(--border-color);
    padding: 0 16px;
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 10px;
    background: rgba(0, 0, 0, 0.1);
}

.footer-btn {
    padding: 8px 16px;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s;
    font-family: inherit;
}

.footer-btn.primary {
    background: var(--active-accent);
    border: 1px solid var(--active-accent);
    color: white;
}

.footer-btn.primary:hover {
    background: var(--active-accent-hover);
    border-color: var(--active-accent-hover);
    box-shadow: 0 0 12px var(--active-accent-glow);
}

.footer-btn.secondary {
    background: transparent;
    border: 1px solid var(--border-color);
    color: var(--text-muted);
}

.footer-btn.secondary:hover {
    background: rgba(255, 255, 255, 0.04);
    color: var(--text-main);
}

.font-mono {
    font-family: var(--font-mono) !important;
}
</style>
