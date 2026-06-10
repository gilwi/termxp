<script lang="ts" setup>
import { ref, computed } from "vue";
import TerminalInstance from "./TerminalInstance.vue";
import { PaneNode } from "../utils/layout";

const props = defineProps<{
    node: PaneNode;
    activePaneId: string;
    theme: any;
    themeClass: string;
    fontSize: number;
}>();

const emit = defineEmits<{
    (
        e: "split-pane",
        paneId: string,
        orientation: "horizontal" | "vertical",
    ): void;
    (e: "close-pane", paneId: string): void;
    (e: "pane-initialized", paneId: string, sessionId: string): void;
    (e: "focus-pane", paneId: string): void;
    (
        e: "move-pane",
        sourceId: string,
        targetId: string,
        position: "left" | "right" | "top" | "bottom" | "swap",
    ): void;
    (e: "update-sizes", nodeId: string, newSizes: number[]): void;
}>();

// Flex style computed for split children
const containerStyle = computed(() => {
    if (props.node.type !== "split") return {};
    return {
        display: "flex",
        width: "100%",
        height: "100%",
        flexDirection: (props.node.orientation === "vertical"
            ? "row"
            : "column") as any,
        overflow: "hidden",
    };
});

function getChildStyle(index: number) {
    if (props.node.type !== "split" || !props.node.sizes) return {};
    const size = props.node.sizes[index];
    const isVertical = props.node.orientation === "vertical";
    return {
        flexGrow: 0,
        flexShrink: 0,
        width: isVertical ? `${size}%` : "100%",
        height: isVertical ? "100%" : `${size}%`,
        position: "relative" as const,
    };
}

// Resizer logic
let startPos = 0;
let startSizeA = 0;
let startSizeB = 0;
let parentSizePx = 0;
let activeSplitterIndex = -1;

const layoutContainer = ref<HTMLDivElement | null>(null);

function startResize(e: MouseEvent, index: number) {
    e.preventDefault();
    if (!props.node.children || !props.node.sizes || !layoutContainer.value)
        return;

    activeSplitterIndex = index;
    const isVertical = props.node.orientation === "vertical";
    startPos = isVertical ? e.clientX : e.clientY;

    startSizeA = props.node.sizes[index];
    startSizeB = props.node.sizes[index + 1];

    const rect = layoutContainer.value.getBoundingClientRect();
    parentSizePx = isVertical ? rect.width : rect.height;

    document.addEventListener("mousemove", onResizeDrag);
    document.addEventListener("mouseup", endResize);
    document.body.style.cursor = isVertical ? "col-resize" : "row-resize";
    document.body.style.userSelect = "none";
}

function onResizeDrag(e: MouseEvent) {
    if (activeSplitterIndex === -1 || !props.node.sizes) return;

    const isVertical = props.node.orientation === "vertical";
    const currentPos = isVertical ? e.clientX : e.clientY;
    const deltaPx = currentPos - startPos;
    const deltaPercent = (deltaPx / parentSizePx) * 100;

    let newSizeA = startSizeA + deltaPercent;
    let newSizeB = startSizeB - deltaPercent;

    // Constrain limits (minimum 10% per pane)
    if (newSizeA < 10) {
        newSizeA = 10;
        newSizeB = startSizeA + startSizeB - 10;
    } else if (newSizeB < 10) {
        newSizeB = 10;
        newSizeA = startSizeA + startSizeB - 10;
    }

    const updatedSizes = [...props.node.sizes];
    updatedSizes[activeSplitterIndex] = newSizeA;
    updatedSizes[activeSplitterIndex + 1] = newSizeB;

    emit("update-sizes", props.node.id, updatedSizes);
}

function endResize() {
    document.removeEventListener("mousemove", onResizeDrag);
    document.removeEventListener("mouseup", endResize);
    document.body.style.cursor = "";
    document.body.style.userSelect = "";
    activeSplitterIndex = -1;
}

// Drag and Drop (DND) overlay states
const dragOverActive = ref(false);
const activeDropZone = ref<"left" | "right" | "top" | "bottom" | "swap" | null>(
    null,
);
const dragCounter = ref(0);

// Context Menu State
const contextMenuVisible = ref(false);
const contextMenuX = ref(0);
const contextMenuY = ref(0);

