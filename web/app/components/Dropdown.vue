<script setup lang="ts">
import { onClickOutside } from '@vueuse/core'

export interface DropdownItem {
    name: string
    to: string
    icon?: string
}

export interface DropdownColumn {
    label: string
    items: DropdownItem[]
}

const props = defineProps<{
    /** v-model open state */
    modelValue?: boolean
    /** Horizontal alignment of the panel */
    align?: 'left' | 'right'
    /**
     * Mega menu mode: pass an array of column groups.
     * Each group has a label and items.
     */
    columns?: DropdownColumn[]
    /** Prominent links rendered above a mega menu's column grid. */
    featuredItems?: DropdownItem[]
    /**
     * Simple list mode: pass a flat array of items.
     * Ignored if columns is provided.
     */
    items?: DropdownItem[]
    /** Show a search input at the top of the panel. The search term is exposed via the slot scope. */
    searchable?: boolean
    /** Placeholder text for the search input */
    searchPlaceholder?: string
}>()

const emit = defineEmits(['update:modelValue'])

const isOpen = computed({
    get: () => props.modelValue ?? false,
    set: (v: boolean) => emit('update:modelValue', v),
})

const containerRef = ref<HTMLElement | null>(null)

onClickOutside(containerRef, () => {
    if (isOpen.value) isOpen.value = false
})

const onKeydown = (e: KeyboardEvent) => {
    if (e.key === 'Escape' && isOpen.value) isOpen.value = false
}

onMounted(() => document.addEventListener('keydown', onKeydown))
onUnmounted(() => document.removeEventListener('keydown', onKeydown))

const search = ref('')
const searchInputRef = ref<HTMLInputElement | null>(null)

const close = () => {
    isOpen.value = false
    search.value = ''
}

watch(isOpen, (open) => {
    if (open && props.searchable) {
        nextTick(() => searchInputRef.value?.focus())
    } else if (!open) {
        search.value = ''
    }
})

const isMega = computed(() => !!props.columns?.length)

const columnIcons: Record<string, string> = {
    Activity: 'lucide:zap',
    Security: 'lucide:shield',
    Timezones: 'lucide:clock',
    'Attacker Counts': 'lucide:users',
    Involvement: 'lucide:crosshair',
    'Faction Warfare': 'lucide:swords',
    Value: 'lucide:coins',
    'Victim Hulls': 'lucide:rocket',
    'Victim Categories': 'lucide:boxes',
    Technology: 'lucide:settings',
}

const columnIcon = (label: string) => columnIcons[label] ?? 'lucide:circle-dot'
</script>

