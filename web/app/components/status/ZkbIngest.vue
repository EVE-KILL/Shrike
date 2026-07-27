<script setup lang="ts">
interface ZkbIngestData {
    window_hours: number
    new_hourly: number[]
    repost_hourly: number[]
}

const props = defineProps<{ data: ZkbIngestData | null }>()

const hoverIdx = ref<number | null>(null)

const newTotal = computed(() =>
    props.data?.new_hourly.reduce((a, b) => a + b, 0) ?? 0,
)
const repostTotal = computed(() =>
    props.data?.repost_hourly.reduce((a, b) => a + b, 0) ?? 0,
)
const newLastHour = computed(() => {
    const arr = props.data?.new_hourly
    return arr && arr.length > 0 ? arr[arr.length - 1] ?? 0 : 0
})
const repostLastHour = computed(() => {
    const arr = props.data?.repost_hourly
    return arr && arr.length > 0 ? arr[arr.length - 1] ?? 0 : 0
})
const totalEvents = computed(() => newTotal.value + repostTotal.value)
const repostRatio = computed(() =>
    totalEvents.value > 0
        ? Math.round((repostTotal.value / totalEvents.value) * 100)
        : 0,
)
const avgNewPerHour = computed(() => {
    if (!props.data || props.data.window_hours === 0) return 0
    return Math.round(newTotal.value / props.data.window_hours)
})
// Bars are stacked green (new) + amber (repost) per hour. Scaling uses
// the busiest hour's total so the tallest bar fills the container — quieter
// hours appear shorter relative to it.
const maxStack = computed(() => {
    if (!props.data) return 1
    let max = 1
    for (let i = 0; i < props.data.new_hourly.length; i++) {
        const stack
            = (props.data.new_hourly[i] ?? 0)
            + (props.data.repost_hourly[i] ?? 0)
        if (stack > max) max = stack
    }
    return max
})

const stackHeight = (v: number): string => {
    if (!v || maxStack.value === 0) return '0%'
    return `${Math.max(1, Math.round((v / maxStack.value) * 100))}%`
}

const hoverBar = computed(() => {
    if (hoverIdx.value === null || !props.data) return null
    const newKills = props.data.new_hourly[hoverIdx.value] ?? 0
    const reposts = props.data.repost_hourly[hoverIdx.value] ?? 0
    return {
        index: hoverIdx.value,
        newKills,
        reposts,
        total: newKills + reposts,
    }
})

const tooltipLeft = computed(() => {
    if (!hoverBar.value || !props.data?.new_hourly.length) return '0%'
    const percent = ((hoverBar.value.index + 0.5) / props.data.new_hourly.length) * 100
    return `${Math.min(Math.max(percent, 12), 88)}%`
})

const formatHourWindow = (index: number): string => {
    if (!props.data) return ''
    const hoursAgo = props.data.window_hours - 1 - index
    const hourMs = 60 * 60 * 1000
    const currentEpochHour = Math.floor(Date.now() / hourMs) * hourMs
    const start = new Date(currentEpochHour - hoursAgo * hourMs)
    const end = new Date(start.getTime() + 60 * 60 * 1000)
    const date = start.toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
    })
    const time = (value: Date) =>
        value.toLocaleTimeString('en-US', {
            hour: 'numeric',
            minute: '2-digit',
        })
    return `${hoursAgo === 0 ? 'Current hour' : date} · ${time(start)}–${time(end)}`
}
</script>

