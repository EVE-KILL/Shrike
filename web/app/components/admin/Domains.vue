<script setup lang="ts">
// Domains section state
const { data: domainsData, refresh: refreshDomains } = useApiFetch<{ domains: any[] }>('/api/admin/domains', {
    immediate: true,
    lazy: true,
})

// The parent keeps this component alive (<KeepAlive>) across tab switches.
// Matching the old tab-switch watcher: on re-activation, refetch only if no
// data ever loaded (e.g. the first fetch errored).
let activatedOnce = false
onActivated(() => {
    if (!activatedOnce) { activatedOnce = true; return }
    if (!domainsData.value) refreshDomains()
})

/**
 * Filters. The endpoint returns every domain in one unpaged response, so this
 * is all client-side — no refetch, no debounce needed.
 *
 * This table had neither a search box nor a filter: finding one domain among
 * them meant Ctrl+F, and the pending-approval queue — the one thing here that
 * needs acting on — could only be spotted by scanning the Pending column.
 */
const search = ref('')
const filter = ref<'all' | 'pending' | 'inactive'>('all')

const allDomains = computed(() => domainsData.value?.domains ?? [])

const pendingCount = computed(() => allDomains.value.filter(d => (d.pending_assets ?? 0) > 0).length)
const inactiveCount = computed(() => allDomains.value.filter(d => !d.active).length)

const domains = computed(() => {
    const q = search.value.trim().toLowerCase()
    return allDomains.value.filter((d) => {
        if (filter.value === 'pending' && !(d.pending_assets > 0)) return false
        if (filter.value === 'inactive' && d.active) return false
        if (!q) return true
        return [d.subdomain, d.site_name, d.owner_name, d.custom_hostname]
            .some((v: string | null) => v?.toLowerCase().includes(q))
    })
})

const filters = computed(() => [
    { value: 'all' as const, label: 'All', count: allDomains.value.length },
    { value: 'pending' as const, label: 'Pending assets', count: pendingCount.value },
    { value: 'inactive' as const, label: 'Inactive', count: inactiveCount.value },
])
</script>

<template>
    <!-- ═══════════════ DOMAINS ═══════════════ -->
    <div class="space-y-4">
        <div class="glass-panel p-4 flex flex-col md:flex-row md:items-center gap-3">
            <div class="flex items-center gap-3 flex-1 min-w-0">
                <Icon name="lucide:search" class="text-gray-500 flex-shrink-0" />
                <input
                    v-model="search"
                    type="text"
                    placeholder="Search subdomain, site name or owner..."
                    class="flex-1 min-w-0 bg-transparent text-sm text-white placeholder-gray-600 outline-none"
                >
            </div>
            <div class="flex items-center gap-1 rounded-md bg-white/[0.03] border border-white/[0.06] p-1 flex-shrink-0">
                <button
                    v-for="f in filters" :key="f.value"
                    class="px-3 py-1 rounded text-xs font-medium transition-colors cursor-pointer flex items-center gap-1.5"
                    :class="filter === f.value ? 'bg-blue-500/[0.15] text-blue-300' : 'text-gray-400 hover:text-white'"
                    @click="filter = f.value"
                >
                    {{ f.label }}
                    <span class="tabular-nums" :class="filter === f.value ? 'text-blue-400/70' : 'text-gray-600'">{{ f.count }}</span>
                </button>
            </div>
        </div>

        <div class="glass-panel overflow-hidden">
            <div class="overflow-x-auto">
                <table class="w-full text-sm">
                    <thead>
                        <tr class="border-b border-white/[0.06] text-left text-fine text-gray-500 uppercase tracking-wider">
                            <th class="px-4 py-3">Domain</th>
                            <th class="px-4 py-3 hidden md:table-cell">Owner</th>
                            <th class="px-4 py-3 hidden md:table-cell">Entities</th>
                            <th class="px-4 py-3">Status</th>
                            <th class="px-4 py-3">Pending</th>
                            <th class="px-4 py-3"></th>
                        </tr>
                    </thead>
                    <tbody>
                        <tr v-for="d in domains" :key="d.id" class="border-b border-white/[0.04] hover:bg-white/[0.02] transition-colors">
                            <td class="px-4 py-3">
                                <div class="font-medium text-white">{{ d.site_name || d.subdomain }}</div>
                                <div class="text-fine text-gray-500">{{ d.subdomain }}.eve-kill.com</div>
                            </td>
                            <td class="px-4 py-3 hidden md:table-cell">
                                <NuxtLink :to="`/character/${d.user_id}`" class="flex items-center gap-2 text-gray-300 hover:text-blue-400">
                                    <img :src="`/images/characters/${d.user_id}/portrait?size=32`" class="w-5 h-5 rounded-full" loading="lazy">
                                    {{ d.owner_name || d.user_id }}
                                </NuxtLink>
                            </td>
                            <td class="px-4 py-3 hidden md:table-cell text-gray-400">{{ (d.entities || []).length }}</td>
                            <td class="px-4 py-3">
                                <span :class="d.active ? 'text-green-400' : 'text-red-400'" class="text-xs font-medium">
                                    {{ d.active ? 'Active' : 'Inactive' }}
                                </span>
                            </td>
                            <td class="px-4 py-3">
                                <span v-if="d.pending_assets > 0" class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-yellow-500/20 text-yellow-400 text-fine font-medium">
                                    <Icon name="lucide:clock" class="text-xs" />
                                    {{ d.pending_assets }}
                                </span>
                                <span v-else class="text-fine text-gray-600">0</span>
                            </td>
                            <td class="px-4 py-3 text-right">
                                <NuxtLink :to="`/admin/domains/${d.id}`" class="px-3 py-1.5 rounded-md text-fine font-medium text-blue-400 hover:bg-blue-500/10 transition-colors">
                                    Manage
                                </NuxtLink>
                            </td>
                        </tr>
                        <tr v-if="!domains.length">
                            <td colspan="6" class="px-4 py-8 text-center text-gray-500 text-sm">
                                {{ allDomains.length ? 'No domains match these filters.' : 'No custom domains found' }}
                            </td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>
    </div>
</template>
