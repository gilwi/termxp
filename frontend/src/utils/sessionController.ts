import { shallowReactive } from "vue";
import type { ITheme } from "@xterm/xterm";
import type { SessionOptions, SessionBridge } from "./domain";
export type { SessionOptions, SessionBridge } from "./domain";

export interface TerminalPort {
  cols: number;
  rows: number;
  options: { theme?: ITheme; fontSize?: number };
  write(data: Uint8Array | string, callback?: () => void): void;
  onData(callback: (data: string) => void): { dispose(): void };
  dispose(): void;
}
export type SessionStatus =
  "idle" | "starting" | "running" | "exited" | "failed" | "closing";

// This owner outlives Vue views. No callback captures a mounted component.
export class SessionController {
  readonly state = shallowReactive({
    status: "idle" as SessionStatus,
    cwd: "",
    code: null as number | null,
    error: "",
    logging: false,
    searchOpen: false,
  });
  sessionId = "";
  private disposed = false;
  private subscriptions: (() => void)[] = [];
  private startup: Promise<void> | null = null;
  private input: { dispose(): void };
  private inputQueue = { tail: Promise.resolve() as Promise<void>, pending: 0 };
  private nextSequence = 1;
  private logRevision = 0;
  constructor(
    readonly term: TerminalPort,
    readonly options: SessionOptions,
    private bridge: SessionBridge,
    route: (controller: SessionController, data: string) => void,
    private uuid: () => string = () => crypto.randomUUID(),
  ) {
    this.state.cwd = options.cwd;
    this.input = term.onData((data) => route(this, data));
  }
  start(): Promise<void> {
    if (this.disposed) return Promise.resolve();
    if (this.startup) return this.startup;
    if (this.state.status === "running") return Promise.resolve();
    this.clearSubscriptions();
    const id = (this.sessionId = this.uuid());
    this.nextSequence = 1;
    this.inputQueue = { tail: Promise.resolve(), pending: 0 };
    this.state.status = "starting";
    this.state.error = "";
    this.state.code = null;
    this.subscriptions = [
      this.bridge.on(
        "data",
        id,
        (event: { sequence: number; data: string }) => {
          if (this.disposed || id !== this.sessionId) return;
          if (
            !event ||
            !Number.isSafeInteger(event.sequence) ||
            typeof event.data !== "string" ||
            event.data.length > 43692 ||
            event.sequence !== this.nextSequence
          ) {
            this.fail(id, "Terminal output arrived out of order");
            return;
          }
          this.nextSequence++;
          try {
            const bytes = Uint8Array.from(atob(event.data), (c) =>
              c.charCodeAt(0),
            );
            this.term.write(bytes, () => {
              if (!this.disposed)
                void this.acknowledge(id, event.sequence).catch((err) =>
                  this.fail(
                    id,
                    `Output acknowledgement failed: ${String(err)}`,
                  ),
                );
            });
          } catch (err) {
            this.fail(id, `Invalid terminal output: ${String(err)}`);
          }
        },
      ),
      this.bridge.on("exit", id, (event: { code: number }) => {
        if (this.state.status === "failed" || id !== this.sessionId) return;
        this.logRevision++;
        this.state.status = "exited";
        this.state.code = event.code;
        this.state.logging = false;
      }),
      this.bridge.on("cwd", id, (cwd: string) => {
        this.state.cwd = cwd;
      }),
      this.bridge.on("log-error", id, (error: string) => {
        this.logRevision++;
        this.state.logging = false;
        this.state.error = `Logging stopped: ${error}`;
      }),
    ];
    this.startup = this.bridge
      .start(id, this.term.cols || 80, this.term.rows || 24, {
        ...this.options,
        cwd: this.state.cwd || this.options.cwd,
      })
      .then(async () => {
        if (this.disposed) {
          await this.bridge.kill(id);
          return;
        }
        if (this.state.status === "starting") this.state.status = "running";
        await this.resize();
      })
      .catch(async (err) => {
        if (!this.disposed) {
          this.state.status = "failed";
          this.report(err);
          this.clearSubscriptions();
        }
      })
      .finally(() => {
        this.startup = null;
      });
    return this.startup;
  }
  send(data: string): void {
    if (this.disposed || !["starting", "running"].includes(this.state.status))
      return;
    const queue = this.inputQueue,
      bytes = new TextEncoder().encode(data).length;
    if (queue.pending + bytes > 1024 * 1024) {
      this.state.error = "Input queue is full; wait before pasting again";
      return;
    }
    // Serialize keypress/paste writes. A failed write cannot poison future input.
    const id = this.sessionId;
    queue.pending += bytes;
    queue.tail = queue.tail
      .then(async () => {
        if (this.state.status === "starting") await this.startup;
        if (
          !this.disposed &&
          id === this.sessionId &&
          this.state.status === "running"
        )
          await this.bridge.write(id, data);
      })
      .catch((err) => {
        if (id === this.sessionId) this.report(err);
      })
      .finally(() => {
        queue.pending -= bytes;
      });
  }
  async resize(): Promise<void> {
    if (!this.disposed && this.state.status === "running") {
      try {
        await this.bridge.resize(
          this.sessionId,
          this.term.cols,
          this.term.rows,
        );
      } catch (err) {
        this.report(err);
      }
    }
  }
  private async acknowledge(id: string, sequence: number): Promise<void> {
    let timer: ReturnType<typeof setTimeout> | undefined;
    try {
      await Promise.race([
        Promise.resolve().then(() => this.bridge.ack(id, sequence)),
        new Promise<never>((_, reject) => {
          timer = setTimeout(
            () => reject(new Error("acknowledgement timed out")),
            5000,
          );
        }),
      ]);
    } finally {
      clearTimeout(timer);
    }
  }
  private fail(id: string, error: string): void {
    if (this.disposed || this.sessionId !== id) return;
    this.state.status = "failed";
    this.state.error = error;
    this.state.logging = false;
    this.clearSubscriptions();
    void this.bridge.kill(id).catch((err) => this.report(err));
  }
  async startLogging(path: string): Promise<void> {
    if (!path) throw new Error("Choose a new log file");
    if (this.state.status !== "running")
      throw new Error("Terminal is not running");
    const id = this.sessionId,
      revision = this.logRevision;
    await this.bridge.log(id, path);
    if (
      !this.disposed &&
      this.sessionId === id &&
      this.state.status === "running" &&
      revision === this.logRevision
    )
      this.state.logging = true;
  }
  async stopLogging(): Promise<void> {
    const id = this.sessionId;
    await this.bridge.log(id, "");
    if (this.sessionId === id) this.state.logging = false;
  }
  updateSSH(profile: SessionOptions["ssh"]): void {
    this.options.ssh = profile ? { ...profile } : undefined;
  }
  useShell(shell: string): void {
    this.options.shell = shell;
  }
  useDirectory(directory: string): void {
    this.state.cwd = directory;
  }
  dismissError(): void {
    this.state.error = "";
  }
  toggleSearch(open = !this.state.searchOpen): void {
    this.state.searchOpen = open;
  }
  report(err: unknown): void {
    this.state.error = String(err);
  }
  private clearSubscriptions(): void {
    this.subscriptions.splice(0).forEach((off) => off());
  }
  dispose(): void {
    if (this.disposed) return;
    this.disposed = true;
    this.state.status = "closing";
    this.clearSubscriptions();
    this.input.dispose();
    this.term.dispose();
    if (this.sessionId) void this.bridge.kill(this.sessionId).catch(() => {});
    // start() also kills after resolution, covering cancellation before registration.
  }
}