<template>
    <div ref="containerRef" class="relative inline-flex">
        <!-- Trigger -->
        <div @click="isOpen = !isOpen">
            <slot name="trigger" />
        </div>

        <!-- Panel -->
        <Transition
            :enter-active-class="isMega ? 'transition-opacity duration-150 ease-out' : 'transition duration-200 ease-out'"
            :enter-from-class="isMega ? 'opacity-0' : 'opacity-0 translate-y-1 scale-[0.98]'"
            :enter-to-class="isMega ? 'opacity-100' : 'opacity-100 translate-y-0 scale-100'"
            :leave-active-class="isMega ? 'transition-opacity duration-100 ease-in' : 'transition duration-150 ease-in'"
            :leave-from-class="isMega ? 'opacity-100' : 'opacity-100 translate-y-0 scale-100'"
            :leave-to-class="isMega ? 'opacity-0' : 'opacity-0 translate-y-1 scale-[0.98]'"
        >
            <div
                v-if="isOpen"
                class="z-50 rounded-xl border border-white/[0.10] bg-[#141414]/95 backdrop-blur-2xl shadow-2xl shadow-black/60"
                :class="[
                    isMega
                        ? 'fixed left-1/2 top-16 w-[min(calc(var(--max-width-inner)-5rem),calc(100vw-6rem))] -translate-x-1/2 origin-top overflow-hidden p-5'
                        : `absolute top-full mt-2 max-h-[70vh] min-w-[11rem] overflow-y-auto p-1.5 ${align === 'right' ? 'right-0 origin-top-right' : 'left-0 origin-top-left'}`,
                ]"
            >
                <!-- ===== MEGA MENU MODE ===== -->
                <template v-if="isMega">
                    <div class="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_12%_0%,rgba(59,130,246,0.09),transparent_34%),radial-gradient(circle_at_88%_100%,rgba(99,102,241,0.07),transparent_38%),linear-gradient(135deg,rgba(255,255,255,0.012),transparent_45%)]" />
                    <div class="pointer-events-none absolute -right-20 -top-24 h-64 w-64 rounded-full border border-blue-400/[0.035]" />
                    <div class="pointer-events-none absolute -right-8 -top-12 h-40 w-40 rounded-full border border-blue-400/[0.035]" />

                    <div v-if="featuredItems?.length" class="relative z-10 mb-4 grid grid-cols-3 gap-2 border-b border-white/[0.08] pb-4 lg:flex">
                        <NuxtLink
                            v-for="item in featuredItems"
                            :key="item.to"
                            :to="item.to"
                            class="inline-flex min-w-0 items-center justify-center gap-1.5 whitespace-nowrap rounded-md border border-blue-400/20 bg-blue-500/[0.08] px-2.5 py-2 text-center text-xs font-semibold text-blue-300 transition-colors hover:border-blue-400/40 hover:bg-blue-500/[0.14] hover:text-blue-200 lg:min-w-max lg:flex-auto"
                            @click="close"
                        >
                            <Icon v-if="item.icon" :name="item.icon" class="shrink-0 text-sm" />
                            <span>{{ item.name }}</span>
                        </NuxtLink>
                    </div>

                    <div
                        class="relative z-10 grid w-full grid-cols-3 gap-x-4 gap-y-4 sm:grid-cols-4 lg:grid-cols-12"
                    >
                        <div v-for="(col, columnIndex) in columns" :key="col.label"
                            class="min-w-0 px-2"
                            :class="columnIndex < 6 ? 'lg:col-span-2' : 'lg:col-span-3'">
                            <div class="mb-1 flex items-center gap-2 border-b border-white/[0.08] px-1 pb-2 text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80">
                                <span>{{ col.label }}</span>
                                <span class="ml-auto font-mono text-[9px] tracking-normal text-gray-700">{{ String(columnIndex + 1).padStart(2, '0') }}</span>
                            </div>
                            <div class="pt-1 space-y-px">
                                <NuxtLink
                                    v-for="item in col.items"
                                    :key="item.to"
                                    :to="item.to"
                                    class="group/link flex items-center overflow-hidden rounded-md px-2 py-0.5 text-xs leading-[1.2rem] text-gray-400 transition-colors hover:bg-blue-500/[0.08] hover:text-blue-300"
                                    @click="close"
                                >
                                    <span class="flex w-0 shrink-0 -translate-x-2 items-center justify-center overflow-hidden text-blue-400/80 opacity-0 transition-all duration-200 group-hover/link:mr-1.5 group-hover/link:w-3.5 group-hover/link:translate-x-0 group-hover/link:opacity-100">
                                        <Icon :name="columnIcon(col.label)" class="text-xs" />
                                    </span>
                                    <span>{{ item.name }}</span>
                                </NuxtLink>
                            </div>
                        </div>
                    </div>
                </template>

                <!-- ===== SIMPLE LIST MODE (from items prop) ===== -->
                <template v-else-if="items?.length">
                    <NuxtLink
                        v-for="item in items"
                        :key="item.to"
                        :to="item.to"
                        class="flex items-center gap-2.5 px-3 py-2 rounded-md text-xs text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.08] transition-colors"
                        @click="close"
                    >
                        <Icon v-if="item.icon" :name="item.icon" class="text-sm opacity-50" />
                        {{ item.name }}
                    </NuxtLink>
                </template>

                <!-- ===== SLOT MODE (fully custom content) ===== -->
                <template v-else>
                    <div v-if="searchable" class="px-1.5 pb-1.5 mb-1 border-b border-white/[0.06]">
                        <input
                            ref="searchInputRef"
                            v-model="search"
                            type="text"
                            :placeholder="searchPlaceholder || 'Search...'"
                            class="w-full px-2.5 py-1.5 text-xs rounded-md bg-white/[0.04] border border-white/[0.08] text-gray-300 placeholder-gray-600 outline-none focus:border-blue-500/40"
                            @keydown.stop
                        >
                    </div>
                    <slot :close="close" :search="search" />
                </template>
            </div>
        </Transition>
    </div>
</template>
