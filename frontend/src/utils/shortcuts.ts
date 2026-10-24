export function shortcutForEvent(
  event: Pick<
    KeyboardEvent,
    "ctrlKey" | "altKey" | "shiftKey" | "metaKey" | "key"
  >,
): string {
  if (event.metaKey || !/^[a-z0-9]$/i.test(event.key)) return "";
  return `${event.ctrlKey ? "Ctrl+" : ""}${event.altKey ? "Alt+" : ""}${event.shiftKey ? "Shift+" : ""}${event.key.toUpperCase()}`;
}
export function shortcutAction(
  event: Pick<
    KeyboardEvent,
    "ctrlKey" | "altKey" | "shiftKey" | "metaKey" | "key"
  >,
  shortcuts: Record<string, string>,
): string | undefined {
  const key = shortcutForEvent(event);
  return key
    ? Object.entries(shortcuts).find(
        ([, value]) => value !== "" && value === key,
      )?.[0]
    : undefined;
}
