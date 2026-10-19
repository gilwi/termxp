import { watch } from "vue";
import { main } from "../../wailsjs/go/models";
import { LoadConfig, SaveConfig } from "../../wailsjs/go/main/ConfigService";
import { ListShells } from "../../wailsjs/go/main/TerminalService";
import { store } from "./store";
import { createTerminalNode, generateUUID, PaneNode } from "./layout";
import { sessions } from "./sessions";

export function describeNode(node: PaneNode): object {
  if (node.type === "terminal")
    return {
      type: "terminal",
      shell: node.shell,
      cwd: sessions.get(node.id)?.controller.state.cwd || node.cwd,
      ssh: node.ssh,
    };
  return {
    type: "split",
    orientation: node.orientation,
    sizes: node.sizes,
    children: node.children.map(describeNode),
  };
}
export function restoreNode(node: main.WorkspaceNode): PaneNode {
  if (node.type === "terminal")
    return {
      ...createTerminalNode(generateUUID(), node.shell || "", node.cwd || ""),
      ssh: node.ssh,
    };
  return {
    id: generateUUID(),
    type: "split",
    orientation: node.orientation === "vertical" ? "vertical" : "horizontal",
    sizes: node.sizes || [],
    children: (node.children || []).map(restoreNode),
  };
}
export function snapshotConfig(): main.Config {
  return new main.Config({
    version: 1,
    theme: store.currentTheme,
    fontSize: store.fontSize,
    defaultShell: store.defaultShell,
    copyOnSelect: store.copyOnSelect,
    restoreWorkspace: store.restoreWorkspace,
    sshProfiles: store.sshProfiles,
    workspace: store.tabs.map((tab) => ({
      name: tab.name,
      root: describeNode(tab.rootNode),
    })),
  });
}
export async function initializeSettings(): Promise<() => Promise<void>> {
  try {
    const [result, shells] = await Promise.all([LoadConfig(), ListShells()]);
    const c = result.config;
    store.currentTheme = c.theme;
    store.fontSize = c.fontSize;
    store.defaultShell = c.defaultShell;
    store.copyOnSelect = c.copyOnSelect;
    store.restoreWorkspace = c.restoreWorkspace;
    store.sshProfiles = c.sshProfiles || [];
    store.shells = shells;
    store.notice = result.warning;
    if (c.restoreWorkspace && c.workspace?.length) {
      store.tabs = c.workspace.map((tab) => ({
        id: generateUUID(),
        name: tab.name,
        rootNode: restoreNode(tab.root),
      }));
      store.selectTab(store.tabs[0].id);
    }
  } catch (err) {
    store.notice = `Settings unavailable: ${String(err)}`;
  }
  if (!store.tabs.length) store.addTab();
  store.ready = true;
  // Serialize persistence calls so old snapshots can never overwrite newer ones.
  let queue = Promise.resolve();
  const save = () => {
    const config = snapshotConfig();
    queue = queue
      .then(() => SaveConfig(config))
      .catch((err) => {
        store.notice = `Could not save settings: ${String(err)}`;
      });
  };
  let timer: ReturnType<typeof setTimeout> | undefined;
  const stop = watch(
    snapshotConfig,
    () => {
      clearTimeout(timer);
      timer = setTimeout(save, 150);
    },
    { deep: true },
  );
  return () => {
    stop();
    clearTimeout(timer);
    save();
    return queue;
  };
}
