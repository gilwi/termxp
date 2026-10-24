import { watch } from "vue";
import { settingsBridge } from "./bridge";
import { PersistenceCoordinator } from "./persistence";
import type { Config, SSHProfile, WorkspaceNode } from "./domain";
import { store } from "./store";
import { createTerminalNode, generateUUID, PaneNode } from "./layout";
import { sessions } from "./sessions";

export function describeNode(node: PaneNode): WorkspaceNode {
  if (node.type === "terminal")
    return {
      type: "terminal",
      shell: node.shell,
      cwd: sessions.get(node.id)?.controller.state.cwd || node.cwd,
      ssh: node.ssh ? { ...node.ssh } : undefined,
      sshProfileId: node.sshProfileId,
    };
  return {
    type: "split",
    orientation: node.orientation,
    sizes: [...node.sizes],
    children: node.children.map(describeNode),
  };
}
export function restoreNode(node: WorkspaceNode): PaneNode {
  if (node.type === "terminal")
    return {
      ...createTerminalNode(generateUUID(), node.shell || "", node.cwd || ""),
      ssh: node.ssh,
      sshProfileId: node.sshProfileId,
    };
  return {
    id: generateUUID(),
    type: "split",
    orientation: node.orientation === "vertical" ? "vertical" : "horizontal",
    sizes: node.sizes || [],
    children: (node.children || []).map(restoreNode),
  };
}
export function snapshotConfig(): Config {
  return JSON.parse(
    JSON.stringify({
      version: 1,
      theme: store.currentTheme,
      fontSize: store.fontSize,
      defaultShell: store.defaultShell,
      copyOnSelect: store.copyOnSelect,
      restoreWorkspace: store.restoreWorkspace,
      workspace: store.tabs.map((t) => ({
        name: t.name,
        root: describeNode(t.rootNode),
      })),
      sshProfiles: store.sshProfiles.map((p) => ({ ...p })),
      snippets: store.snippets.map((s) => ({ ...s })),
      shortcuts: { ...store.shortcuts },
    }),
  ) as Config;
}
// All profile edits and preference saves pass through this coordinator.
export const persistence = new PersistenceCoordinator(
  snapshotConfig,
  settingsBridge.save,
  (err) => {
    store.notice = `Could not save settings: ${String(err)}`;
  },
);
export async function saveSSHProfile(profile: SSHProfile): Promise<void> {
  const next = { ...profile, id: profile.id || generateUUID() };
  await persistence.edit(() => store.updateSSHProfile(next));
}
export async function deleteSSHProfile(id: string): Promise<void> {
  await persistence.edit(() => store.deleteSSHProfile(id));
}
export async function recoverSettings(): Promise<void> {
  let backup = "";
  await persistence.recover(async (config) => {
    backup = await settingsBridge.recover(config);
  });
  store.recoveryRequired = false;
  store.notice = backup
    ? `Original configuration preserved at ${backup}`
    : "Configuration recovered";
}
export async function initializeSettings(): Promise<{
  flush(): Promise<void>;
  dispose(): void;
}> {
  const [loaded, shells] = await Promise.allSettled([
    settingsBridge.load(),
    settingsBridge.shells(),
  ]);
  if (loaded.status === "fulfilled") {
    const result = loaded.value,
      c = result.config;
    store.currentTheme = c.theme;
    store.fontSize = c.fontSize;
    store.defaultShell = c.defaultShell;
    store.copyOnSelect = c.copyOnSelect;
    store.restoreWorkspace = c.restoreWorkspace;
    store.sshProfiles = c.sshProfiles || [];
    store.snippets = c.snippets;
    store.shortcuts = c.shortcuts;
    store.notice = result.warning;
    store.recoveryRequired = result.recoveryRequired;
    if (c.restoreWorkspace && c.workspace?.length) {
      store.tabs = c.workspace.map((t) => ({
        id: generateUUID(),
        name: t.name,
        rootNode: restoreNode(t.root),
      }));
      store.selectTab(store.tabs[0].id);
    }
  } else {
    store.notice = `Settings unavailable: ${String(loaded.reason)}`;
    store.recoveryRequired = true;
  }
  if (shells.status === "fulfilled") store.shells = shells.value;
  else
    store.notice +=
      (store.notice ? "; " : "") +
      `Shell discovery unavailable: ${String(shells.reason)}`;
  persistence.blocked = store.recoveryRequired;
  if (!store.tabs.length) store.addTab();
  store.ready = true;
  const stop = watch(snapshotConfig, () => persistence.schedule(), {
    deep: true,
  });
  return {
    flush: () => persistence.save(),
    dispose: () => {
      stop();
      persistence.cancelScheduled();
    },
  };
}