<template>
    <div v-if="data" class="glass-panel p-4">
        <div class="flex items-center justify-between mb-3">
            <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80">zKB Ingest</div>
            <div class="text-xs text-gray-500">Last {{ data.window_hours }}h · hourly</div>
        </div>

        <div class="grid grid-cols-2 md:grid-cols-4 gap-2 mb-4">
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-xs text-gray-400">New kills</div>
                <div class="text-sm text-white font-medium tabular-nums">{{ formatNumber(newTotal) }}</div>
                <div class="text-fine text-gray-600 tabular-nums">{{ formatNumber(newLastHour) }} last hour</div>
            </div>
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-xs text-gray-400">Reposts skipped</div>
                <div class="text-sm text-white font-medium tabular-nums">{{ formatNumber(repostTotal) }}</div>
                <div class="text-fine text-gray-600 tabular-nums">{{ formatNumber(repostLastHour) }} last hour</div>
            </div>
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-xs text-gray-400">Repost ratio</div>
                <div class="text-sm font-medium tabular-nums" :class="repostRatio > 50 ? 'text-amber-400' : 'text-green-400'">{{ repostRatio }}%</div>
                <div class="text-fine text-gray-600">of all feed events</div>
            </div>
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-xs text-gray-400">New / hour avg</div>
                <div class="text-sm text-white font-medium tabular-nums">{{ formatNumber(avgNewPerHour) }}</div>
                <div class="text-fine text-gray-600">over {{ data.window_hours }}h</div>
            </div>
        </div>

        <div class="relative" @mouseleave="hoverIdx = null">
            <div class="relative h-24">
                <div class="absolute inset-0 pointer-events-none flex flex-col justify-between py-px">
                    <div v-for="line in 5" :key="line" class="border-t border-white/[0.05]"></div>
                </div>

                <div class="relative flex items-stretch gap-0.5 h-full">
                    <div
                        v-for="(n, i) in data.new_hourly"
                        :key="i"
                        class="relative flex-1 h-full min-w-0 flex items-end justify-center rounded-sm transition-colors duration-100"
                        :class="hoverIdx === i ? 'bg-white/[0.05]' : ''"
                        @mouseenter="hoverIdx = i"
                    >
                        <div
                            class="relative w-[68%] min-w-[2px] max-w-7 overflow-hidden rounded-t-sm transition-opacity duration-100"
                            :class="hoverIdx === null || hoverIdx === i ? 'opacity-100' : 'opacity-35'"
                            :style="{ height: stackHeight(n + (data.repost_hourly[i] ?? 0)) }"
                        >
                            <div class="flex flex-col h-full gap-px">
                                <div
                                    v-if="data.repost_hourly[i]"
                                    class="bg-amber-500/70 min-h-px"
                                    :style="{ flexGrow: data.repost_hourly[i], flexBasis: 0 }"
                                ></div>
                                <div
                                    v-if="n"
                                    class="bg-green-500/85 min-h-px"
                                    :style="{ flexGrow: n, flexBasis: 0 }"
                                ></div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <div
                v-if="hoverBar"
                class="pointer-events-none absolute top-1 z-20 -translate-x-1/2 rounded-lg bg-black/90 backdrop-blur-xl border border-white/[0.10] px-3 py-2 shadow-2xl shadow-black/60 min-w-[190px]"
                :style="{ left: tooltipLeft }"
            >
                <div class="text-fine font-medium text-gray-300 mb-1.5">
                    {{ formatHourWindow(hoverBar.index) }}
                </div>
                <div class="flex items-center gap-2 py-0.5 text-fine">
                    <span class="w-2 h-2 rounded-sm bg-green-500/85 flex-shrink-0"></span>
                    <span class="text-gray-400 flex-1">New kills</span>
                    <span class="text-gray-200 tabular-nums">{{ formatNumber(hoverBar.newKills) }}</span>
                </div>
                <div class="flex items-center gap-2 py-0.5 text-fine">
                    <span class="w-2 h-2 rounded-sm bg-amber-500/70 flex-shrink-0"></span>
                    <span class="text-gray-400 flex-1">Reposts</span>
                    <span class="text-gray-200 tabular-nums">{{ formatNumber(hoverBar.reposts) }}</span>
                </div>
                <div class="flex items-center gap-2 pt-1 mt-1 border-t border-white/[0.08] text-fine">
                    <span class="w-2 h-2 flex-shrink-0"></span>
                    <span class="text-gray-500 flex-1">Total events</span>
                    <span class="text-white tabular-nums">{{ formatNumber(hoverBar.total) }}</span>
                </div>
            </div>

            <div class="text-fine text-gray-500 mt-1 flex justify-between tabular-nums">
                <span>-{{ data.window_hours }}h</span>
                <span class="flex gap-3">
                    <span><span class="inline-block w-2 h-2 bg-green-500/80 mr-1 align-middle"></span>new</span>
                    <span><span class="inline-block w-2 h-2 bg-amber-500/60 mr-1 align-middle"></span>repost</span>
                </span>
                <span>now</span>
            </div>
        </div>
    </div>
</template>
