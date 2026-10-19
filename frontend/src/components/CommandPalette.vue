<script setup lang="ts">
import { ref, computed, watch, onMounted } from "vue";
import { actions, fuzzyMatch, Action } from "../utils/actions";
import { store } from "../utils/store";
import { sessions } from "../utils/sessions";
const query = ref(""),
  selected = ref(0),
  input = ref<HTMLInputElement | null>(null);
const matches = computed(() =>
  actions().filter((a) => fuzzyMatch(query.value, a.label)),
);
watch(query, () => {
  selected.value = 0;
});
onMounted(() => input.value?.focus());
function close() {
  store.paletteOpen = false;
  sessions.get(store.activePaneId)?.term.focus();
}
function run(action?: Action) {
  if (action) {
    close();
    action.run();
  }
}
function key(event: KeyboardEvent) {
  if (event.key === "Escape") {
    event.preventDefault();
    close();
  }
  if (event.key === "ArrowDown") {
    event.preventDefault();
    selected.value = Math.min(matches.value.length - 1, selected.value + 1);
  }
  if (event.key === "ArrowUp") {
    event.preventDefault();
    selected.value = Math.max(0, selected.value - 1);
  }
  if (event.key === "Enter") {
    event.preventDefault();
    run(matches.value[selected.value]);
  }
  if (event.key === "Tab") {
    event.preventDefault();
    input.value?.focus();
  }
}
</script>
<template>
  <div class="palette-backdrop" @click.self="close" @keydown.stop="key">
    <section
      class="palette"
      role="dialog"
      aria-modal="true"
      aria-label="Command palette"
    >
      <input
        ref="input"
        v-model="query"
        placeholder="Search actions…"
        aria-label="Search actions"
        role="combobox"
        aria-autocomplete="list"
        aria-controls="palette-options"
        :aria-expanded="true"
        :aria-activedescendant="
          matches[selected] ? `action-${selected}` : undefined
        "
      />
      <div id="palette-options" role="listbox">
        <button
          v-for="(action, index) in matches"
          :id="`action-${index}`"
          :key="action.id"
          role="option"
          :aria-selected="selected === index"
          :class="{ selected: selected === index }"
          tabindex="-1"
          @click="run(action)"
        >
          {{ action.label }}
        </button>
        <p v-if="!matches.length">No matching actions</p>
      </div>
    </section>
  </div>
</template>
<style scoped>
.palette-backdrop {
  position: fixed;
  inset: 0;
  z-index: 1000;
  background: #0008;
  display: flex;
  justify-content: center;
  align-items: flex-start;
  padding-top: 12vh;
}
.palette {
  width: min(560px, 90vw);
  padding: 12px;
  border: 1px solid #777;
  border-radius: 12px;
  background: #181825;
  color: #fff;
}
.palette input {
  width: 100%;
  box-sizing: border-box;
  padding: 12px;
  color: inherit;
  background: #101018;
  border: 1px solid #888;
  border-radius: 6px;
}
.palette [role="listbox"] {
  max-height: 55vh;
  overflow: auto;
  margin-top: 8px;
}
.palette button {
  display: block;
  width: 100%;
  text-align: left;
  padding: 10px;
  background: transparent;
  color: inherit;
  border: 0;
  cursor: pointer;
}
.palette button.selected,
.palette button:hover {
  background: #6d28d9;
}
</style>
