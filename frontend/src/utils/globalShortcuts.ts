import { actions } from "./actions";
import { findNode } from "./layout";
import { shortcutAction } from "./shortcuts";
import { store } from "./store";

export function handleGlobalKeyDown(event: KeyboardEvent): void {
  if (store.paletteOpen || !store.ready) return;
  const action = shortcutAction(event, store.shortcuts);
  if (!action) return;
  event.preventDefault();
  if (event.repeat && (action === "close" || action === "close-tab")) return;

  // The event's terminal is authoritative even if focus tracking is stale.
  const paneId = event.target instanceof Element
    ? event.target.closest("[data-pane-id]")?.getAttribute("data-pane-id")
    : null;
  if (paneId) {
    const tab = store.getActiveTab();
    if (!tab || findNode(tab.rootNode, paneId)?.node.type !== "terminal") return;
    store.activePaneId = paneId;
    tab.focusedPaneId = paneId;
  }

  if (action === "palette") store.paletteOpen = true;
  else actions().find((item) => item.id === action)?.run();
}
