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
</script>

<template>
    <div ref="containerRef" class="relative inline-flex">
        <!-- Trigger -->
        <div @click="isOpen = !isOpen">
            <slot name="trigger" />
        </div>

        <!-- Panel -->
        <Transition
            enter-active-class="transition duration-200 ease-out"
            enter-from-class="opacity-0 translate-y-1 scale-[0.98]"
            enter-to-class="opacity-100 translate-y-0 scale-100"
            leave-active-class="transition duration-150 ease-in"
            leave-from-class="opacity-100 translate-y-0 scale-100"
            leave-to-class="opacity-0 translate-y-1 scale-[0.98]"
        >
            <div
                v-if="isOpen"
                class="absolute top-full mt-2 z-50 rounded-xl border border-white/[0.10] bg-[#141414]/95 backdrop-blur-2xl shadow-2xl shadow-black/60 overflow-y-auto max-h-[70vh]"
                :class="[
                    align === 'right' ? 'right-0 origin-top-right' : 'left-0 origin-top-left',
                    isMega ? 'p-5' : 'p-1.5 min-w-[11rem]',
                ]"
            >
                <!-- ===== MEGA MENU MODE ===== -->
                <template v-if="isMega">
                    <div class="flex gap-6">
                        <div v-for="col in columns" :key="col.label" class="min-w-[8.5rem]">
                            <div class="px-2 pb-2 mb-1 text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 border-b border-white/[0.08]">
                                {{ col.label }}
                            </div>
                            <div class="pt-1 space-y-px">
                                <NuxtLink
                                    v-for="item in col.items"
                                    :key="item.to"
                                    :to="item.to"
                                    class="block px-2 py-1.5 rounded-md text-xs text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.08] transition-colors"
                                    @click="close"
                                >
                                    {{ item.name }}
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
