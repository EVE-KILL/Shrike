// Global paste handler — anywhere on the site, if the user pastes a string
// containing one or more ESI killmail URLs, intercept it and dispatch the
// killmails through /api/killmail/post. Ignores pastes inside text inputs,
// textareas, and contenteditable elements so normal typing isn't disrupted.

const ESI_KILLMAIL_RE = /https?:\/\/esi\.evetech\.net\/(?:latest\/|v\d+\/)?killmails\/(\d+)\/([a-f0-9]{20,64})\/?/i

function isEditableTarget(target: EventTarget | null): boolean {
    if (!(target instanceof HTMLElement)) return false
    const tag = target.tagName
    if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return true
    if (target.isContentEditable) return true
    return false
}

export default defineNuxtPlugin(() => {
    if (import.meta.server) return

    const toast = useToast()
    const watcher = useKillWatcher()

    document.addEventListener('paste', async (ev) => {
        if (isEditableTarget(ev.target)) return

        const text = ev.clipboardData?.getData('text/plain')?.trim()
        if (!text) return
        if (!ESI_KILLMAIL_RE.test(text)) return

        ev.preventDefault()

        try {
            const res = await apiFetch<{ accepted: number; rejected: number; existing: number; total: number; killmails: number[]; existingIds: number[] }>(
                '/api/killmail/post',
                { method: 'POST', body: { text } },
            )
            if (res.total === 0) {
                toast.info('No killmail links found in clipboard')
                return
            }
            // Existing single — jump straight there
            if (res.total === 1 && res.existing === 1) {
                const id = res.killmails[0]!
                toast.info(`Killmail #${id} already exists`)
                await navigateTo(`/kill/${id}`)
                return
            }
            // Watch any newly queued killmails
            const newIds = res.killmails.filter(id => !res.existingIds.includes(id))
            if (newIds.length > 0) watcher.watch(newIds)

            const parts: string[] = []
            if (res.accepted) parts.push(`${res.accepted} queued — watching for arrival`)
            if (res.existing) parts.push(`${res.existing} already exist`)
            if (res.rejected) parts.push(`${res.rejected} failed`)
            const msg = parts.join(', ')
            if (res.rejected) toast.error(msg)
            else toast.success(msg)
        } catch (err: any) {
            toast.error(extractFetchError(err, 'Failed to queue killmail'))
        }
    }, { capture: true })
})
