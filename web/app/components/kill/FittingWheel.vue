<script setup lang="ts">
const AMMO_CATEGORY_ID = 8

const props = defineProps<{
    shipTypeId: number
    shipName: string | null
    items: {
        type_id: number
        type_name: string | null
        flag_id: number
        slot: string
        category_id: number | null
        quantity_dropped: number
        quantity_destroyed: number
    }[]
}>()

// Slot flag ranges
const HIGH_FLAGS = [27, 28, 29, 30, 31, 32, 33, 34]
const MID_FLAGS = [19, 20, 21, 22, 23, 24, 25, 26]
const LOW_FLAGS = [11, 12, 13, 14, 15, 16, 17, 18]
const RIG_FLAGS = [92, 93, 94]
const SUB_FLAGS = [125, 126, 127, 128]

// Organize modules into slot arrays (exclude ammo)
const organizeSlots = (flags: number[], max: number) => {
    const slots: (typeof props.items[0] | null)[] = Array(max).fill(null)
    for (const item of props.items) {
        if (item.category_id === AMMO_CATEGORY_ID) continue
        const idx = flags.indexOf(item.flag_id)
        if (idx !== -1 && idx < max) slots[idx] = item
    }
    return slots
}

// Build ammo lookup by flag_id
const ammoByFlag = computed(() => {
    const map = new Map<number, typeof props.items[0]>()
    for (const item of props.items) {
        if (item.category_id === AMMO_CATEGORY_ID) {
            map.set(item.flag_id, item)
        }
    }
    return map
})

const highSlots = computed(() => organizeSlots(HIGH_FLAGS, 8))
const midSlots = computed(() => organizeSlots(MID_FLAGS, 8))
const lowSlots = computed(() => organizeSlots(LOW_FLAGS, 8))
const rigSlots = computed(() => organizeSlots(RIG_FLAGS, 3))
const subSlots = computed(() => organizeSlots(SUB_FLAGS, 4))
const hasSubsystems = computed(() => subSlots.value.some(s => s !== null))

// Get ammo for a specific slot
const getAmmo = (flags: number[], idx: number) => {
    return ammoByFlag.value.get(flags[idx]!) ?? null
}

// Position a slot on the circle
const getSlotPos = (index: number, position: string): { left: string; top: string } => {
    const radius = 42
    let angle = 0

    switch (position) {
        case 'top': angle = -125 + index * 10.5; break
        case 'right': angle = -37 + index * 10.5; break
        case 'bottom': angle = 54 + index * 10.5; break
        case 'left': angle = 196 + index * 10.5; break
        case 'sub': angle = 140 + index * 10.5; break
    }

    const rad = angle * (Math.PI / 180)
    const x = Math.round((50 + radius * Math.cos(rad)) * 1000) / 1000
    const y = Math.round((50 + radius * Math.sin(rad)) * 1000) / 1000

    return { left: `${x}%`, top: `${y}%` }
}

// Tooltip
const tooltip = reactive({ visible: false, name: '', ammoName: '', dropped: 0, destroyed: 0 })
let hideTimer: ReturnType<typeof setTimeout> | null = null
let showTimer: ReturnType<typeof setTimeout> | null = null

const showTip = (item: typeof props.items[0], ammo: typeof props.items[0] | null) => {
    if (hideTimer) clearTimeout(hideTimer)
    if (showTimer) clearTimeout(showTimer)
    const delay = tooltip.visible ? 0 : 250
    showTimer = setTimeout(() => {
        tooltip.name = item.type_name || `Type ${item.type_id}`
        tooltip.ammoName = ammo?.type_name || ''
        tooltip.dropped = item.quantity_dropped
        tooltip.destroyed = item.quantity_destroyed
        tooltip.visible = true
    }, delay)
}

const hideTip = () => {
    if (showTimer) clearTimeout(showTimer)
    hideTimer = setTimeout(() => { tooltip.visible = false }, 150)
}

const wheelRef = ref<HTMLElement | null>(null)
const onWheelMouseMove = (e: MouseEvent) => {
    if (!tooltip.visible || !wheelRef.value) return
    const rect = wheelRef.value.getBoundingClientRect()
    const cx = rect.left + rect.width / 2
    const cy = rect.top + rect.height / 2
    const r = rect.width / 2
    const dx = e.clientX - cx
    const dy = e.clientY - cy
    if (dx * dx + dy * dy > r * r) {
        hideTip()
    }
}

