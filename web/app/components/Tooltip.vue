<script setup lang="ts">
/**
 * Tooltip.
 *
 * Native `title` has three problems we keep running into: it takes about a
 * second to appear, it cannot be styled to match anything else on the page,
 * and it never shows on touch or from the keyboard. This replaces it.
 *
 * The bubble is teleported to <body> and positioned fixed, so it escapes the
 * `overflow: hidden` on panels, the ticker track and the killmail item table —
 * an absolutely-positioned tooltip gets clipped by all three.
 *
 * Shows on hover and on keyboard focus; Escape dismisses it.
 */
import { TOOLTIP_BUBBLE_CLASS, clampTooltipCentre, resolveTooltipPosition } from '~/utils/tooltipBubble'

const props = withDefaults(defineProps<{
    /** Tooltip text. Use the #content slot instead for anything richer. */
    text?: string
    /** Preferred side. Flips automatically when there is no room. */
    placement?: 'top' | 'bottom'
    /** Milliseconds before it appears on hover. Focus always shows instantly. */
    delay?: number
    /** Render the trigger inline-flex rather than inline-block. */
    inlineFlex?: boolean
    /**
     * Render the trigger as a block. Needed whenever the trigger is meant to
     * fill its container — a passed-in `class="block"` cannot do this, because
     * it merges with the default `inline-block` and loses on stylesheet order,
     * leaving a wrapper that shrink-wraps to zero width.
     */
    block?: boolean
    disabled?: boolean
}>(), {
    placement: 'bottom',
    delay: 120,
    inlineFlex: false,
    block: false,
    disabled: false,
})

const display = computed(() => props.block ? 'block' : props.inlineFlex ? 'inline-flex' : 'inline-block')

const trigger = ref<HTMLElement | null>(null)
const bubble = ref<HTMLElement | null>(null)
const shown = ref(false)
const coords = ref({ x: 0, y: 0 })

let timer: ReturnType<typeof setTimeout> | null = null
const clearTimer = () => { if (timer) { clearTimeout(timer); timer = null } }

/** Side actually used — the preferred one unless it would run off-screen. */
const resolved = ref<'top' | 'bottom'>('bottom')

// Placement comes from utils/tooltipBubble, shared with the v-tooltip directive.
// This component used to carry its own copy of the gap, the flip threshold and
// the flip logic, which is precisely how the two drift apart.
const place = () => {
    const el = trigger.value
    if (!el) return
    const pos = resolveTooltipPosition(el.getBoundingClientRect(), props.placement, window.innerHeight)
    resolved.value = pos.side
    coords.value = { x: pos.x, y: pos.y }
}

/**
 * Pull the bubble back on screen once it has a measurable width. Cannot be done
 * in `place()`: the bubble is not rendered until `shown` flips, and max-w-xs
 * plus the content means its width is not predictable.
 */
const clampToViewport = async () => {
    await nextTick()
    const el = bubble.value
    if (!el) return
    coords.value = { ...coords.value, x: clampTooltipCentre(coords.value.x, el.offsetWidth, window.innerWidth) }
}

const show = (immediate = false) => {
    if (props.disabled) return
    clearTimer()
    const reveal = () => { place(); shown.value = true; clampToViewport() }
    if (immediate || props.delay <= 0) reveal()
    else timer = setTimeout(reveal, props.delay)
}

const hide = () => {
    clearTimer()
    shown.value = false
}

const onKeydown = (e: KeyboardEvent) => { if (e.key === 'Escape') hide() }

onMounted(() => {
    document.addEventListener('keydown', onKeydown)
    // A tooltip anchored to something that has scrolled away is worse than
    // none, so dismiss rather than chase.
    window.addEventListener('scroll', hide, true)
})
onUnmounted(() => {
    clearTimer()
    document.removeEventListener('keydown', onKeydown)
    window.removeEventListener('scroll', hide, true)
})
</script>

<template>
    <span
        ref="trigger"
        :class="display"
        @mouseenter="show()"
        @mouseleave="hide"
        @focusin="show(true)"
        @focusout="hide"
    >
        <slot />

        <Teleport to="body">
            <Transition
                enter-active-class="transition duration-150 ease-out"
                enter-from-class="opacity-0 -translate-y-0.5"
                leave-active-class="transition duration-100 ease-in"
                leave-to-class="opacity-0"
            >
                <span
                    v-if="shown && (text || $slots.content)"
                    ref="bubble"
                    role="tooltip"
                    :class="TOOLTIP_BUBBLE_CLASS"
                    :style="{
                        left: `${coords.x}px`,
                        top: `${coords.y}px`,
                        transform: resolved === 'top' ? 'translate(-50%, -100%)' : 'translate(-50%, 0)',
                    }"
                >
                    <slot name="content">{{ text }}</slot>
                </span>
            </Transition>
        </Teleport>
    </span>
</template>
