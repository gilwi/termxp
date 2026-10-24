<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch, nextTick } from "vue";
import type { ITheme } from "@xterm/xterm";
import "@xterm/xterm/css/xterm.css";
import { acquireSession } from "../utils/sessions";
import type { TerminalNode } from "../utils/layout";
import { store } from "../utils/store";

const props = defineProps<{
  node: TerminalNode;
  theme: ITheme;
  fontSize: number;
  active: boolean;
}>();
const host = ref<HTMLDivElement | null>(null);
const searchInput = ref<HTMLInputElement | null>(null);
const query = ref("");
const retryDirectory = ref(props.node.cwd);
function reconnect() {
  if (state.status === "failed")
    view.controller.useDirectory(retryDirectory.value);
  void view.controller.start();
}
const searchResult = ref("");
const logOpen = ref(false),
  logPath = ref("");
const view = acquireSession(props.node, props.theme, props.fontSize);
const state = view.controller.state;
let observer: ResizeObserver | undefined;
let mounted = false;
function fit() {
  if (!mounted || !host.value?.offsetWidth || !host.value?.offsetHeight) return;
  try {
    view.fit.fit();
    void view.controller.resize();
  } catch (err) {
    view.controller.report(err);
  }
}
function search(backwards = false) {
  if (!query.value) {
    view.search.clearDecorations();
    searchResult.value = "";
    return;
  }
  const options = {
    decorations: {
      matchBackground: "#7c3aed",
      matchBorder: "#a78bfa",
      matchOverviewRuler: "#a78bfa",
      activeMatchBackground: "#b45309",
      activeMatchBorder: "#fbbf24",
      activeMatchColorOverviewRuler: "#fbbf24",
    },
  };
  const found = backwards
    ? view.search.findPrevious(query.value, options)
    : view.search.findNext(query.value, options);
  searchResult.value = found ? "Match found" : "No matches";
}
function closeSearch() {
  view.controller.toggleSearch(false);
  view.search.clearDecorations();
  view.term.focus();
}
function wheel(event: WheelEvent) {
  if (event.ctrlKey) {
    event.preventDefault();
    store.fontSize = Math.min(
      24,
      Math.max(10, store.fontSize + (event.deltaY < 0 ? 1 : -1)),
    );
  }
}
async function logging() {
  try {
    if (state.logging) await view.controller.stopLogging();
    else await view.controller.startLogging(logPath.value);
    logOpen.value = false;
  } catch (err) {
    view.controller.report(err);
  }
}
onMounted(() => {
  mounted = true;
  host.value!.appendChild(view.wrapper);
  if (!view.opened) {
    view.term.open(view.wrapper);
    view.opened = true;
  }
  view.term.options.theme = props.theme;
  view.term.options.fontSize = props.fontSize;
  view.copyOnSelect = store.copyOnSelect;
  view.shortcuts = store.shortcuts;
  fit();
  void view.controller.start();
  observer = new ResizeObserver(fit);
  observer.observe(host.value!);
  host.value!.addEventListener("wheel", wheel, {
    passive: false,
    capture: true,
  });
  if (props.active) view.term.focus();
});
watch(
  () => state.status,
  (status) => {
    if (status === "failed") retryDirectory.value = state.cwd;
  },
  { immediate: true },
);
watch(
  () => props.theme,
  (theme) => {
    view.term.options.theme = theme;
  },
  { deep: true },
);
watch(
  () => props.fontSize,
  (size) => {
    view.term.options.fontSize = size;
    void nextTick(fit);
  },
);
watch(
  () => store.copyOnSelect,
  (enabled) => {
    view.copyOnSelect = enabled;
  },
);
watch(
  () => store.shortcuts,
  (keys) => {
    view.shortcuts = keys;
  },
  { deep: true },
);
watch(
  () => props.active,
  (active) => {
    if (active)
      void nextTick(() => {
        fit();
        view.term.focus();
      });
  },
);
watch(
  () => state.searchOpen,
  (open) => {
    if (open) void nextTick(() => searchInput.value?.focus());
  },
);
onBeforeUnmount(() => {
  mounted = false;
  observer?.disconnect();
  host.value?.removeEventListener("wheel", wheel, { capture: true });
  if (view.wrapper.parentElement === host.value) view.wrapper.remove();
});
</script>

<template>
  <div class="terminal-instance-container">
    <div v-if="state.searchOpen" class="terminal-tools" role="search">
      <input
        ref="searchInput"
        v-model="query"
        aria-label="Search scrollback"
        placeholder="Search scrollback"
        @input="search()"
        @keydown.enter.prevent="search($event.shiftKey)"
        @keydown.esc.prevent="closeSearch"
      />
      <button @click="search(true)" aria-label="Previous match">↑</button
      ><button @click="search()" aria-label="Next match">↓</button>
      <span role="status">{{ searchResult }}</span
      ><button @click="closeSearch" aria-label="Close search">×</button>
    </div>
    <div
      v-if="state.status === 'exited' || state.status === 'failed'"
      class="terminal-tools"
      role="status"
    >
      <span>{{
        state.status === "exited"
          ? `Shell exited (${state.code})`
          : "Terminal failed"
      }}</span>
      <label v-if="state.status === 'failed'"
        >Start directory
        <input v-model="retryDirectory" placeholder="/absolute/path"
      /></label>
      <button @click="reconnect">Reconnect</button>
      <button
        v-if="state.status === 'failed' && !node.ssh"
        @click="
          view.controller.useShell('');
          reconnect();
        "
      >
        Use system default shell
      </button>
    </div>
    <div v-if="state.error" class="terminal-tools terminal-error" role="alert">
      {{ state.error }}
      <button
        @click="view.controller.dismissError()"
        aria-label="Dismiss error"
      >
        ×
      </button>
    </div>
    <div class="terminal-host" ref="host"></div>
    <div
      v-if="active && state.status === 'running'"
      class="terminal-tools terminal-footer"
    >
      <span class="cwd-label" :title="state.cwd">{{
        state.cwd || node.ssh?.host || "Shell"
      }}</span>
      <button @click="view.controller.toggleSearch()">Search</button>
      <button @click="state.logging ? logging() : (logOpen = !logOpen)">
        {{ state.logging ? "Stop logging" : "Log…" }}
      </button>
    </div>
    <div v-if="logOpen" class="terminal-tools">
      <label
        >New log file
        <input v-model="logPath" placeholder="/absolute/path/session.log"
      /></label>
      <span>Logs may contain secrets.</span
      ><button :disabled="!logPath" @click="logging">Start</button
      ><button @click="logOpen = false">Cancel</button>
    </div>
  </div>
</template>

<style scoped>
.terminal-instance-container {
  width: 100%;
  height: 100%;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.terminal-host {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
.terminal-tools {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 8px;
  background: var(--bg-surface, #181825);
  font-size: 12px;
  flex-wrap: wrap;
}
.terminal-tools input {
  background: #161622;
  color: #fff;
  border: 1px solid #666;
  border-radius: 4px;
  padding: 4px;
}
.terminal-tools button {
  color: inherit;
  background: transparent;
  border: 1px solid #666;
  border-radius: 4px;
  padding: 3px 7px;
  cursor: pointer;
}
.terminal-error {
  color: #fda4af;
}
.cwd-label {
  flex: 1;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
</style>
