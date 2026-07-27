/**
 * Tri-state viewport breakpoint for SSR-safe dual desktop/mobile trees.
 *
 * Several views render BOTH a desktop and a mobile variant and hide one with
 * CSS (`hidden md:block` / `md:hidden`) — the server can't know the viewport,
 * and the hydration render must match the server HTML, so both variants must
 * exist initially. But keeping both mounted doubles the live DOM, component
 * instances, and watcher count for the page's whole lifetime.
 *
 * This composable returns `null` during SSR and hydration ("render both"),
 * then resolves to true/false after mount so `v-if` can unmount the inactive
 * variant. Usage:
 *
 *   const isDesktop = useIsDesktop()
 *   <div class="hidden md:block" v-if="isDesktop !== false">…desktop…</div>
 *   <div class="md:hidden"       v-if="isDesktop !== true">…mobile…</div>
 *
 * The CSS classes stay — they do the hiding before the tri-state resolves,
 * and keep doing it if JS never runs.
 */

// Module-level so every caller shares one matchMedia listener per query.
const installedQueries = new Set<string>()

export function useIsDesktop(query = '(min-width: 768px)'): Readonly<Ref<boolean | null>> {
    const state = useState<boolean | null>(`is-desktop:${query}`, () => null)

    if (import.meta.client && !installedQueries.has(query)) {
        installedQueries.add(query)
        onMounted(() => {
            const mq = window.matchMedia(query)
            state.value = mq.matches
            // App-lifetime listener by design — the state is shared globally.
            mq.addEventListener('change', (e) => { state.value = e.matches })
        })
    }

    return readonly(state)
}
