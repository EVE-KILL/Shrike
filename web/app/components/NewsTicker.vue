<script setup lang="ts">
const { tickerItems, tickerCollapsed, dismiss, toggleTicker } = useAnnouncements()

const colorClasses: Record<string, { text: string; bg: string }> = {
    info:    { text: 'text-blue-300',  bg: 'bg-blue-500/15 border-blue-500/25' },
    warning: { text: 'text-amber-300', bg: 'bg-amber-500/15 border-amber-500/25' },
    danger:  { text: 'text-red-300',   bg: 'bg-red-500/15 border-red-500/25' },
    success: { text: 'text-green-300', bg: 'bg-green-500/15 border-green-500/25' },
}

function getColor(color: string) {
    return colorClasses[color] || colorClasses.info
}

// ── Decay ──────────────────────────────────────────────────────────────────
// Items used to hold full strength right up to expiry and then vanish
// mid-scroll. Instead they fade over the last stretch of their life, so the
// strip visibly ages: fresh news is bright, stale news dims out before it
// goes. Purely visual — expiry itself is still owned by useAnnouncements.
const FADE_WINDOW_MS = 10 * 60 * 1000 // last 10 minutes of an item's life
const MIN_OPACITY = 0.35

// One shared clock rather than a timer per item; a coarse tick is plenty for
// a fade measured in minutes, and it keeps the marquee off the main thread.
const now = ref(Date.now())
let clock: ReturnType<typeof setInterval> | null = null

function freshness(item: { expires_at?: string | Date | null }): number {
    if (!item.expires_at) return 1
    const remaining = new Date(item.expires_at).getTime() - now.value
    if (!Number.isFinite(remaining)) return 1
    if (remaining >= FADE_WINDOW_MS) return 1
    if (remaining <= 0) return MIN_OPACITY
    return MIN_OPACITY + (1 - MIN_OPACITY) * (remaining / FADE_WINDOW_MS)
}

// ── Constant-speed marquee, compositor-driven ─────────────────────────────
// The strip moves via a CSS animation, so scrolling costs zero main-thread
// work (an earlier rAF version woke JS 60×/s for the whole page lifetime).
// Item churn via WebSocket changes the strip width, and naively changing
// animation-duration mid-flight makes browsers re-map elapsed time — a
// visible jump. resync() avoids that: it reads the current visual offset
// and restarts the animation with a negative delay landing on the exact
// same offset, so a width change is seamless.
const SCROLL_SPEED = 80 // px/sec — tune to taste
const scrollRef = ref<HTMLElement | null>(null)

let resizeOb: ResizeObserver | null = null
let setWidth = 0

function resync() {
    const el = scrollRef.value
    if (!el) return
    const firstSet = el.querySelector('.ticker-set') as HTMLElement | null
    const newWidth = firstSet?.scrollWidth ?? 0
    if (newWidth === setWidth) return

    // Current visual offset within the loop, from the live transform matrix
    let offset = 0
    if (newWidth > 0) {
        const t = getComputedStyle(el).transform
        const m = t && t !== 'none' ? t.match(/matrix\(([^)]+)\)/) : null
        if (m?.[1]) {
            const tx = Number(m[1].split(',')[4] ?? 0)
            offset = ((-tx % newWidth) + newWidth) % newWidth
        }
    }

    setWidth = newWidth
    if (newWidth <= 0) return
    el.style.setProperty('--set-width', `${newWidth}px`)
    el.style.setProperty('--marquee-duration', `${(newWidth / SCROLL_SPEED).toFixed(3)}s`)
    // Restart the animation at the equivalent progress: name→none forces a
    // fresh start, the negative delay fast-forwards it to `offset`.
    el.style.animationName = 'none'
    void el.offsetWidth
    el.style.animationDelay = `-${(offset / SCROLL_SPEED).toFixed(3)}s`
    el.style.animationName = ''
}

onMounted(() => {
    clock = setInterval(() => { now.value = Date.now() }, 15_000)
    resizeOb = new ResizeObserver(resync)
    if (scrollRef.value) resizeOb.observe(scrollRef.value)
    resync()
})

watch(scrollRef, (el) => {
    setWidth = 0
    if (el && resizeOb) {
        resizeOb.observe(el)
        resync()
    }
})

onBeforeUnmount(() => {
    resizeOb?.disconnect()
    if (clock) clearInterval(clock)
})
</script>

