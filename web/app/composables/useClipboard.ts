/**
 * Thin wrapper around `navigator.clipboard.writeText` with a
 * transient "copied" flag that auto-resets after a timeout. Used
 * by the FitNavbar to flash "Copied!" on the button for a moment
 * after a successful paste.
 *
 * Returns a separate `{ copied, copy }` per call so each button
 * can have its own feedback state without stepping on its
 * neighbors.
 */

export function useClipboardFeedback(timeoutMs = 1800) {
    const copied = ref(false);
    let timer: ReturnType<typeof setTimeout> | null = null;

    async function copy(text: string): Promise<boolean> {
        if (typeof navigator === "undefined" || !navigator.clipboard) return false;
        try {
            await navigator.clipboard.writeText(text);
            copied.value = true;
            if (timer) clearTimeout(timer);
            timer = setTimeout(() => {
                copied.value = false;
            }, timeoutMs);
            return true;
        } catch {
            // Clipboard API rejects for permission reasons (e.g.
            // user hasn't interacted with the page yet in Firefox).
            return false;
        }
    }

    async function readText(): Promise<string | null> {
        if (typeof navigator === "undefined" || !navigator.clipboard) return null;
        try {
            return await navigator.clipboard.readText();
        } catch {
            return null;
        }
    }

    return { copied, copy, readText };
}
