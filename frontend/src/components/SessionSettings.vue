<script setup lang="ts">
import { reactive, ref } from "vue";
import { store } from "../utils/store";
import type { SSHProfile } from "../utils/domain";
import {
  saveSSHProfile,
  deleteSSHProfile,
  persistence,
} from "../utils/settings";
const editing = ref(false),
  oldId = ref(""),
  saving = ref(false),
  error = ref("");
const empty = () => ({
  name: "",
  host: "",
  user: "",
  port: 22,
  keyPath: "",
  jumpHost: "",
});
const profile = reactive(empty());
function edit(p?: SSHProfile) {
  Object.assign(profile, p || empty());
  oldId.value = p?.id || "";
  editing.value = true;
  error.value = "";
}
async function save() {
  if (saving.value) return;
  saving.value = true;
  try {
    await saveSSHProfile({
      ...profile,
      id: oldId.value || undefined,
      port: Number(profile.port),
    });
    editing.value = false;
  } catch (err) {
    error.value = String(err);
  } finally {
    saving.value = false;
  }
}
async function remove(id: string) {
  try {
    await deleteSSHProfile(id);
  } catch (err) {
    error.value = String(err);
  }
}
</script>
<template>
  <section class="section session-settings">
    <h3>Shell preferences</h3>
    <label
      >Default shell
      <select v-model="store.defaultShell">
        <option value="">System default</option>
        <option
          v-if="
            store.defaultShell && !store.shells.includes(store.defaultShell)
          "
          :value="store.defaultShell"
        >
          {{ store.defaultShell }} (unavailable)
        </option>
        <option v-for="shell in store.shells" :key="shell" :value="shell">
          {{ shell }}
        </option>
      </select>
    </label>
    <label
      ><input type="checkbox" v-model="store.copyOnSelect" /> Copy on
      select</label
    >
    <label
      ><input type="checkbox" v-model="store.restoreWorkspace" /> Restore
      workspace on startup</label
    >
  </section>
  <section class="section session-settings">
    <h3>SSH profiles</h3>
    <div v-for="p in store.sshProfiles" :key="p.id" class="profile-row">
      <button
        @click="store.addTab(p)"
        :title="`${p.user ? p.user + '@' : ''}${p.host}`"
      >
        {{ p.name }}
      </button>
      <button @click="edit(p)" :aria-label="`Edit ${p.name}`">Edit</button>
      <button @click="remove(p.id!)" :aria-label="`Delete ${p.name}`">×</button>
    </div>
    <p class="profile-note">
      Edits apply to future connections and reconnects. Deleting a profile keeps
      a connection copy in existing panes.
    </p>
    <p v-if="error && !editing" role="alert">{{ error }}</p>
    <button v-if="!editing" @click="edit()">Add SSH profile</button>
    <form v-if="editing" @submit.prevent="save">
      <label
        >Name<input v-model="profile.name" required maxlength="128"
      /></label>
      <label>Host<input v-model="profile.host" required /></label>
      <label>User<input v-model="profile.user" /></label>
      <label
        >Port<input
          v-model.number="profile.port"
          type="number"
          min="1"
          max="65535"
          required
      /></label>
      <label
        >Key path<input
          v-model="profile.keyPath"
          placeholder="/home/user/.ssh/id_ed25519"
      /></label>
      <label
        >Jump host<input
          v-model="profile.jumpHost"
          placeholder="user@bastion:22"
      /></label>
      <p v-if="error" role="alert">{{ error }}</p>
      <button type="submit">Save profile</button
      ><button type="button" @click="editing = false">Cancel</button>
    </form>
  </section>
</template>
<style scoped>
.session-settings label {
  display: block;
  font-size: 12px;
  margin: 8px 0;
}
.session-settings input:not([type="checkbox"]),
.session-settings select {
  display: block;
  width: 100%;
  box-sizing: border-box;
  padding: 6px;
  margin: 4px 0;
  color: inherit;
  background: #171724;
  border: 1px solid #777;
  border-radius: 4px;
}
.session-settings button {
  background: transparent;
  color: inherit;
  border: 1px solid #777;
  border-radius: 4px;
  padding: 5px;
  margin: 3px 2px;
  cursor: pointer;
}
.profile-row {
  display: flex;
  align-items: center;
}
.profile-row button:first-child {
  flex: 1;
  text-align: left;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
