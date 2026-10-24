import { describe, expect, it, vi } from "vitest";
import {
  SessionController,
  SessionBridge,
  TerminalPort,
} from "./sessionController";
function fixture() {
  const listeners = new Map<string, (data: any) => void>();
  const writes: { data: string | Uint8Array; callback?: () => void }[] = [];
  let input = (_data: string) => {};
  let next = 0;
  const term: TerminalPort = {
    cols: 80,
    rows: 24,
    options: {},
    write: (data, callback) => writes.push({ data, callback }),
    onData: (handler) => {
      input = handler;
      return { dispose: vi.fn() };
    },
    dispose: vi.fn(),
  };
  const bridge: SessionBridge = {
    start: vi.fn().mockResolvedValue(undefined),
    kill: vi.fn().mockResolvedValue(undefined),
    write: vi.fn().mockResolvedValue(undefined),
    resize: vi.fn().mockResolvedValue(undefined),
    ack: vi.fn().mockResolvedValue(undefined),
    log: vi.fn().mockResolvedValue(undefined),
    on: (event, id, handler) => {
      const name = `terminal:${event}:${id}`;
      listeners.set(name, handler);
      return () => {
        listeners.delete(name);
      };
    },
  };
  const controller = new SessionController(
    term,
    { shell: "/bin/sh", cwd: "/tmp" },
    bridge,
    (owner, data) => owner.send(data),
    () => `session-${++next}`,
  );
  return {
    controller,
    bridge,
    term,
    writes,
    listeners,
    input: (data: string) => input(data),
  };
}
describe("session controller ownership", () => {
  it("subscribes before startup, acknowledges consumed bytes, and retains output after exit", async () => {
    const f = fixture();
    vi.mocked(f.bridge.start).mockImplementation(async () => {
      expect(f.listeners.has("terminal:data:session-1")).toBe(true);
    });
    await f.controller.start();
    f.listeners.get("terminal:data:session-1")!({
      sequence: 1,
      data: btoa(String.fromCharCode(0xf0, 0x9f)),
    });
    expect(f.writes[0].data).toEqual(new Uint8Array([0xf0, 0x9f]));
    expect(f.bridge.ack).not.toHaveBeenCalled();
    f.writes[0].callback!();
    await vi.waitFor(() =>
      expect(f.bridge.ack).toHaveBeenCalledWith("session-1", 1),
    );
    f.listeners.get("terminal:exit:session-1")!({ code: 7 });
    expect(f.controller.state.status).toBe("exited");
    expect(f.controller.state.code).toBe(7);
    expect(f.term.dispose).not.toHaveBeenCalled();
    await f.controller.start();
    expect(f.controller.sessionId).toBe("session-2");
    expect(f.listeners.has("terminal:data:session-1")).toBe(false);
    f.controller.dispose();
    expect(f.listeners.size).toBe(0);
  });
  it("kills startup completing after close and cannot restart after disposal", async () => {
    const f = fixture();
    let resolve!: () => void;
    vi.mocked(f.bridge.start).mockImplementation(
      () =>
        new Promise<void>((done) => {
          resolve = done;
        }),
    );
    const startup = f.controller.start();
    f.controller.dispose();
    resolve();
    await startup;
    expect(f.bridge.kill).toHaveBeenCalledWith("session-1");
    expect(f.listeners.size).toBe(0);
    await f.controller.start();
    expect(f.bridge.start).toHaveBeenCalledTimes(1);
  });
  it("shares concurrent startup and retries failures with a fresh identity", async () => {
    const f = fixture();
    vi.mocked(f.bridge.start).mockRejectedValueOnce(new Error("missing shell"));
    const start = f.controller.start();
    expect(f.controller.start()).toBe(start);
    await start;
    expect(f.controller.state.status).toBe("failed");
    expect(f.listeners.size).toBe(0);
    await f.controller.start();
    expect(f.controller.state.status).toBe("running");
    expect(f.controller.sessionId).toBe("session-2");
    f.controller.dispose();
  });
  it("serializes input with a stable destination", async () => {
    const f = fixture();
    await f.controller.start();
    f.input("first");
    f.input("second");
    await vi.waitFor(() => expect(f.bridge.write).toHaveBeenCalledTimes(2));
    expect(vi.mocked(f.bridge.write).mock.calls).toEqual([
      ["session-1", "first"],
      ["session-1", "second"],
    ]);
    f.controller.dispose();
    f.input("late");
    await Promise.resolve();
    expect(f.bridge.write).toHaveBeenCalledTimes(2);
  });
  it("queues typing during startup instead of losing it", async () => {
    const f = fixture();
    let resolve!: () => void;
    vi.mocked(f.bridge.start).mockImplementation(
      () =>
        new Promise<void>((done) => {
          resolve = done;
        }),
    );
    const startup = f.controller.start();
    f.input("early");
    await Promise.resolve();
    expect(f.bridge.write).not.toHaveBeenCalled();
    resolve();
    await startup;
    await vi.waitFor(() =>
      expect(f.bridge.write).toHaveBeenCalledWith("session-1", "early"),
    );
    f.controller.dispose();
  });
  it("rejects out-of-order output without acknowledging it", async () => {
    const f = fixture();
    await f.controller.start();
    f.listeners.get("terminal:data:session-1")!({ sequence: 4, data: "" });
    expect(f.bridge.kill).toHaveBeenCalledWith("session-1");
    expect(f.bridge.ack).not.toHaveBeenCalled();
    f.controller.dispose();
  });
  it("makes acknowledgement failures reconnectable", async () => {
    const f = fixture();
    await f.controller.start();
    vi.mocked(f.bridge.ack).mockRejectedValue(new Error("bridge down"));
    f.listeners.get("terminal:data:session-1")!({ sequence: 1, data: "YQ==" });
    f.writes[0].callback!();
    await vi.waitFor(() => expect(f.controller.state.status).toBe("failed"));
    expect(f.listeners.size).toBe(0);
    expect(f.bridge.kill).toHaveBeenCalledWith("session-1");
    await f.controller.start();
    expect(f.controller.state.status).toBe("running");
    f.controller.dispose();
  });
  it("recovers from malformed bytes and keeps output history", async () => {
    const f = fixture();
    await f.controller.start();
    f.listeners.get("terminal:data:session-1")!({
      sequence: 1,
      data: "invalid%%%",
    });
    expect(f.controller.state.status).toBe("failed");
    expect(f.term.dispose).not.toHaveBeenCalled();
    f.controller.dispose();
  });
  it("owns logging state and preserves errors from failed log operations", async () => {
    const f = fixture();
    await f.controller.start();
    await f.controller.startLogging("/tmp/session.log");
    expect(f.controller.state.logging).toBe(true);
    vi.mocked(f.bridge.log).mockRejectedValueOnce(new Error("cannot stop"));
    await expect(f.controller.stopLogging()).rejects.toThrow();
    expect(f.controller.state.logging).toBe(true);
    await f.controller.stopLogging();
    expect(f.controller.state.logging).toBe(false);
    f.controller.dispose();
  });

  it("makes a hung acknowledgement RPC recoverable", async () => {
    vi.useFakeTimers();
    const f = fixture();
    await f.controller.start();
    vi.mocked(f.bridge.ack).mockImplementation(() => new Promise(() => {}));
    f.listeners.get("terminal:data:session-1")!({ sequence: 1, data: "YQ==" });
    f.writes[0].callback!();
    await vi.advanceTimersByTimeAsync(5000);
    expect(f.controller.state.status).toBe("failed");
    expect(f.bridge.kill).toHaveBeenCalledWith("session-1");
    f.controller.dispose();
    vi.useRealTimers();
  });
  it("does not overwrite an immediate logging error with a late RPC success", async () => {
    const f = fixture();
    await f.controller.start();
    let resolve!: () => void;
    vi.mocked(f.bridge.log).mockImplementation(
      () =>
        new Promise<void>((done) => {
          resolve = done;
        }),
    );
    const start = f.controller.startLogging("/tmp/log");
    f.listeners.get("terminal:log-error:session-1")!("disk full");
    resolve();
    await start;
    expect(f.controller.state.logging).toBe(false);
    f.controller.dispose();
  });
  it("does not let a stuck old write block input after reconnect", async () => {
    const f = fixture();
    await f.controller.start();
    let finish!: () => void;
    vi.mocked(f.bridge.write).mockImplementationOnce(
      () =>
        new Promise<void>((done) => {
          finish = done;
        }),
    );
    f.input("old");
    await vi.waitFor(() => expect(f.bridge.write).toHaveBeenCalledTimes(1));
    f.listeners.get("terminal:data:session-1")!({ sequence: 99, data: "" });
    await f.controller.start();
    f.input("new");
    await vi.waitFor(() =>
      expect(f.bridge.write).toHaveBeenCalledWith("session-2", "new"),
    );
    finish();
    f.controller.dispose();
  });
});
