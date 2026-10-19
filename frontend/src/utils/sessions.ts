import { shallowReactive } from "vue";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { SearchAddon } from "@xterm/addon-search";
import { main } from "../../wailsjs/go/models";
import {
  AcknowledgeOutput,
  StartSessionWithOptions,
  KillSession,
  WriteToTerminal,
  ResizeTerminal,
} from "../../wailsjs/go/main/TerminalService";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import { SessionController } from "./sessionController";
import type { TerminalNode } from "./layout";
import type { ITheme } from "@xterm/xterm";

export interface TerminalView {
  controller: SessionController;
  term: Terminal;
  fit: FitAddon;
  search: SearchAddon;
  wrapper: HTMLDivElement;
  copyOnSelect: boolean;
  opened: boolean;
}
export const sessions = shallowReactive(new Map<string, TerminalView>());
let routeInput = (controller: SessionController, data: string) =>
  controller.send(data);
export function configureInputRouter(route: typeof routeInput): void {
  routeInput = route;
}
export function acquireSession(
  node: TerminalNode,
  theme: ITheme,
  fontSize: number,
): TerminalView {
  const existing = sessions.get(node.id);
  if (existing) return existing;
  const wrapper = document.createElement("div");
  wrapper.style.cssText = "width:100%;height:100%";
  const term = new Terminal({
    fontSize,
    theme,
    fontFamily: '"Anka/Coder", monospace',
    cursorBlink: true,
    allowProposedApi: true,
    scrollback: 10000,
  });
  const fit = new FitAddon(),
    search = new SearchAddon();
  term.loadAddon(fit);
  term.loadAddon(search);
  const controller = new SessionController(
    term,
    { shell: node.shell, cwd: node.cwd, ssh: node.ssh },
    {
      start: (id, cols, rows, options) =>
        StartSessionWithOptions(
          id,
          cols,
          rows,
          new main.SessionOptions(options),
        ),
      kill: KillSession,
      write: WriteToTerminal,
      resize: ResizeTerminal,
      ack: AcknowledgeOutput,
      on: EventsOn,
    },
    (owner, data) => routeInput(owner, data),
  );
  const view: TerminalView = {
    controller,
    term,
    fit,
    search,
    wrapper,
    copyOnSelect: false,
    opened: false,
  };
  term.attachCustomKeyEventHandler((e) => {
    const key = e.key.toLowerCase();
    if (e.ctrlKey && e.shiftKey && (key === "c" || key === "v")) {
      if (e.type === "keydown" && !e.repeat) {
        if (key === "c") {
          const text = term.getSelection();
          if (text)
            void navigator.clipboard
              .writeText(text)
              .catch((err) => controller.report(err));
        } else {
          void navigator.clipboard
            .readText()
            .then((text) => {
              if (controller.state.status === "running") term.paste(text);
            })
            .catch((err) => controller.report(err));
        }
      }
      e.preventDefault();
      return false;
    }
    if (
      e.ctrlKey &&
      (key === "f" ||
        key === "p" ||
        (e.shiftKey && ["t", "w", "x", "e", "o"].includes(key)))
    )
      return false;
    return true;
  });
  term.onSelectionChange(() => {
    if (view.copyOnSelect && term.hasSelection())
      void navigator.clipboard
        .writeText(term.getSelection())
        .catch((err) => controller.report(err));
  });
  sessions.set(node.id, view);
  return view;
}
export function disposeSession(paneId: string): void {
  const view = sessions.get(paneId);
  if (view) {
    view.controller.dispose();
    sessions.delete(paneId);
  }
}
