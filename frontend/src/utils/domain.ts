// Application contracts: generated Wails models are confined to bridge.ts.
export interface SSHProfile {
  id?: string;
  name: string;
  host: string;
  user: string;
  port: number;
  keyPath: string;
  jumpHost: string;
}
export interface SessionOptions {
  shell: string;
  cwd: string;
  ssh?: SSHProfile;
}
export interface WorkspaceNode {
  type: "terminal" | "split";
  orientation?: "horizontal" | "vertical";
  children?: WorkspaceNode[];
  sizes?: number[];
  shell?: string;
  cwd?: string;
  ssh?: SSHProfile;
  sshProfileId?: string;
}
export interface WorkspaceTab {
  name: string;
  root: WorkspaceNode;
}
export interface Snippet {
  id: string;
  label: string;
  command: string;
}
export interface Config {
  version: number;
  theme: string;
  fontSize: number;
  defaultShell: string;
  copyOnSelect: boolean;
  restoreWorkspace: boolean;
  workspace: WorkspaceTab[];
  sshProfiles: SSHProfile[];
  snippets: Snippet[];
  shortcuts: Record<string, string>;
}
export interface ConfigResult {
  config: Config;
  warning: string;
  recoveryRequired: boolean;
}
export interface SessionEvents {
  data: { sequence: number; data: string };
  exit: { code: number };
  cwd: string;
  "log-error": string;
}
export interface SessionBridge {
  start(
    id: string,
    cols: number,
    rows: number,
    options: SessionOptions,
  ): Promise<void>;
  kill(id: string): Promise<void>;
  write(id: string, data: string): Promise<void>;
  resize(id: string, cols: number, rows: number): Promise<void>;
  ack(id: string, sequence: number): Promise<void>;
  log(id: string, path: string): Promise<void>;
  on<K extends keyof SessionEvents>(
    event: K,
    id: string,
    callback: (data: SessionEvents[K]) => void,
  ): () => void;
}
export const defaultShortcuts: Record<string, string> = {
  new: "Ctrl+Shift+T",
  close: "Ctrl+Shift+W",
  right: "Ctrl+Shift+E",
  below: "Ctrl+Shift+O",
  maximize: "Ctrl+Shift+X",
  search: "Ctrl+F",
  palette: "Ctrl+P",
};
export const defaultSnippets: Snippet[] = [
  { id: "list", label: "List Files", command: "ls -lah" },
  { id: "system", label: "System Info", command: "uname -a" },
  { id: "disk", label: "Disk Space", command: "df -h" },
  { id: "network", label: "Active Network", command: "ss -tulpn" },
  { id: "cpu", label: "CPU Load Check", command: "cat /proc/loadavg" },
  { id: "who", label: "Who Am I", command: "whoami && pwd" },
];