function showContextMenu(e: MouseEvent) {
    e.preventDefault();
    emit("focus-pane", props.node.id);

    contextMenuX.value = e.clientX;
    contextMenuY.value = e.clientY;
    contextMenuVisible.value = true;

    // Listen to click events to close the context menu
    setTimeout(() => {
        document.addEventListener("click", closeContextMenu);
        document.addEventListener("contextmenu", handleGlobalContextMenu);
    }, 10);
}

function closeContextMenu() {
    contextMenuVisible.value = false;
    document.removeEventListener("click", closeContextMenu);
    document.removeEventListener("contextmenu", handleGlobalContextMenu);
}

function handleGlobalContextMenu(e: MouseEvent) {
    closeContextMenu();
}

function onHeaderDragStart(e: DragEvent) {
    if (props.node.type === "terminal" && e.dataTransfer) {
        e.dataTransfer.effectAllowed = "move";
        e.dataTransfer.setData("text/plain", props.node.id);
    }
}

function onPaneDragEnter(e: DragEvent) {
    e.preventDefault();
    dragCounter.value++;
    dragOverActive.value = true;
}

function onPaneDragLeave(e: DragEvent) {
    e.preventDefault();
    dragCounter.value--;
    if (dragCounter.value <= 0) {
        dragCounter.value = 0;
        dragOverActive.value = false;
        activeDropZone.value = null;
    }
}

function onPaneDragOver(e: DragEvent) {
    e.preventDefault();
}

function onDrop(
    e: DragEvent,
    zone: "left" | "right" | "top" | "bottom" | "swap",
) {
    e.preventDefault();
    dragCounter.value = 0;
    dragOverActive.value = false;
    activeDropZone.value = null;

    const sourceId = e.dataTransfer?.getData("text/plain");
    if (sourceId && sourceId !== props.node.id) {
        emit("move-pane", sourceId, props.node.id, zone);
    }
}
</script>

