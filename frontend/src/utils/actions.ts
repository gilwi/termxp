import { store } from "./store";
import { recoverSettings } from "./settings";
import { themes } from "./themes";
import { sessions } from "./sessions";
export interface Action {
  id: string;
  label: string;
  run(): void;
}
export function actions(): Action[] {
  return [
    ...(store.recoveryRequired
      ? [
          {
            id: "recover",
            label: "Recover configuration with backup",
            run: () => {
              void recoverSettings().catch((err) => {
                store.notice = String(err);
              });
            },
          },
        ]
      : []),
    { id: "new", label: "New shell tab", run: () => store.addTab() },
    {
      id: "close",
      label: "Close current tab",
      run: () => store.closeTab(store.activeTabId),
    },
    {
      id: "right",
      label: "Split right",
      run: () => store.splitPane(store.activePaneId, "vertical"),
    },
    {
      id: "below",
      label: "Split below",
      run: () => store.splitPane(store.activePaneId, "horizontal"),
    },
    {
      id: "right-same",
      label: "Split right with same connection",
      run: () => store.splitPane(store.activePaneId, "vertical", "same"),
    },
    {
      id: "below-same",
      label: "Split below with same connection",
      run: () => store.splitPane(store.activePaneId, "horizontal", "same"),
    },
    {
      id: "maximize",
      label: "Toggle maximized pane",
      run: () => store.toggleMaximize(),
    },
    {
      id: "search",
      label: "Search terminal scrollback",
      run: () => {
        const c = sessions.get(store.activePaneId)?.controller;
        if (c) c.toggleSearch(true);
      },
    },
    {
      id: "reconnect",
      label: "Reconnect exited shell",
      run: () => {
        const c = sessions.get(store.activePaneId)?.controller;
        if (c && ["exited", "failed"].includes(c.state.status)) void c.start();
      },
    },
    {
      id: "broadcast",
      label: "Toggle broadcast input to this tab",
      run: () => {
        store.broadcast = !store.broadcast;
      },
    },
    {
      id: "sidebar",
      label: "Toggle settings sidebar",
      run: () => {
        store.sidebarOpen = !store.sidebarOpen;
      },
    },
    ...Object.entries(themes).map(([id, theme]) => ({
      id: `theme:${id}`,
      label: `Theme: ${theme.name}`,
      run: () => {
        store.currentTheme = id;
      },
    })),
    ...store.tabs.map((tab) => ({
      id: `tab:${tab.id}`,
      label: `Switch tab: ${tab.name}`,
      run: () => store.selectTab(tab.id),
    })),
    ...store.sshProfiles.map((profile) => ({
      id: `ssh:${profile.id}`,
      label: `SSH: ${profile.name}`,
      run: () => store.addTab(profile),
    })),
    ...store.snippets.map((s) => ({
      id: `snippet:${s.id}`,
      label: `Run: ${s.label}`,
      run: () => store.runSnippet(s.command),
    })),
  ];
}
export function fuzzyMatch(query: string, text: string): boolean {
  let i = 0;
  for (const char of text.toLowerCase())
    if (char === query.toLowerCase()[i]) i++;
  return i === query.length;
}
