<template>
    <div class="terminal-instance-container" ref="terminalContainer"></div>
</template>

<script lang="ts" setup>
import {
    onMounted,
    onBeforeUnmount,
    ref,
    watch,
    nextTick,
    computed,
} from "vue";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import {
    StartSession,
    WriteToTerminal,
    ResizeTerminal,
} from "../../wailsjs/go/main/TerminalService";
import {
    store,
    instanceCache,
    generateUUID,
} from "../utils/store";

const props = defineProps<{
    paneId: string;
    theme: any;
    fontSize: number;
    active: boolean;
    sessionId?: string;
}>();

const emit = defineEmits<{
    (e: "exit"): void;
    (e: "initialized", sessionId: string): void;
}>();

const terminalContainer = ref<HTMLDivElement | null>(null);
const internalSessionId = ref<string>("");

const currentSessionId = computed(
    () => props.sessionId || internalSessionId.value,
);

let term: Terminal | null = null;
let fitAddon: FitAddon | null = null;
let resizeObserver: ResizeObserver | null = null;
let localWrapper: HTMLDivElement | null = null;

function safeFit(): boolean {
    if (!fitAddon || !term || !terminalContainer.value) return false;
    const width = terminalContainer.value.offsetWidth;
    const height = terminalContainer.value.offsetHeight;
    if (width > 0 && height > 0) {
        try {
            fitAddon.fit();
            return true;
        } catch (e) {
            console.error("fit failed:", e);
        }
    }
    return false;
}

onMounted(async () => {
    if (!terminalContainer.value) return;

    let cached = instanceCache.get(props.paneId);

    // If cache hits, we just re-attach the existing DOM element to the new parent
    if (cached) {
        term = cached.term;
        fitAddon = cached.fitAddon;
        localWrapper = cached.containerWrapper;

        terminalContainer.value.appendChild(localWrapper);

        term.options.theme = props.theme;
        term.options.fontSize = props.fontSize;

        nextTick(() => {
            if (safeFit()) {
                const sId = currentSessionId.value;
                if (sId) {
                    ResizeTerminal(sId, term!.cols, term!.rows).catch(() => {});
                }
            }
        });
    } else {
        // First run logic
        localWrapper = document.createElement("div");
        localWrapper.style.width = "100%";
        localWrapper.style.height = "100%";
        terminalContainer.value.appendChild(localWrapper);

        term = new Terminal({
            fontSize: props.fontSize,
            fontFamily:
                '"Anka/Coder", SFMono-Regular, Consolas, Menlo, Monaco, monospace',
            theme: props.theme,
            cursorBlink: true,
            cursorStyle: "block",
            cursorWidth: 2,
            drawBoldTextInBrightColors: true,
            allowProposedApi: true,
        });

        fitAddon = new FitAddon();
        term.loadAddon(fitAddon);

        term.open(localWrapper);

        term.attachCustomKeyEventHandler((e: KeyboardEvent) => {
            if (e.ctrlKey && e.shiftKey && e.key.toLowerCase() === "c") {
                if (e.type === "keydown" && !e.repeat) {
                    const selection = term?.getSelection();
                    if (selection) navigator.clipboard.writeText(selection);
                }
                e.preventDefault();
                e.stopPropagation();
                return false;
            }
            if (e.ctrlKey && e.shiftKey && e.key.toLowerCase() === "v") {
                if (e.type === "keydown" && !e.repeat) {
                    navigator.clipboard.readText().then((text) => {
                        const sId = currentSessionId.value;
                        if (text && sId) WriteToTerminal(sId, text);
                    });
                }
                e.preventDefault();
                e.stopPropagation();
                return false;
            }
            if (
                e.ctrlKey &&
                e.shiftKey &&
                ["t", "w", "x", "e", "o"].includes(e.key.toLowerCase())
            ) {
                return false;
            }
            return true;
        });

        // Initialize cache
        instanceCache.set(props.paneId, {
            term,
            fitAddon,
            containerWrapper: localWrapper,
            initialized: false,
            sId: "",
        });

        setTimeout(() => {
            if (fitAddon && term) {
                // Try to fit, but suppress errors if container is still hiding
                safeFit();

                // Floor the dimensions to a minimum safe size so the Go backend doesn't crash
                const cols = Math.max(term.cols || 80, 20);
                const rows = Math.max(term.rows || 24, 10);

                initSession(cols, rows);
            }
        }, 50);
    }

    resizeObserver = new ResizeObserver(() => {
        if (safeFit()) {
            const sId = currentSessionId.value;
            if (sId) {
                ResizeTerminal(sId, term!.cols, term!.rows).catch((err) => {
                    console.error("Failed to resize backend terminal:", err);
                });
            }
        }
    });
    resizeObserver.observe(terminalContainer.value);
});

async function initSession(cols: number, rows: number) {
    try {
        let sId: string = props.sessionId || "";
        const isNewSession = !sId;
        if (isNewSession) {
            sId = generateUUID();
            internalSessionId.value = sId;
            emit("initialized", sId);
        }

        if (!term) return;

        const cached = instanceCache.get(props.paneId);
        if (cached) {
            cached.sId = sId;
            if (cached.initialized) return; // Prevent double-binding events
            cached.initialized = true;
        }

        // Register listeners BEFORE starting the backend session to prevent race condition
        EventsOn(`terminal:data:${sId}`, (data: string) => {
            term?.write(data);
        });

        EventsOn(`terminal:exit:${sId}`, () => {
            emit("exit");
        });

        term.onData((data) => {
            const currentId = currentSessionId.value;
            if (currentId) {
                WriteToTerminal(currentId, data).catch((err) => {
                    console.error("Failed to write to terminal:", err);
                });
            }
        });

        if (isNewSession) {
            await StartSession(sId, cols, rows);
        } else {
            await ResizeTerminal(sId, cols, rows);
        }

        // Sync backend size if frontend was resized/fitted during initialization
        if (term.cols !== cols || term.rows !== rows) {
            await ResizeTerminal(sId, term.cols, term.rows).catch(() => {});
        }

        if (props.active) term.focus();
    } catch (err) {
        console.error("Failed to start terminal session:", err);
        term?.write(
            "\r\n\x1b[31m[ERROR] Failed to start terminal shell session.\x1b[0m\r\n",
        );
    }
}

watch(
    () => props.theme,
    (newTheme) => {
        if (term) term.options.theme = newTheme;
    },
    { deep: true },
);

watch(
    () => props.fontSize,
    (newSize) => {
        if (term) {
            term.options.fontSize = newSize;
            nextTick(() => {
                if (safeFit()) {
                    const sId = currentSessionId.value;
                    if (sId)
                        ResizeTerminal(sId, term!.cols, term!.rows).catch(() => {});
                }
            });
        }
    },
);

watch(
    () => props.active,
    (isActive) => {
        if (isActive && term) {
            nextTick(() => {
                safeFit();
                term?.focus();
            });
        }
    },
);

onBeforeUnmount(() => {
    if (resizeObserver) resizeObserver.disconnect();
});
</script>

<style scoped>
.terminal-instance-container {
    width: 100%;
    height: 100%;
    overflow: hidden;
}
</style>
