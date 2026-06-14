<script lang="ts" setup>
import {
    WindowMinimise,
    WindowToggleMaximise,
    Quit,
} from "../../wailsjs/runtime/runtime";

defineProps<{
    title: string;
}>();
</script>

<template>
    <header class="custom-title-bar">
        <!-- This div captures mouse events and tells Wails to drag the window -->
        <div class="drag-region" style="--wails-drag-zone: drag"></div>

        <div class="title-section">
            <span class="app-icon">🚀</span>
            <span class="app-title font-mono">{{ title }}</span>
        </div>

        <div class="window-controls" style="--wails-drag-zone: no-drag">
            <button
                class="control-btn minimize"
                @click="WindowMinimise"
                title="Minimize"
            >
                <svg
                    width="12"
                    height="12"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                >
                    <line x1="5" y1="12" x2="19" y2="12"></line>
                </svg>
            </button>
            <button
                class="control-btn maximize"
                @click="WindowToggleMaximise"
                title="Maximize"
            >
                <svg
                    width="12"
                    height="12"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                >
                    <rect
                        x="3"
                        y="3"
                        width="18"
                        height="18"
                        rx="2"
                        ry="2"
                    ></rect>
                </svg>
            </button>
            <button class="control-btn close" @click="Quit" title="Close">
                <svg
                    width="12"
                    height="12"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                >
                    <line x1="18" y1="6" x2="6" y2="18"></line>
                    <line x1="6" y1="6" x2="18" y2="18"></line>
                </svg>
            </button>
        </div>
    </header>
</template>

<style scoped>
.custom-title-bar {
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    background: var(--header-bg);
    backdrop-filter: var(--backdrop-blur);
    -webkit-backdrop-filter: var(--backdrop-blur);
    border-bottom: 1px solid var(--border-color);
    padding: 0 12px;
    user-select: none;
    flex-shrink: 0;
    z-index: 1000;
    position: relative; /* needed for drag-region absolute positioning */
}

/* Sits behind the buttons but above the terminal, receives mouse events for Wails */
.drag-region {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    z-index: 0;
    /* No pointer-events override — must receive events for Wails drag to work */
}

.title-section {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    font-weight: 600;
    color: var(--text-muted);
    position: relative;
    z-index: 1; /* sits above drag-region, but pointer-events fall through to it */
    pointer-events: none; /* label is decorative, let clicks reach the drag-region */
}

.app-icon {
    font-size: 14px;
}

.app-title {
    text-transform: uppercase;
    letter-spacing: 1px;
}

.window-controls {
    display: flex;
    height: 100%;
    position: relative;
    z-index: 1; /* above drag-region */
    /* pointer-events left as default so buttons remain clickable */
}

.control-btn {
    width: 36px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    transition: all 0.2s;
}

.control-btn:hover {
    background: rgba(255, 255, 255, 0.1);
    color: var(--text-main);
}

.control-btn.close:hover {
    background: #e81123;
    color: white;
}

.font-mono {
    font-family: var(--font-mono);
}
</style>
