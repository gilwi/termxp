import { shallowReactive } from "vue";
import type { ITheme } from "@xterm/xterm";
import type { main } from "../../wailsjs/go/models";

export type SessionOptions = Pick<main.SessionOptions, "shell" | "cwd" | "ssh">;

export interface TerminalPort {
  cols: number;
  rows: number;
  options: { theme?: ITheme; fontSize?: number };
  write(data: Uint8Array | string, callback?: () => void): void;
  onData(callback: (data: string) => void): { dispose(): void };
  dispose(): void;
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
  on(name: string, callback: (data: any) => void): () => void;
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
  private writes: Promise<void> = Promise.resolve();
  private pendingInput = 0;
  private nextSequence = 1;
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
    this.state.status = "starting";
    this.state.error = "";
    this.state.code = null;
    this.subscriptions = [
      this.bridge.on(
        `terminal:data:${id}`,
        (event: { sequence: number; data: string }) => {
          if (this.disposed || id !== this.sessionId) return;
          if (event.sequence !== this.nextSequence) {
            this.state.error = "Terminal output arrived out of order";
            void this.bridge.kill(id).catch((err) => this.report(err));
            return;
          }
          this.nextSequence++;
          const bytes = Uint8Array.from(atob(event.data), (c) =>
            c.charCodeAt(0),
          );
          this.term.write(bytes, () => {
            if (!this.disposed)
              void this.bridge
                .ack(id, event.sequence)
                .catch((err) => this.report(err));
          });
        },
      ),
      this.bridge.on(`terminal:exit:${id}`, (event: { code: number }) => {
        this.state.status = "exited";
        this.state.code = event.code;
        this.state.logging = false;
      }),
      this.bridge.on(`terminal:cwd:${id}`, (cwd: string) => {
        this.state.cwd = cwd;
      }),
      this.bridge.on(`terminal:log-error:${id}`, (error: string) => {
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
    if (this.pendingInput + data.length > 1024 * 1024) {
      this.state.error = "Input queue is full; wait before pasting again";
      return;
    }
    // Serialize keypress/paste writes. A failed write cannot poison future input.
    const id = this.sessionId;
    this.pendingInput += data.length;
    this.writes = this.writes
      .then(async () => {
        if (this.state.status === "starting") await this.startup;
        if (
          !this.disposed &&
          id === this.sessionId &&
          this.state.status === "running"
        )
          await this.bridge.write(id, data);
      })
      .catch((err) => this.report(err))
      .finally(() => {
        this.pendingInput -= data.length;
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
