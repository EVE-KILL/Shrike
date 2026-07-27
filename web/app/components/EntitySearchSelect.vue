<script setup lang="ts">
import { onClickOutside } from '@vueuse/core'

interface Entity {
    id: number
    name: string
    type: 'character' | 'corporation'
}

const props = defineProps<{
    modelValue: Entity | null
    placeholder?: string
    searchUrl?: string
}>()

const emit = defineEmits(['update:modelValue'])

const isOpen = ref(false)
const searchQuery = ref('')
const containerRef = ref<HTMLElement | null>(null)
const inputRef = ref<HTMLInputElement | null>(null)
const results = ref<Entity[]>([])
const loading = ref(false)

const debouncedQuery = refDebounced(searchQuery, 250)

onClickOutside(containerRef, () => { isOpen.value = false })

watch(debouncedQuery, async (q) => {
    if (q.length < 2) { results.value = []; return }
    loading.value = true
    try {
        const url = props.searchUrl || '/api/admin/esi-entities'
        const data = await apiFetch<{ results: Entity[] }>(url, { params: { q } })
        results.value = data.results ?? []
    } catch {
        results.value = []
    } finally {
        loading.value = false
    }
})

const select = (entity: Entity) => {
    emit('update:modelValue', entity)
    isOpen.value = false
    searchQuery.value = ''
    results.value = []
}

const clear = () => {
    emit('update:modelValue', null)
    searchQuery.value = ''
    results.value = []
}

const open = () => {
    isOpen.value = true
    nextTick(() => inputRef.value?.focus())
}

const onKeydown = (e: KeyboardEvent) => {
    if (e.key === 'Escape' && isOpen.value) { isOpen.value = false }
}

const imageUrl = (entity: Entity) => {
    return entity.type === 'character'
        ? `/images/characters/${entity.id}/portrait?size=32`
        : `/images/corporations/${entity.id}/logo?size=32`
}
</script>

<template>
    <div ref="containerRef" class="relative inline-flex">
        <!-- Selected state / trigger -->
        <button
            v-if="!isOpen"
            class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs transition-colors cursor-pointer border max-w-[200px]"
            :class="modelValue
                ? 'bg-white/[0.06] text-gray-300 border-white/[0.08] hover:border-white/[0.12]'
                : 'bg-white/[0.04] text-gray-500 border-white/[0.06] hover:border-white/[0.10] hover:text-blue-400'"
            @click="open"
        >
            <template v-if="modelValue">
                <EveImage :src="imageUrl(modelValue)" :size="16" :alt="modelValue.name" class="w-4 h-4 rounded-full flex-shrink-0" />
                <span class="truncate">{{ modelValue.name }}</span>
                <button
                    class="ml-0.5 text-gray-600 hover:text-blue-400 transition-colors flex-shrink-0"
                    @click.stop="clear"
                >
                    <Icon name="lucide:x" class="text-[10px]" />
                </button>
            </template>
            <template v-else>
                <Icon name="lucide:search" class="text-[10px] opacity-60" />
                <span>{{ placeholder ?? 'Search entity...' }}</span>
            </template>
        </button>

        <!-- Search input (when open) -->
        <div v-if="isOpen" class="flex items-center gap-1.5 bg-white/[0.08] border border-white/[0.12] rounded-lg px-2.5 py-1.5 min-w-[200px]">
            <Icon name="lucide:search" class="text-gray-500 text-xs flex-shrink-0" />
            <input
                ref="inputRef"
                v-model="searchQuery"
                type="text"
                :placeholder="placeholder ?? 'Search character or corp...'"
                class="bg-transparent text-xs text-white placeholder-gray-600 outline-none w-full"
                @keydown="onKeydown"
            >
            <Icon v-if="loading" name="lucide:loader-2" class="text-xs text-gray-500 animate-spin flex-shrink-0" />
        </div>

        <!-- Results dropdown -->
        <Transition
            enter-active-class="transition duration-150 ease-out"
            enter-from-class="opacity-0 translate-y-1 scale-[0.98]"
            enter-to-class="opacity-100 translate-y-0 scale-100"
            leave-active-class="transition duration-100 ease-in"
            leave-from-class="opacity-100 translate-y-0 scale-100"
            leave-to-class="opacity-0 translate-y-1 scale-[0.98]"
        >
            <div
                v-if="isOpen && (results.length > 0 || (debouncedQuery.length >= 2 && !loading))"
                class="absolute top-full left-0 mt-1.5 z-50 rounded-xl border border-white/[0.10] bg-[#141414]/95 backdrop-blur-2xl shadow-2xl shadow-black/60 overflow-y-auto max-h-[50vh] p-1.5 min-w-[220px]"
            >
                <button
                    v-for="r in results"
                    :key="`${r.type}_${r.id}`"
                    class="flex items-center gap-2.5 w-full px-2.5 py-2 rounded-md text-xs text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors text-left cursor-pointer"
                    @click="select(r)"
                >
                    <EveImage :src="imageUrl(r)" :size="32" :alt="r.name" class="w-5 h-5 rounded-full flex-shrink-0" />
                    <div class="flex-1 min-w-0">
                        <div class="truncate">{{ r.name }}</div>
                    </div>
                    <span class="text-[10px] text-gray-600 uppercase flex-shrink-0">{{ r.type }}</span>
                </button>
                <div v-if="results.length === 0 && !loading" class="px-2.5 py-3 text-xs text-gray-600 text-center">
                    No matching entities with ESI logs
                </div>
            </div>
        </Transition>
    </div>
</template>