<template>
    <!-- SPLIT CONTAINER NODE -->
    <div
        v-if="node.type === 'split'"
        ref="layoutContainer"
        :style="containerStyle"
        class="split-parent"
    >
        <template v-for="(child, index) in node.children" :key="child.id">
            <TerminalLayout
                :node="child"
                :active-pane-id="activePaneId"
                :theme="theme"
                :theme-class="themeClass"
                :font-size="fontSize"
                :style="getChildStyle(index)"
                @split-pane="(pId, orient) => emit('split-pane', pId, orient)"
                @close-pane="(pId) => emit('close-pane', pId)"
                @pane-initialized="
                    (pId, sId) => emit('pane-initialized', pId, sId)
                "
                @focus-pane="(pId) => emit('focus-pane', pId)"
                @move-pane="(src, tgt, pos) => emit('move-pane', src, tgt, pos)"
                @update-sizes="
                    (nodeId, sizes) => emit('update-sizes', nodeId, sizes)
                "
            />
            <!-- Splitter bar indicator -->
            <div
                v-if="node.children && index < node.children.length - 1"
                class="splitter-bar"
                :class="node.orientation"
                @mousedown="startResize($event, index)"
            >
                <div class="splitter-line"></div>
            </div>
        </template>
    </div>

    <!-- TERMINAL PANE NODE -->
    <div
        v-else
        :class="['terminal-pane', { active: activePaneId === node.id }]"
        @click.capture="emit('focus-pane', node.id)"
        @contextmenu.prevent="showContextMenu"
        @dragenter="onPaneDragEnter"
        @dragover="onPaneDragOver"
        @dragleave="onPaneDragLeave"
    >
        <!-- Pane Header -->
        <div
            class="pane-header"
            draggable="true"
            @dragstart="onHeaderDragStart"
            title="Drag to reposition this pane"
        >
            <div class="pane-title">
                <span class="terminal-icon">🐚</span>
                <span class="title-text font-mono">Shell Session</span>
            </div>
            <div class="pane-controls">
                <!-- Split controls -->
                <button
                    class="pane-btn"
                    @click="emit('split-pane', node.id, 'vertical')"
                    title="Split Vertically"
                >
                    <svg
                        width="14"
                        height="14"
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
                        <line x1="12" y1="3" x2="12" y2="21"></line>
                    </svg>
                </button>
                <button
                    class="pane-btn"
                    @click="emit('split-pane', node.id, 'horizontal')"
                    title="Split Horizontally"
                >
                    <svg
                        width="14"
                        height="14"
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
                        <line x1="3" y1="12" x2="21" y2="12"></line>
                    </svg>
                </button>
                <!-- Close pane control -->
                <button
                    class="pane-btn close"
                    @click="emit('close-pane', node.id)"
                    title="Close Shell Pane"
                >
                    <svg
                        width="14"
                        height="14"
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
        </div>

        <!-- Terminal component -->
        <div class="pane-body">
            <TerminalInstance
                :theme="theme"
                :fontSize="fontSize"
                :active="activePaneId === node.id"
                @exit="emit('close-pane', node.id)"
                @initialized="(sId) => emit('pane-initialized', node.id, sId)"
            />
        </div>

        <!-- Drag & Drop Overlay Visual Areas -->
        <div v-if="dragOverActive" class="dnd-overlay-container">
            <!-- 5 dropzones: Top, Bottom, Left, Right, Center -->
            <div
                class="dropzone dropzone-top"
                :class="{ active: activeDropZone === 'top' }"
                @dragenter="activeDropZone = 'top'"
                @dragover.prevent
                @dragleave="activeDropZone = null"
                @drop="onDrop($event, 'top')"
            ></div>
            <div
                class="dropzone dropzone-bottom"
                :class="{ active: activeDropZone === 'bottom' }"
                @dragenter="activeDropZone = 'bottom'"
                @dragover.prevent
                @dragleave="activeDropZone = null"
                @drop="onDrop($event, 'bottom')"
            ></div>
            <div
                class="dropzone dropzone-left"
                :class="{ active: activeDropZone === 'left' }"
                @dragenter="activeDropZone = 'left'"
                @dragover.prevent
                @dragleave="activeDropZone = null"
                @drop="onDrop($event, 'left')"
            ></div>
            <div
                class="dropzone dropzone-right"
                :class="{ active: activeDropZone === 'right' }"
                @dragenter="activeDropZone = 'right'"
                @dragover.prevent
                @dragleave="activeDropZone = null"
                @drop="onDrop($event, 'right')"
            ></div>
            <div
                class="dropzone dropzone-swap"
                :class="{ active: activeDropZone === 'swap' }"
                @dragenter="activeDropZone = 'swap'"
                @dragover.prevent
                @dragleave="activeDropZone = null"
                @drop="onDrop($event, 'swap')"
            >
                <div class="swap-icon">🔄 Swap</div>
            </div>
        </div>

        <!-- Right-Click Context Menu -->
        <teleport to="body">
            <div
                v-if="contextMenuVisible"
                class="custom-context-menu"
                :class="themeClass"
                :style="{ top: contextMenuY + 'px', left: contextMenuX + 'px' }"
            >
                <button
                    class="context-menu-item"
                    @click="emit('split-pane', node.id, 'vertical')"
                >
                    <span class="item-icon">❘</span> Split Vertically
                </button>
                <button
                    class="context-menu-item"
                    @click="emit('split-pane', node.id, 'horizontal')"
                >
                    <span class="item-icon">▬</span> Split Horizontally
                </button>
                <div class="context-menu-divider"></div>
                <button
                    class="context-menu-item close-item"
                    @click="emit('close-pane', node.id)"
                >
                    <span class="item-icon font-mono">✕</span> Close Pane
                </button>
            </div>
        </teleport>
    </div>
</template>

<style scoped>
.split-parent {
    box-sizing: border-box;
}

/* Splitter bars styling */
.splitter-bar {
    background: transparent;
    flex-shrink: 0;
    position: relative;
    z-index: 5;
    transition: background 0.2s;
}

.splitter-bar:hover {
    background: var(--active-accent);
}

.splitter-bar.vertical {
    width: 6px;
    cursor: col-resize;
    height: 100%;
}

.splitter-bar.horizontal {
    height: 6px;
    cursor: row-resize;
    width: 100%;
}

/* Splitter visual indicator line */
.splitter-line {
    position: absolute;
    background: var(--border-color);
    pointer-events: none;
}

.splitter-bar.vertical .splitter-line {
    width: 1px;
    height: 100%;
    left: 3px;
}

.splitter-bar.horizontal .splitter-line {
    height: 1px;
    width: 100%;
    top: 3px;
}

/* Terminal pane wrapper styles */
.terminal-pane {
    display: flex;
    flex-direction: column;
    width: 100%;
    height: 100%;
    background: var(--card-bg);
    border: 1px solid var(--border-color);
    box-sizing: border-box;
    overflow: hidden;
    position: relative;
    transition:
        border-color 0.25s,
        box-shadow 0.25s;
}