// Slot colors
const slotColors: Record<string, string> = {
    high: 'rgba(180, 60, 60, 0.8)',
    mid: 'rgba(60, 120, 180, 0.8)',
    low: 'rgba(180, 140, 60, 0.8)',
    rig: 'rgba(150, 150, 150, 0.6)',
}
</script>

<template>
    <div class="w-full mx-auto">
        <div ref="wheelRef" class="relative w-full" style="padding-bottom: 100%" @mousemove="onWheelMouseMove" @mouseleave="hideTip">
            <!-- Ship render (background, clipped to circle) -->
            <div class="absolute inset-[3%] rounded-full overflow-hidden z-[1]" :class="{ 'brightness-[0.4] blur-[1px]': tooltip.visible }">
                <img
                    :src="`/images/types/${shipTypeId}/overlayrender?size=512`"
                    :alt="shipName || ''"
                    width="512"
                    height="512"
                    loading="eager"
                    decoding="async"
                    fetchpriority="high"
                    class="w-full h-full object-cover rounded-full transition-all duration-200"
                >
            </div>

            <!-- Outer ring -->
            <svg class="absolute inset-0 w-full h-full z-[4] pointer-events-none" viewBox="24 24 464 464">
                <circle cx="256" cy="256" r="224" fill="none" stroke="black" stroke-width="16" />
            </svg>

            <!-- Inner ring (slot track) -->
            <svg class="absolute inset-0 w-full h-full z-[3] pointer-events-none" viewBox="24 24 464 464">
                <circle cx="256" cy="256" r="195" fill="none" stroke="black" stroke-width="46" stroke-opacity="0.6" />
            </svg>

            <!-- Center tooltip overlay -->
            <div v-if="tooltip.visible" class="absolute inset-[15%] z-[10] flex items-center justify-center pointer-events-none">
                <div class="bg-black/80 backdrop-blur-sm rounded-lg px-4 py-2.5 text-center border border-white/[0.08]">
                    <div class="text-xs text-white font-medium">{{ tooltip.name }}</div>
                    <div v-if="tooltip.ammoName" class="text-fine text-amber-400/80 mt-0.5">{{ tooltip.ammoName }}</div>
                    <div class="flex items-center justify-center gap-3 mt-1 text-fine">
                        <span v-if="tooltip.dropped" class="text-green-400">+{{ tooltip.dropped }} dropped</span>
                        <span v-if="tooltip.destroyed" class="text-red-400">-{{ tooltip.destroyed }} destroyed</span>
                    </div>
                </div>
            </div>

            <!-- Slot indicator dots (colored markers at start of each slot group) -->
            <div v-for="(pos, key) in { top: 'high', right: 'mid', bottom: 'low', left: 'rig' }" :key="key"
                class="absolute w-3 h-3 rounded-full z-[7]"
                :style="{ ...getSlotPos(-1, key), transform: 'translate(-50%, -50%)', backgroundColor: slotColors[pos] }"
            ></div>

            <!-- High slots (top) -->
            <div
                v-for="(item, idx) in highSlots" :key="`h-${idx}`"
                class="absolute w-[42px] h-[42px] rounded-full z-[6] flex items-center justify-center"
                :style="{ ...getSlotPos(idx, 'top'), transform: 'translate(-50%, -50%)' }"
            >
                <div v-if="item" class="relative w-full h-full rounded-full overflow-visible cursor-pointer"
                    @mouseenter="showTip(item, getAmmo(HIGH_FLAGS, idx))">
                    <img :src="`/images/types/${item.type_id}/icon?size=64`" :alt="item.type_name || 'Module'" class="w-full h-full object-cover rounded-full bg-black/40" loading="lazy">
                    <div v-if="getAmmo(HIGH_FLAGS, idx)" class="absolute -bottom-1 -right-1 w-[20px] h-[20px] rounded-full overflow-hidden border border-gray-700/60 bg-black/60 z-[9]">
                        <img :src="`/images/types/${getAmmo(HIGH_FLAGS, idx)!.type_id}/icon?size=64`" alt="Ammo/charge" class="w-full h-full object-cover">
                    </div>
                </div>
                <div v-else class="w-full h-full rounded-full bg-black/10"></div>
            </div>

            <!-- Mid slots (right) -->
            <div
                v-for="(item, idx) in midSlots" :key="`m-${idx}`"
                class="absolute w-[42px] h-[42px] rounded-full z-[6] flex items-center justify-center"
                :style="{ ...getSlotPos(idx, 'right'), transform: 'translate(-50%, -50%)' }"
            >
                <div v-if="item" class="relative w-full h-full rounded-full overflow-visible cursor-pointer"
                    @mouseenter="showTip(item, getAmmo(MID_FLAGS, idx))">
                    <img :src="`/images/types/${item.type_id}/icon?size=64`" :alt="item.type_name || 'Module'" class="w-full h-full object-cover rounded-full bg-black/40" loading="lazy">
                    <div v-if="getAmmo(MID_FLAGS, idx)" class="absolute -bottom-1 -right-1 w-[20px] h-[20px] rounded-full overflow-hidden border border-gray-700/60 bg-black/60 z-[9]">
                        <img :src="`/images/types/${getAmmo(MID_FLAGS, idx)!.type_id}/icon?size=64`" alt="Ammo/charge" class="w-full h-full object-cover">
                    </div>
                </div>
                <div v-else class="w-full h-full rounded-full bg-black/10"></div>
            </div>

            <!-- Low slots (bottom) -->
            <div
                v-for="(item, idx) in lowSlots" :key="`l-${idx}`"
                class="absolute w-[42px] h-[42px] rounded-full z-[6] flex items-center justify-center"
                :style="{ ...getSlotPos(idx, 'bottom'), transform: 'translate(-50%, -50%)' }"
            >
                <div v-if="item" class="relative w-full h-full rounded-full overflow-visible cursor-pointer"
                    @mouseenter="showTip(item, getAmmo(LOW_FLAGS, idx))">
                    <img :src="`/images/types/${item.type_id}/icon?size=64`" :alt="item.type_name || 'Module'" class="w-full h-full object-cover rounded-full bg-black/40" loading="lazy">
                    <div v-if="getAmmo(LOW_FLAGS, idx)" class="absolute -bottom-1 -right-1 w-[20px] h-[20px] rounded-full overflow-hidden border border-gray-700/60 bg-black/60 z-[9]">
                        <img :src="`/images/types/${getAmmo(LOW_FLAGS, idx)!.type_id}/icon?size=64`" alt="Ammo/charge" class="w-full h-full object-cover">
                    </div>
                </div>
                <div v-else class="w-full h-full rounded-full bg-black/10"></div>
            </div>

            <!-- Rig slots (left) -->
            <div
                v-for="(item, idx) in rigSlots" :key="`r-${idx}`"
                class="absolute w-[42px] h-[42px] rounded-full z-[6] flex items-center justify-center"
                :style="{ ...getSlotPos(idx, 'left'), transform: 'translate(-50%, -50%)' }"
            >
                <div v-if="item" class="w-full h-full rounded-full overflow-hidden cursor-pointer bg-black/40"
                    @mouseenter="showTip(item, null)">
                    <img :src="`/images/types/${item.type_id}/icon?size=64`" :alt="item.type_name || 'Module'" class="w-full h-full object-cover" loading="lazy">
                </div>
                <div v-else class="w-full h-full rounded-full bg-black/10"></div>
            </div>

            <!-- Subsystem slots (T3 only) -->
            <template v-if="hasSubsystems">
                <div
                    v-for="(item, idx) in subSlots" :key="`s-${idx}`"
                    class="absolute w-[42px] h-[42px] rounded-full z-[6] flex items-center justify-center"
                    :style="{ ...getSlotPos(idx, 'sub'), transform: 'translate(-50%, -50%)' }"
                >
                    <div v-if="item" class="w-full h-full rounded-full overflow-hidden cursor-pointer bg-black/40"
                        @mouseenter="showTip(item, null)">
                        <img :src="`/images/types/${item.type_id}/icon?size=64`" :alt="item.type_name || 'Module'" class="w-full h-full object-cover" loading="lazy">
                    </div>
                    <div v-else class="w-full h-full rounded-full bg-black/10"></div>
                </div>
            </template>
        </div>
    </div>
</template>
