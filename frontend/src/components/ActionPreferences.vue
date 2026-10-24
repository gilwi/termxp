<script setup lang="ts">
import { ref } from "vue";
import { store } from "../utils/store";
import { persistence } from "../utils/settings";
import { defaultShortcuts, type Snippet } from "../utils/domain";
const editing = ref(false),
  saving = ref(false),
  error = ref("");
const shortcuts = ref<Record<string, string>>({}),
  snippets = ref<Snippet[]>([]);
function addSnippet() {
  snippets.value.push({
    id: crypto.randomUUID(),
    label: "New snippet",
    command: "",
  });
}
function edit() {
  shortcuts.value = { ...store.shortcuts };
  snippets.value = store.snippets.map((s) => ({ ...s }));
  error.value = "";
  editing.value = true;
}
async function save() {
  if (saving.value) return;
  saving.value = true;
  try {
    const nextKeys = { ...shortcuts.value },
      nextSnippets = snippets.value.map((s) => ({ ...s }));
    await persistence.edit(() => {
      const oldKeys = store.shortcuts,
        oldSnippets = store.snippets;
      store.shortcuts = nextKeys;
      store.snippets = nextSnippets;
      return () => {
        store.shortcuts = oldKeys;
        store.snippets = oldSnippets;
      };
    });
    editing.value = false;
  } catch (err) {
    error.value = String(err);
  } finally {
    saving.value = false;
  }
}
</script>
<template>
  <section class="section action-preferences">
    <h3>Actions and shortcuts</h3>
    <button v-if="!editing" @click="edit">Customize actions</button>
    <form v-else @submit.prevent="save" @keydown.stop>
      <label v-for="(_, action) in defaultShortcuts" :key="action"
        >{{ action
        }}<input
          v-model="shortcuts[action]"
          :aria-label="`${action} shortcut`"
          placeholder="Ctrl+Shift+T"
      /></label>
      <p>
        Use Ctrl, Alt, Shift and a letter or digit. Leave empty to disable. Copy
        and paste shortcuts are reserved.
      </p>
      <button type="button" @click="shortcuts = { ...defaultShortcuts }">
        Reset shortcuts
      </button>
      <fieldset v-for="snippet in snippets" :key="snippet.id">
        <label
          >Label<input v-model="snippet.label" required maxlength="128"
        /></label>
        <label
          >Command<textarea v-model="snippet.command" maxlength="65536" />
        </label>
        <button
          type="button"
          @click="snippets = snippets.filter((s) => s.id !== snippet.id)"
        >
          Remove snippet
        </button>
      </fieldset>
      <button type="button" @click="addSnippet">Add snippet</button>
      <p v-if="error" role="alert">{{ error }}</p>
      <button type="submit" :disabled="saving">Save actions</button
      ><button type="button" @click="editing = false">Cancel</button>
    </form>
  </section>
</template>
<style scoped>
.action-preferences label {
  display: block;
  font-size: 12px;
  margin: 8px 0;
}
.action-preferences input,
.action-preferences textarea {
  display: block;
  box-sizing: border-box;
  width: 100%;
  background: #171724;
  color: inherit;
  border: 1px solid #777;
  border-radius: 4px;
  padding: 6px;
}
.action-preferences button {
  background: transparent;
  color: inherit;
  border: 1px solid #777;
  border-radius: 4px;
  padding: 5px;
  margin: 3px;
}
.action-preferences fieldset {
  margin: 8px 0;
  border: 1px solid #666;
}
.action-preferences p {
  font-size: 11px;
}
</style>
