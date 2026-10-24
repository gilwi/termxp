import { describe, it, expect, vi } from "vitest";
import { PersistenceCoordinator } from "./persistence";
const deferred = () => {
  let resolve!: () => void;
  let reject!: (err: unknown) => void;
  const promise = new Promise<void>((yes, no) => {
    resolve = yes;
    reject = no;
  });
  return { promise, resolve, reject };
};
describe("serialized persistence", () => {
  it("propagates final-save failure and can retry", async () => {
    const write = vi
      .fn()
      .mockRejectedValueOnce(new Error("disk full"))
      .mockResolvedValue(undefined);
    const report = vi.fn(),
      coordinator = new PersistenceCoordinator(
        () => ({ theme: "dark" }),
        write,
        report,
      );
    await expect(coordinator.save()).rejects.toThrow("disk full");
    await expect(coordinator.save()).resolves.toBeUndefined();
    expect(report).toHaveBeenCalledTimes(1);
  });
  it("snapshots at execution time so queued autosave cannot undo a profile edit", async () => {
    const first = deferred(),
      values: string[] = [],
      state = { profile: "old", theme: "old" };
    const write = vi.fn(async (value: { profile: string; theme: string }) => {
      values.push(`${value.profile}:${value.theme}`);
      if (values.length === 1) await first.promise;
    });
    const coordinator = new PersistenceCoordinator(
      () => ({ ...state }),
      write,
      vi.fn(),
    );
    const initial = coordinator.save();
    await Promise.resolve();
    const edit = coordinator.edit(() => {
      state.profile = "new";
    });
    const queued = coordinator.save();
    state.theme = "new";
    first.resolve();
    await Promise.all([initial, edit, queued]);
    expect(values.at(-1)).toBe("new:new");
  });
  it("rolls back only the failed edit and preserves other preferences", async () => {
    const pending = deferred(),
      state = { profile: "old", theme: "old" };
    const coordinator = new PersistenceCoordinator(
      () => ({ ...state }),
      () => pending.promise,
      vi.fn(),
    );
    const edit = coordinator.edit(() => {
      state.profile = "new";
      return () => {
        state.profile = "old";
      };
    });
    await Promise.resolve();
    await Promise.resolve();
    state.theme = "new";
    pending.reject(new Error("invalid profile"));
    await expect(edit).rejects.toThrow();
    expect(state).toEqual({ profile: "old", theme: "new" });
  });
  it("blocks autosave until explicit recovery succeeds", async () => {
    vi.useFakeTimers();
    const write = vi.fn().mockResolvedValue(undefined),
      coordinator = new PersistenceCoordinator(
        () => ({ version: 1 }),
        write,
        vi.fn(),
      );
    coordinator.blocked = true;
    coordinator.schedule();
    await vi.runAllTimersAsync();
    expect(write).not.toHaveBeenCalled();
    await expect(coordinator.save()).rejects.toThrow("recovery");
    const recover = vi.fn().mockResolvedValue("backup");
    await coordinator.recover(recover);
    await coordinator.save();
    expect(write).toHaveBeenCalledTimes(1);
    vi.useRealTimers();
  });
});
