// One queue for autosave, explicit edits, recovery, and final close. Snapshots
// are taken when the queued operation executes, never before another edit.
export class PersistenceCoordinator<T> {
  private tail: Promise<void> = Promise.resolve();
  private timer: ReturnType<typeof setTimeout> | undefined;
  blocked = false;
  constructor(
    private snapshot: () => T,
    private write: (value: T) => Promise<void>,
    private report: (error: unknown) => void,
  ) {}
  private enqueue(operation: () => Promise<void>): Promise<void> {
    const result = this.tail.catch(() => {}).then(operation);
    this.tail = result;
    return result;
  }
  save(): Promise<void> {
    clearTimeout(this.timer);
    return this.enqueue(async () => {
      if (this.blocked)
        throw new Error("Configuration recovery is required before saving");
      try {
        await this.write(this.snapshot());
      } catch (err) {
        this.report(err);
        throw err;
      }
    });
  }
  edit(change: () => void | (() => void)): Promise<void> {
    return this.enqueue(async () => {
      if (this.blocked)
        throw new Error(
          "Recover the configuration before editing saved preferences",
        );
      const undo = change();
      try {
        await this.write(this.snapshot());
      } catch (err) {
        if (typeof undo === "function") undo();
        this.report(err);
        throw err;
      }
    });
  }
  schedule(): void {
    clearTimeout(this.timer);
    if (!this.blocked)
      this.timer = setTimeout(() => {
        void this.save().catch(() => {});
      }, 150);
  }
  recover(replace: (value: T) => Promise<unknown>): Promise<void> {
    clearTimeout(this.timer);
    return this.enqueue(async () => {
      await replace(this.snapshot());
      this.blocked = false;
    });
  }
  cancelScheduled(): void {
    clearTimeout(this.timer);
  }
}
