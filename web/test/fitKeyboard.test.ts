import { describe, expect, test } from "bun:test";
import { isFitShortcutHelpKey } from "../app/utils/fitKeyboard";

describe("fitting shortcut help key", () => {
    test("accepts a question mark reported directly by the browser", () => {
        expect(isFitShortcutHelpKey({ key: "?", code: "Slash", shiftKey: true })).toBe(true);
    });

    test("accepts shifted plus on layouts where that produces a question mark", () => {
        expect(isFitShortcutHelpKey({ key: "+", code: "Minus", shiftKey: true })).toBe(true);
        expect(isFitShortcutHelpKey({ key: "+", code: "Equal", shiftKey: true })).toBe(true);
    });

    test("does not treat an unshifted plus or shifted minus as help", () => {
        expect(isFitShortcutHelpKey({ key: "+", code: "Minus", shiftKey: false })).toBe(false);
        expect(isFitShortcutHelpKey({ key: "_", code: "Minus", shiftKey: true })).toBe(false);
    });
});
