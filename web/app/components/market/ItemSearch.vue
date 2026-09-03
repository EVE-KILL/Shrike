<script setup lang="ts">
interface SearchHit {
    id: string
    name: string
    type: 'ship' | 'item'
}

const query = ref('')
const loading = ref(false)
const open = ref(false)
const results = ref<SearchHit[]>([])
let timer: ReturnType<typeof setTimeout> | undefined

function numericId(hit: SearchHit): string {
    return hit.id.split('_').slice(1).join('_')
}

function search() {
    if (timer) clearTimeout(timer)
    const value = query.value.trim()
    if (value.length < 2) {
        results.value = []
        open.value = false
        return
    }
    timer = setTimeout(async () => {
        loading.value = true
        try {
            const [ships, items] = await Promise.all([
                apiFetch<{ hits: SearchHit[] }>('/api/search', { query: { q: value, type: 'ship', limit: 8 } }),
                apiFetch<{ hits: SearchHit[] }>('/api/search', { query: { q: value, type: 'item', limit: 8 } }),
            ])
            const unique = new Map<string, SearchHit>()
            for (const hit of [...(ships.hits ?? []), ...(items.hits ?? [])]) unique.set(hit.id, hit)
            results.value = [...unique.values()].slice(0, 12)
            open.value = true
        } catch {
            results.value = []
            open.value = true
        } finally {
            loading.value = false
        }
    }, 220)
}

async function choose(hit: SearchHit) {
    open.value = false
    query.value = hit.name
    await navigateTo(`/market/item/${numericId(hit)}`)
}
</script>

<template>
    <div class="relative">
        <div class="flex items-center rounded-md border border-white/[0.08] bg-black/30 focus-within:border-blue-500/40">
            <Icon name="lucide:search" class="ml-3 h-4 w-4 text-gray-600" />
            <input
                v-model="query"
                type="search"
                placeholder="Search market items…"
                class="min-w-0 flex-1 bg-transparent px-2.5 py-2 text-xs text-white outline-none placeholder:text-gray-600"
                @input="search"
                @focus="results.length && (open = true)"
                @keydown.esc="open = false"
            >
            <Icon v-if="loading" name="lucide:loader-circle" class="mr-3 h-3.5 w-3.5 animate-spin text-blue-400" />
        </div>

        <div v-if="open" class="absolute left-0 right-0 top-full z-50 mt-1 overflow-hidden rounded-md border border-white/10 bg-[#121316] shadow-2xl">
            <button
                v-for="hit in results"
                :key="hit.id"
                type="button"
                class="flex w-full items-center gap-2.5 border-b border-white/[0.04] px-3 py-2 text-left last:border-0 hover:bg-blue-500/[0.08]"
                @click="choose(hit)"
            >
                <img :src="`/images/types/${numericId(hit)}/icon?size=32`" alt="" class="h-7 w-7 rounded">
                <span class="min-w-0 flex-1 truncate text-xs text-gray-200">{{ hit.name }}</span>
                <span class="text-fine uppercase tracking-wider text-gray-600">{{ hit.type }}</span>
            </button>
            <div v-if="!loading && results.length === 0" class="px-3 py-4 text-center text-xs text-gray-600">
                No market items found.
            </div>
        </div>
    </div>
</template>
