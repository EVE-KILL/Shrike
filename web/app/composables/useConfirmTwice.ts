/**
 * Arm-then-confirm for destructive row actions: the first click arms the
 * button (so the label can flip to "Sure?"), a second click within the window
 * goes through, and the arming lapses on its own if the user walks away.
 *
 * Replaces `window.confirm()`, which cannot be styled and blocks the page.
 *
 *     const { pendingId, confirm } = useConfirmTwice()
 *
 *     async function deleteRow(id: number) {
 *         if (!confirm(id)) return    // first click — armed, nothing else to do
 *         await apiFetch(...)
 *     }
 *
 * Arming a different row re-arms rather than firing, so clicking delete on row
 * A then row B cannot delete B on what the user experienced as a first click.
 */
export function useConfirmTwice<T extends string | number = number>(timeoutMs = 5000) {
    const pendingId = ref<T | null>(null)
    let timer: ReturnType<typeof setTimeout> | null = null

    const clear = () => {
        if (timer) { clearTimeout(timer); timer = null }
    }

    /** True when this is the confirming (second) click and the caller should proceed. */
    const confirm = (id: T): boolean => {
        if (pendingId.value !== id) {
            pendingId.value = id
            clear()
            timer = setTimeout(() => {
                if (pendingId.value === id) pendingId.value = null
            }, timeoutMs)
            return false
        }
        pendingId.value = null
        clear()
        return true
    }

    /** Disarm without acting — for closing a panel or cancelling explicitly. */
    const reset = () => {
        pendingId.value = null
        clear()
    }

    onUnmounted(clear)

    return { pendingId, confirm, reset }
}