<template>
    <Teleport to="body">
        <Transition
            enter-active-class="transition duration-300 ease-out"
            enter-from-class="translate-y-full opacity-0"
            enter-to-class="translate-y-0 opacity-100"
            leave-active-class="transition duration-200 ease-in"
            leave-from-class="translate-y-0 opacity-100"
            leave-to-class="translate-y-full opacity-0"
        >
            <div
                v-if="tickerItems.length > 0"
                class="fixed bottom-0 inset-x-0 z-[9998] border-t border-white/[0.08]"
            >
                <!-- Collapsed state: small toggle button -->
                <div
                    v-if="tickerCollapsed"
                    class="flex justify-center py-0.5 bg-black/70 backdrop-blur-sm cursor-pointer hover:bg-black/80 transition-colors"
                    @click="toggleTicker"
                >
                    <Icon name="lucide:chevron-up" class="text-xs text-gray-500" />
                </div>

                <!-- Expanded ticker -->
                <div v-else class="ticker-bg relative">
                    <!-- Controls -->
                    <div class="absolute right-0 top-0 bottom-0 z-10 flex items-center gap-1 px-2 bg-gradient-to-l from-black/90 via-black/70 to-transparent pl-8">
                        <button
                            class="flex items-center justify-center w-5 h-5 rounded text-gray-500 hover:text-white transition-colors cursor-pointer"
                            v-tooltip="'Collapse ticker'"
                            @click="toggleTicker"
                        >
                            <Icon name="lucide:chevron-down" class="text-xs" />
                        </button>
                    </div>

                    <!-- Scrolling content -->
                    <div class="ticker-track overflow-hidden py-1.5 pr-16">
                        <div ref="scrollRef" class="ticker-scroll">
                            <!-- Each copy is forced to at least viewport width so you never see both -->
                            <span v-for="pass in 2" :key="pass" class="ticker-set">
                                <span
                                    v-for="item in tickerItems"
                                    :key="`${pass}-${item.id}`"
                                    class="inline-flex items-center gap-1.5 text-xs mx-3 px-2.5 py-1 rounded-md border transition-opacity duration-1000"
                                    :class="getColor(item.color).bg"
                                    :style="{ opacity: freshness(item) }"
                                >
                                    <Icon
                                        :name="item.icon || 'lucide:info'"
                                        class="text-[10px] flex-shrink-0"
                                        :class="getColor(item.color).text"
                                    />
                                    <NuxtLink
                                        v-if="item.link_url"
                                        :to="item.link_url"
                                        class="hover:underline"
                                        :class="getColor(item.color).text"
                                    >
                                        <strong>{{ item.title }}</strong>
                                        <span v-if="item.body_md" class="text-gray-400 ml-1">— {{ item.body_md }}</span>
                                    </NuxtLink>
                                    <span v-else>
                                        <strong :class="getColor(item.color).text">{{ item.title }}</strong>
                                        <span v-if="item.body_md" class="text-gray-400 ml-1">— {{ item.body_md }}</span>
                                    </span>
                                    <button
                                        class="text-gray-600 hover:text-gray-400 transition-colors cursor-pointer ml-1"
                                        v-tooltip="'Dismiss'"
                                        @click.prevent="dismiss(item.id)"
                                    >
                                        <Icon name="lucide:x" class="text-[10px]" />
                                    </button>
                                </span>
                            </span>
                        </div>
                    </div>
                </div>
            </div>
        </Transition>
    </Teleport>
</template>

<style scoped>
/* Isolate the ticker's compositing from the page — without this, Chrome can
   re-rasterize page content when the wide animated layer + backdrop blur
   composite together, which shows up as full-page flicker. */
.ticker-bg {
    isolation: isolate;
    transform: translateZ(0);
}

/* Blur lives on a static pseudo-element in its own layer so the moving strip
   above never invalidates the cached blur result. */
.ticker-bg::before {
    content: '';
    position: absolute;
    inset: 0;
    z-index: 0;
    background: rgba(0, 0, 0, 0.4);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    transform: translateZ(0);
}

/* Scroll layer — compositor CSS animation; script only resyncs on width
   change (see resync()). Custom props are set inline by the script. */
.ticker-scroll {
    display: inline-flex;
    white-space: nowrap;
    position: relative;
    z-index: 1;
    will-change: transform;
    contain: layout style;
    animation: ticker-marquee var(--marquee-duration, 60s) linear infinite;
}

.ticker-track:hover .ticker-scroll {
    animation-play-state: paused;
}

@media (prefers-reduced-motion: reduce) {
    .ticker-scroll {
        animation: none;
    }
}

@keyframes ticker-marquee {
    from { transform: translateX(0); }
    to { transform: translateX(calc(-1 * var(--set-width, 0px))); }
}

/* Each copy of the items spans at least the full viewport. With two copies
   side by side and the offset wrapping at one copy's width, the loop is
   seamless — you never see both copies at once. */
.ticker-set {
    display: inline-flex;
    align-items: center;
    min-width: 100vw;
    justify-content: center;
}
</style>
