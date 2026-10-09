type ShortcutKeyEvent = Pick<KeyboardEvent, "key" | "code" | "shiftKey">;

/**
 * KeyboardEvent.key normally contains `?`, but some non-US layouts report
 * the shifted physical plus key instead. Accept both representations so the
 * fitting help shortcut follows the character printed by the user's layout.
 */
export function isFitShortcutHelpKey(event: ShortcutKeyEvent): boolean {
    return event.key === "?"
        || (event.shiftKey && event.key === "+" && (event.code === "Minus" || event.code === "Equal"));
}
