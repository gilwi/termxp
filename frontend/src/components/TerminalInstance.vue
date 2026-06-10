<script lang="ts" setup>
import { onMounted, onBeforeUnmount, ref, watch, nextTick } from "vue";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";

// Import Wails runtime events and generated Go bindings
import { EventsOn, EventsOff } from "../../wailsjs/runtime/runtime";
import {
    StartSession,
    WriteToTerminal,
    ResizeTerminal,
    KillSession,
} from "../../wailsjs/go/main/TerminalService";

const props = defineProps<{
    theme: any;
    fontSize: number;
    active: boolean;
}>();

const emit = defineEmits<{
    (e: "exit"): void;
    (e: "initialized", sessionId: string): void;
}>();

const terminalContainer = ref<HTMLDivElement | null>(null);
const sessionId = ref<string>("");

let term: Terminal | null = null;
let fitAddon: FitAddon | null = null;
let resizeObserver: ResizeObserver | null = null;

// Initialize the terminal
onMounted(async () => {
    if (!terminalContainer.value) return;

    // 1. Create Xterm Terminal instance
    term = new Terminal({
        fontSize: props.fontSize,
        fontFamily:
            'SFMono-Regular, Consolas, Menlo, Monaco, "Liberation Mono", "Courier New", monospace',
        theme: props.theme,
        cursorBlink: true,
        cursorStyle: "block",
        cursorWidth: 2,
        drawBoldTextInBrightColors: true,
        allowProposedApi: true,
    });

    // 2. Setup Fit Addon
    fitAddon = new FitAddon();
    term.loadAddon(fitAddon);

    // 3. Open terminal in DOM container
    term.open(terminalContainer.value);

    // 4. Handle custom keys (allow Ctrl+T, Ctrl+W to bubble up to App.vue)
    term.attachCustomKeyEventHandler((e: KeyboardEvent) => {
        if (
            e.ctrlKey &&
            (e.key.toLowerCase() === "t" || e.key.toLowerCase() === "w")
        ) {
            return false; // allow to bubble
        }
        return true;
    });

    // 5. Force initial fit
    nextTick(() => {
        if (fitAddon && term) {
            fitAddon.fit();
            initSession(term.cols, term.rows);
        }
    });

    // 5. Watch for container resizing
    resizeObserver = new ResizeObserver(() => {
        if (fitAddon && term && sessionId.value) {
            fitAddon.fit();
            const newCols = term.cols;
            const newRows = term.rows;
            ResizeTerminal(sessionId.value, newCols, newRows).catch((err) => {
                console.error("Failed to resize backend terminal:", err);
            });
        }
    });
    resizeObserver.observe(terminalContainer.value);
});

// Initialize Go-side PTY process and bind event streams
async function initSession(cols: number, rows: number) {
    try {
        const sId = await StartSession(cols, rows);
        sessionId.value = sId;
        emit("initialized", sId);

        if (!term) return;

        // Listen to data streaming from backend
        EventsOn(`terminal:data:${sId}`, (data: string) => {
            term?.write(data);
        });

        // Listen to shell process termination
        EventsOn(`terminal:exit:${sId}`, () => {
            emit("exit");
        });

        // Send frontend key input to backend
        term.onData((data) => {
            if (sessionId.value) {
                WriteToTerminal(sessionId.value, data).catch((err) => {
                    console.error("Failed to write to terminal:", err);
                });
            }
        });

        // Focus if currently active
        if (props.active) {
            term.focus();
        }
    } catch (err) {
        console.error("Failed to start terminal session:", err);
        term?.write(
            "\r\n\x1b[31m[ERROR] Failed to start terminal shell session.\x1b[0m\r\n",
        );
    }
}

// Watchers for props update
watch(
    () => props.theme,
    (newTheme) => {
        if (term) {
            term.options.theme = newTheme;
        }
    },
    { deep: true },
);

watch(
    () => props.fontSize,
    (newSize) => {
        if (term && fitAddon) {
            term.options.fontSize = newSize;
            nextTick(() => {
                fitAddon?.fit();
                if (sessionId.value) {
                    ResizeTerminal(sessionId.value, term!.cols, term!.rows);
                }
            });
        }
    },
);

watch(
    () => props.active,
    (isActive) => {
        if (isActive && term && fitAddon) {
            nextTick(() => {
                fitAddon?.fit();
                term?.focus();
            });
        }
    },
);

// Cleanup on unmount
onBeforeUnmount(() => {
    if (resizeObserver) {
        resizeObserver.disconnect();
    }

    const sId = sessionId.value;
    if (sId) {
        EventsOff(`terminal:data:${sId}`);
        EventsOff(`terminal:exit:${sId}`);
        KillSession(sId).catch((err) => {
            console.error("Failed to kill terminal session:", err);
        });
    }

    if (term) {
        term.dispose();
    }
});
</script>

<template>
    <div class="terminal-wrapper">
        <div ref="terminalContainer" class="terminal-container"></div>
    </div>
</template>

<style scoped>
.terminal-wrapper {
    width: 100%;
    height: 100%;
    padding: 12px;
    background-color: transparent;
    box-sizing: border-box;
    overflow: hidden;
}

.terminal-container {
    width: 100%;
    height: 100%;
    box-sizing: border-box;
}

/* Customize xterm selection and scrollbars natively */
:deep(.xterm) {
    padding: 4px;
}

:deep(.xterm-viewport) {
    --scrollbar-thumb: var(
        --terminal-scrollbar-thumb,
        rgba(255, 255, 255, 0.15)
    );
    --scrollbar-track: var(--terminal-scrollbar-track, transparent);

    scrollbar-color: var(--scrollbar-thumb) var(--scrollbar-track);
    scrollbar-width: thin;
    overflow-y: auto;
}

:deep(.xterm-viewport::-webkit-scrollbar) {
    width: 8px;
}

:deep(.xterm-viewport::-webkit-scrollbar-thumb) {
    background: var(--scrollbar-thumb);
    border-radius: 4px;
}

:deep(.xterm-viewport::-webkit-scrollbar-thumb:hover) {
    background: var(--terminal-scrollbar-thumb-hover, rgba(255, 255, 255, 0.3));
}

:deep(.xterm-viewport::-webkit-scrollbar-track) {
    background: var(--scrollbar-track);
}
</style>
