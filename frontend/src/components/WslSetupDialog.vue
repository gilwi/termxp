<script lang="ts" setup>
import { ref, onMounted } from "vue";
import {
    ListWSLDistros,
    GetWSLDistro,
    SetWSLDistro,
} from "../../wailsjs/go/main/TerminalService";

const emit = defineEmits<{ (e: "done"): void }>();

const distros = ref<string[]>([]);
const selected = ref<string>("");
const loading = ref(true);
const saving = ref(false);
const error = ref("");

onMounted(async () => {
    try {
        const [list, current] = await Promise.all([
            ListWSLDistros(),
            GetWSLDistro(),
        ]);
        distros.value = list ?? [];
        selected.value = current || distros.value[0] || "";
    } catch (e: any) {
        error.value = "Failed to detect WSL distros: " + e;
    } finally {
        loading.value = false;
    }
});

async function confirm() {
    if (!selected.value) return;
    saving.value = true;
    try {
        await SetWSLDistro(selected.value);
        emit("done");
    } catch (e: any) {
        error.value = "Failed to save: " + e;
    } finally {
        saving.value = false;
    }
}
</script>

<template>
    <div class="wsl-overlay">
        <div class="wsl-dialog">
            <div class="wsl-dialog-header">
                <span class="wsl-icon">🐧</span>
                <h2>Choose WSL Distro</h2>
            </div>

            <p class="wsl-subtitle">
                Select the WSL distribution TermXP should use for shell
                sessions.
            </p>

            <div v-if="loading" class="wsl-loading">Detecting distros…</div>

            <div v-else-if="distros.length === 0" class="wsl-empty">
                No WSL distros found. Install one from the Microsoft Store and
                restart TermXP.
            </div>

            <div v-else class="wsl-list">
                <button
                    v-for="d in distros"
                    :key="d"
                    :class="['wsl-item', { active: selected === d }]"
                    @click="selected = d"
                >
                    <span class="wsl-item-name">{{ d }}</span>
                    <span v-if="selected === d" class="wsl-check">✓</span>
                </button>
            </div>

            <p v-if="error" class="wsl-error">{{ error }}</p>

            <div class="wsl-actions">
                <button
                    class="wsl-btn-confirm"
                    :disabled="!selected || saving"
                    @click="confirm"
                >
                    {{ saving ? "Saving…" : "Confirm" }}
                </button>
            </div>
        </div>
    </div>
</template>

<style scoped>
.wsl-overlay {
    position: fixed;
    inset: 0;
    z-index: 9999;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.6);
    backdrop-filter: blur(6px);
}

.wsl-dialog {
    background: var(--sidebar-bg, rgba(21, 15, 47, 0.95));
    border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
    border-radius: 12px;
    padding: 28px 32px;
    width: 360px;
    max-width: 90vw;
    box-shadow: 0 24px 64px rgba(0, 0, 0, 0.5);
    display: flex;
    flex-direction: column;
    gap: 16px;
}

.wsl-dialog-header {
    display: flex;
    align-items: center;
    gap: 10px;
}

.wsl-icon {
    font-size: 22px;
}

h2 {
    margin: 0;
    font-size: 16px;
    font-weight: 700;
    color: var(--text-main, #f1f5f9);
    font-family: var(--font-mono);
    text-transform: uppercase;
    letter-spacing: 1px;
}

.wsl-subtitle {
    margin: 0;
    font-size: 12px;
    color: var(--text-muted, rgba(241, 245, 249, 0.5));
    line-height: 1.5;
}

.wsl-loading,
.wsl-empty {
    font-size: 12px;
    color: var(--text-muted, rgba(241, 245, 249, 0.5));
    text-align: center;
    padding: 12px 0;
}

.wsl-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
    max-height: 240px;
    overflow-y: auto;
}

.wsl-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 14px;
    border-radius: 8px;
    border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
    background: transparent;
    color: var(--text-main, #f1f5f9);
    font-family: var(--font-mono);
    font-size: 13px;
    cursor: pointer;
    transition:
        background 0.15s,
        border-color 0.15s;
    text-align: left;
}

.wsl-item:hover {
    background: rgba(255, 255, 255, 0.06);
}

.wsl-item.active {
    border-color: var(--accent, #9333ea);
    background: rgba(147, 51, 234, 0.12);
}

.wsl-check {
    color: var(--accent, #9333ea);
    font-size: 14px;
}

.wsl-error {
    margin: 0;
    font-size: 11px;
    color: #f43f5e;
}

.wsl-actions {
    display: flex;
    justify-content: flex-end;
}

.wsl-btn-confirm {
    padding: 8px 24px;
    border-radius: 8px;
    border: none;
    background: var(--accent, #9333ea);
    color: #fff;
    font-family: var(--font-mono);
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    transition: opacity 0.15s;
}

.wsl-btn-confirm:hover:not(:disabled) {
    opacity: 0.85;
}

.wsl-btn-confirm:disabled {
    opacity: 0.4;
    cursor: not-allowed;
}
</style>
