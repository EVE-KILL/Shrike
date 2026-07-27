/**
 * The tooltip bubble's look, in one place.
 *
 * Shared by the Tooltip component and the v-tooltip directive so the two can
 * never drift apart. The component wraps its trigger in an element and is the
 * right choice when the content is rich; the directive attaches to an existing
 * element and is the right choice when replacing a native `title`, because it
 * adds no wrapper and so cannot disturb a flex or grid layout.
 */
export const TOOLTIP_BUBBLE_CLASS
    = 'fixed z-[10000] pointer-events-none max-w-xs rounded-md border border-white/[0.12] '
    + 'bg-[#141414]/97 backdrop-blur-xl px-2.5 py-1.5 text-fine text-gray-300 shadow-xl shadow-black/60'

export const TOOLTIP_GAP = 8

/** Enough room for a two-line bubble; below this, flip rather than clip. */
export const TOOLTIP_ESTIMATED_HEIGHT = 64

/** Keep this clear of the viewport edges. */
export const TOOLTIP_MARGIN = 8

/**
 * Pull the bubble's centre point in far enough that its edges stay on screen.
 *
 * The bubble is centred on its trigger with `translate(-50%)`, so a trigger
 * within half a bubble-width of either edge hangs off it — the background
 * controls in the bottom-left corner rendered as "e background view", with the
 * first word cut off. Clamping the centre keeps the transform untouched.
 *
 * Requires the bubble's real width, so callers must measure after it is in the
 * DOM rather than guessing.
 */
export function clampTooltipCentre(centreX: number, bubbleWidth: number, viewportWidth: number) {
    const half = bubbleWidth / 2
    const min = TOOLTIP_MARGIN + half
    const max = viewportWidth - TOOLTIP_MARGIN - half

    // Wider than the viewport: centre it and overflow evenly, rather than
    // pinning one edge and pushing the whole thing off the other side.
    if (min > max) return viewportWidth / 2

    return Math.min(Math.max(centreX, min), max)
}

/**
 * Where to put a bubble anchored to `rect`, preferring `placement` but flipping
 * when there is no room. Returns viewport coordinates plus the side used, so the
 * caller can pick the matching transform.
 */
export function resolveTooltipPosition(
    rect: { top: number; bottom: number; left: number; width: number },
    placement: 'top' | 'bottom',
    viewportHeight: number,
) {
    const roomBelow = viewportHeight - rect.bottom
    const roomAbove = rect.top

    let side = placement
    if (side === 'bottom' && roomBelow < TOOLTIP_ESTIMATED_HEIGHT && roomAbove > roomBelow) side = 'top'
    else if (side === 'top' && roomAbove < TOOLTIP_ESTIMATED_HEIGHT && roomBelow > roomAbove) side = 'bottom'

    return {
        side,
        x: Math.round(rect.left + rect.width / 2),
        y: side === 'top' ? Math.round(rect.top - TOOLTIP_GAP) : Math.round(rect.bottom + TOOLTIP_GAP),
        transform: side === 'top' ? 'translate(-50%, -100%)' : 'translate(-50%, 0)',
    }
}