.terminal-pane.active {
    border-color: var(--active-accent);
    box-shadow: inset 0 0 6px var(--active-accent-glow);
}

/* Pane Header controls */
.pane-header {
    height: 32px;
    background: rgba(0, 0, 0, 0.25);
    border-bottom: 1px solid var(--border-color);
    padding: 0 10px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    cursor: grab;
    user-select: none;
    flex-shrink: 0;
}

.pane-header:active {
    cursor: grabbing;
}

.pane-title {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    font-weight: 600;
    color: var(--text-muted);
}

.terminal-pane.active .pane-title {
    color: var(--active-accent);
}

.title-text {
    text-transform: uppercase;
    letter-spacing: 0.5px;
}

.pane-controls {
    display: flex;
    align-items: center;
    gap: 4px;
}

.pane-btn {
    background: transparent;
    border: none;
    color: var(--text-muted);
    width: 22px;
    height: 22px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 4px;
    cursor: pointer;
    padding: 0;
    transition: all 0.2s;
}

.pane-btn:hover {
    background: rgba(255, 255, 255, 0.08);
    color: var(--text-main);
}

.pane-btn.close:hover {
    background: rgba(244, 63, 94, 0.15);
    color: #f43f5e;
}

.pane-body {
    flex: 1;
    overflow: hidden;
    position: relative;
}

/* Drag and Drop Overlays Grid */
.dnd-overlay-container {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    z-index: 100;
    background: rgba(0, 0, 0, 0.4);
    backdrop-filter: blur(2px);
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    grid-template-rows: repeat(4, 1fr);
    box-sizing: border-box;
    pointer-events: auto;
}

.dropzone {
    box-sizing: border-box;
    transition: all 0.2s;
    position: relative;
    background: rgba(255, 255, 255, 0.02);
}

/* Map positions of zones */
.dropzone-top {
    grid-column: 1 / 5;
    grid-row: 1;
    border-bottom: 2px dashed rgba(255, 255, 255, 0.15);
}

.dropzone-bottom {
    grid-column: 1 / 5;
    grid-row: 4;
    border-top: 2px dashed rgba(255, 255, 255, 0.15);
}

.dropzone-left {
    grid-column: 1;
    grid-row: 2 / 4;
    border-right: 2px dashed rgba(255, 255, 255, 0.15);
}

.dropzone-right {
    grid-column: 4;
    grid-row: 2 / 4;
    border-left: 2px dashed rgba(255, 255, 255, 0.15);
}

.dropzone-swap {
    grid-column: 2 / 4;
    grid-row: 2 / 4;
    background: rgba(0, 0, 0, 0.6);
    border: 2px dashed rgba(255, 255, 255, 0.2);
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    margin: 10px;
}

/* Glow active drag highlights */
.dropzone.active {
    background: var(--active-accent-glow);
    border-color: var(--active-accent) !important;
    box-shadow: 0 0 15px var(--active-accent-glow);
}

.swap-icon {
    font-size: 13px;
    font-weight: 700;
    color: var(--text-main);
    text-shadow: 0 0 8px var(--active-accent-glow);
}

/* Custom Context Menu Styling */
.custom-context-menu {
    position: fixed;
    z-index: 99999;
    background: var(--sidebar-bg);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    padding: 6px 0;
    min-width: 180px;
    box-shadow:
        0 10px 30px rgba(0, 0, 0, 0.4),
        0 0 8px var(--active-accent-glow);
    backdrop-filter: var(--backdrop-blur);
    -webkit-backdrop-filter: var(--backdrop-blur);
    display: flex;
    flex-direction: column;
    overflow: hidden;
}

.context-menu-item {
    background: transparent;
    border: none;
    color: var(--text-main);
    padding: 8px 14px;
    font-size: 12px;
    font-weight: 500;
    text-align: left;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 8px;
    transition:
        background 0.15s,
        color 0.15s;
    font-family: inherit;
}

.context-menu-item:hover {
    background: var(--card-hover);
    color: var(--active-accent);
}

.context-menu-item.close-item:hover {
    background: rgba(244, 63, 94, 0.15);
    color: #f43f5e;
}

.item-icon {
    font-size: 11px;
    width: 12px;
    display: inline-block;
    color: var(--text-muted);
}

.context-menu-item:hover .item-icon {
    color: inherit;
}

.context-menu-divider {
    height: 1px;
    background: var(--border-color);
    margin: 4px 0;
}
</style>
