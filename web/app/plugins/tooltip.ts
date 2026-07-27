import {
    TOOLTIP_BUBBLE_CLASS,
    clampTooltipCentre,
    resolveTooltipPosition,
} from '~/utils/tooltipBubble'

/**
 * `v-tooltip` — a styled replacement for the native `title` attribute.
 *
 * Native `title` takes about a second to appear, cannot be styled, and never
 * shows on touch or from the keyboard. The Tooltip component fixes that but
 * wraps its trigger in an element, which is fine for a handful of deliberate
 * placements and unsafe as a bulk replacement: wrapping a flex or grid child
 * changes the layout, and passing a display class to that wrapper does not
 * reliably fix it (see the `block` prop on Tooltip.vue).
 *
 * This directive attaches to the element that already exists, so a `title` can
 * be swapped for a `v-tooltip` without touching the surrounding markup.
 *
 * Usage:
 *   <button v-tooltip="'Report comment'">      (was title="Report comment")
 *   <span v-tooltip="formatEveDateTime(x)">    (was :title="...")
 *   <span v-tooltip="{ text: t, placement: 'top' }">
 *
 * Registered on the server as well as the client, even though the bubble itself
 * is client-only. A directive used in an SSR-rendered template must exist during
 * SSR: Vue's server renderer reads `getSSRProps` off it and throws
 * "Cannot read properties of undefined (reading 'getSSRProps')" if the directive
 * is unregistered, which 500s the page. The client merely warns, so registering
 * this as a `.client` plugin looks fine in dev navigation and breaks every
 * server-rendered route.
 *
 * A falsy value is a no-op, so `v-tooltip="maybeEmpty"` is safe.
 */

interface TooltipOptions {
    text?: string | null
    placement?: 'top' | 'bottom'
    delay?: number
}

interface TooltipState {
    text: string
    placement: 'top' | 'bottom'
    delay: number
    bubble: HTMLElement | null
    timer: ReturnType<typeof setTimeout> | null
    show: () => void
    hide: () => void
    onEnter: () => void
    onLeave: () => void
    onFocus: () => void
    onKeydown: (e: KeyboardEvent) => void
}

const states = new WeakMap<HTMLElement, TooltipState>()

const normalise = (value: unknown): TooltipOptions => {
    if (value == null) return {}
    if (typeof value === 'string') return { text: value }
    if (typeof value === 'object') return value as TooltipOptions
    return { text: String(value) }
}

export default defineNuxtPlugin((nuxtApp) => {
    nuxtApp.vueApp.directive('tooltip', {
        /**
         * Server render contributes nothing: no attribute, no markup. Present
         * only because Vue's SSR renderer requires it on any directive that
         * appears in a server-rendered template.
         */
        getSSRProps() {
            return {}
        },

        mounted(el: HTMLElement, binding) {
            const opts = normalise(binding.value)

            const state: TooltipState = {
                text: opts.text ? String(opts.text) : '',
                placement: opts.placement ?? 'bottom',
                delay: opts.delay ?? 120,
                bubble: null,
                timer: null,

                show() {
                    if (!state.text || state.bubble) return

                    const bubble = document.createElement('div')
                    bubble.className = TOOLTIP_BUBBLE_CLASS
                    bubble.setAttribute('role', 'tooltip')
                    bubble.textContent = state.text

                    const r = el.getBoundingClientRect()
                    const pos = resolveTooltipPosition(r, state.placement, window.innerHeight)
                    bubble.style.left = `${pos.x}px`
                    bubble.style.top = `${pos.y}px`
                    bubble.style.transform = pos.transform

                    document.body.appendChild(bubble)

                    // Clamp only once the bubble is in the DOM and has a real
                    // width — max-w-xs plus the text means it cannot be guessed.
                    bubble.style.left = `${clampTooltipCentre(pos.x, bubble.offsetWidth, window.innerWidth)}px`

                    state.bubble = bubble
                },

                hide() {
                    if (state.timer) {
                        clearTimeout(state.timer)
                        state.timer = null
                    }
                    state.bubble?.remove()
                    state.bubble = null
                },

                onEnter() {
                    if (state.timer) clearTimeout(state.timer)
                    state.timer = setTimeout(() => state.show(), state.delay)
                },
                onLeave() { state.hide() },
                // Keyboard focus shows immediately — a delay on focus just feels broken.
                onFocus() { state.hide(); state.show() },
                onKeydown(e: KeyboardEvent) { if (e.key === 'Escape') state.hide() },
            }

            // A tooltip anchored to something that has scrolled away is worse
            // than none, so dismiss rather than chase.
            el.addEventListener('mouseenter', state.onEnter)
            el.addEventListener('mouseleave', state.onLeave)
            el.addEventListener('focusin', state.onFocus)
            el.addEventListener('focusout', state.onLeave)
            window.addEventListener('scroll', state.onLeave, true)
            document.addEventListener('keydown', state.onKeydown)

            states.set(el, state)

            // Never leave both a native tooltip and this one on the same element.
            el.removeAttribute('title')
        },

        updated(el: HTMLElement, binding) {
            const state = states.get(el)
            if (!state) return
            const opts = normalise(binding.value)
            const next = opts.text ? String(opts.text) : ''
            if (next === state.text) return
            state.text = next
            // Re-render an open bubble so it does not show stale text.
            if (state.bubble) {
                state.hide()
                if (next) state.show()
            }
        },

        beforeUnmount(el: HTMLElement) {
            const state = states.get(el)
            if (!state) return
            state.hide()
            el.removeEventListener('mouseenter', state.onEnter)
            el.removeEventListener('mouseleave', state.onLeave)
            el.removeEventListener('focusin', state.onFocus)
            el.removeEventListener('focusout', state.onLeave)
            window.removeEventListener('scroll', state.onLeave, true)
            document.removeEventListener('keydown', state.onKeydown)
            states.delete(el)
        },
    })
})
