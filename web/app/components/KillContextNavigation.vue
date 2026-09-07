<script setup lang="ts">
import { adjacentKill, readKillNavigation, type KillNavigationContext } from '~/utils/killNavigation'
const route = useRoute()
const context = ref<KillNavigationContext | null>(null)
const loaded = ref(false)
const id = computed(() => Number(route.params.id))
function restore() { context.value = readKillNavigation(route.query.nav); loaded.value = true }
onMounted(restore)
watch(() => route.fullPath, restore)
const valid = computed(() => context.value?.ids.includes(id.value) ? context.value : null)
const previous = computed(() => valid.value ? adjacentKill(valid.value, id.value, -1) : null)
const next = computed(() => valid.value ? adjacentKill(valid.value, id.value, 1) : null)
const target = (killID: number) => ({ path: `/kill/${killID}`, query: { nav: String(route.query.nav) } })
</script>

<template>
    <nav v-if="loaded && route.query.nav" aria-label="Killmail result navigation" class="mb-4 rounded-lg border border-white/[0.08] bg-white/[0.03] px-3 py-2">
        <template v-if="valid">
            <div class="flex flex-wrap items-center justify-between gap-3">
                <NuxtLink :to="valid.returnTo" class="text-xs text-gray-400 hover:text-blue-400"><Icon name="lucide:arrow-left" class="mr-1" /> {{ valid.label }}</NuxtLink>
                <div class="flex items-center gap-3 text-xs">
                    <NuxtLink v-if="previous" :to="target(previous)" class="text-gray-200 hover:text-blue-400"><Icon name="lucide:chevron-left" /> Previous</NuxtLink>
                    <span v-else class="text-gray-600" aria-disabled="true">Previous</span>
                    <span class="text-gray-500 tabular-nums">{{ valid.ids.indexOf(id) + 1 }} / {{ valid.ids.length }} on this page</span>
                    <NuxtLink v-if="next" :to="target(next)" class="text-gray-200 hover:text-blue-400">Next <Icon name="lucide:chevron-right" /></NuxtLink>
                    <span v-else class="text-gray-600" aria-disabled="true">Next</span>
                </div>
            </div>
            <p v-if="!previous || !next" class="mt-2 text-xs text-gray-500">{{ !previous ? 'Start' : 'End' }} of this results page. Return to the list to browse another page.</p>
        </template>
        <p v-else class="text-xs text-gray-500">This browsing context has expired or is unavailable. <NuxtLink :to="`/kill/${id}`" class="text-blue-400">View direct killmail link</NuxtLink></p>
    </nav>
</template>
