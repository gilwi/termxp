import { main } from "../../wailsjs/go/models";
import * as terminal from "../../wailsjs/go/main/TerminalService";
import { GetSystemStats, ConfirmClose } from "../../wailsjs/go/main/App";
import * as config from "../../wailsjs/go/main/ConfigService";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import type {
  Config,
  ConfigResult,
  SessionBridge,
  SessionEvents,
} from "./domain";
export const settingsBridge = {
  load: async (): Promise<ConfigResult> => {
    const result = await config.LoadConfig();
    // Clone across the boundary: the domain never holds generated model methods.
    return JSON.parse(JSON.stringify(result)) as ConfigResult;
  },
  save: (value: Config) => config.SaveConfig(new main.Config(value)),
  recover: (value: Config) => config.RecoverConfig(new main.Config(value)),
  shells: () => terminal.ListShells(),
};
export const terminalBridge: SessionBridge = {
  start: (id, cols, rows, options) =>
    terminal.StartSessionWithOptions(
      id,
      cols,
      rows,
      new main.SessionOptions(options),
    ),
  kill: terminal.KillSession,
  write: terminal.WriteToTerminal,
  resize: terminal.ResizeTerminal,
  ack: terminal.AcknowledgeOutput,
  log: terminal.LogSession,
  on: <K extends keyof SessionEvents>(
    event: K,
    id: string,
    callback: (data: SessionEvents[K]) => void,
  ) =>
    EventsOn(`terminal:${event}:${id}`, (data: unknown) =>
      callback(data as SessionEvents[K]),
    ),
};

export const applicationBridge = {
  stats: GetSystemStats,
  confirmClose: ConfirmClose,
};
