<script setup lang="ts">
const route = useRoute()
const id = Number(route.params.id)

if (!Number.isInteger(id) || id < 1 || id > 2147483647) {
    throw createError({ statusCode: 404, statusMessage: 'Group not found' })
}

interface GroupType {
    type_id: number
    name: string | null
    description: string | null
    published: boolean
    meta_group_id: number | null
    meta_group_name: string | null
    volume: number | null
    mass: number | null
    base_price: number | null
}

interface GroupResponse {
    group: {
        group_id: number
        name: string | null
        category_id: number | null
        category_name: string | null
        published: boolean
        icon_id: number | null
        type_count: number
        published_type_count: number
    }
    types: GroupType[]
}

const { data, pending, error } = await useApiFetch<GroupResponse>(`/api/group/${id}`)
if (error.value) {
    throw createError({
        statusCode: error.value.statusCode || 404,
        statusMessage: (error.value.data as any)?.message || 'Group not found',
    })
}

const group = computed(() => data.value?.group)
const types = computed(() => data.value?.types ?? [])
const publishedTypes = computed(() => types.value.filter(type => type.published))
const unpublishedTypes = computed(() => types.value.filter(type => !type.published))

useHead({ title: computed(() => group.value?.name ? `${group.value.name} — Item Group` : 'Item Group') })
useSeoMeta({
    description: computed(() => group.value?.name
        ? `${group.value.name} is an EVE Online ${group.value.category_name ?? 'inventory'} group containing ${group.value.published_type_count.toLocaleString('en-US')} published types. Browse its ships and items on EVE-KILL.`
        : 'Browse an EVE Online inventory group on EVE-KILL.'),
    ogTitle: computed(() => group.value?.name ? `${group.value.name} — EVE Online item group` : 'Item Group — EVE-KILL'),
    ogDescription: computed(() => group.value?.name
        ? `Browse ${group.value.published_type_count.toLocaleString('en-US')} published types in the ${group.value.name} group.`
        : 'Browse an EVE Online inventory group.'),
    ogType: 'website',
})

useSchemaOrg([defineBreadcrumb(computed(() => ({
    itemListElement: [
        { name: 'Home', item: '/' },
        { name: group.value?.category_name ?? 'Items', item: '/market' },
        { name: group.value?.name ?? 'Group', item: `/group/${id}` },
    ],
})))])
</script>

<template>
    <div>
        <EntityHeader v-if="pending" loading />

        <div v-else-if="group">
            <EntityHeader>
                <template #image>
                    <div class="grid grid-cols-2 gap-1 w-24 h-24 md:w-32 md:h-32 rounded-lg overflow-hidden bg-white/[0.04] p-1">
                        <img v-for="type in publishedTypes.slice(0, 4)" :key="type.type_id"
                            :src="`/images/types/${type.type_id}/icon?size=64`" :alt="type.name ?? ''"
                            class="w-full h-full min-w-0 min-h-0 object-contain rounded bg-black/20" loading="eager">
                        <div v-if="!publishedTypes.length" class="col-span-2 flex items-center justify-center text-gray-600">
                            <Icon name="lucide:boxes" class="w-10 h-10" />
                        </div>
                    </div>
                </template>

                <div class="flex items-center gap-3 mb-1 flex-wrap">
                    <h1 class="text-2xl md:text-3xl font-bold text-white">{{ group.name }}</h1>
                    <span v-if="!group.published" class="px-2 py-0.5 rounded text-fine font-bold uppercase tracking-wider text-amber-400 bg-white/[0.06]">Unpublished</span>
                </div>
                <div class="flex items-center gap-1.5 text-xs text-gray-500 mb-3">
                    <NuxtLink to="/market" class="hover:text-blue-400 transition-colors">Market</NuxtLink>
                    <span class="text-gray-700">/</span>
                    <span>{{ group.category_name ?? 'Uncategorized' }}</span>
                    <span class="text-gray-700">/</span>
                    <span class="text-gray-400">{{ group.name }}</span>
                </div>
                <p class="text-sm text-gray-400 max-w-2xl">
                    EVE Online inventory group in the {{ group.category_name ?? 'uncategorized' }} category.
                </p>

                <template #stats>
                    <EntityStatGrid variant="inline" :columns="4">
                        <div>
                            <div class="text-fine uppercase tracking-wider text-gray-500">Published Types</div>
                            <div class="text-base font-bold text-white tabular-nums">{{ group.published_type_count.toLocaleString('en-US') }}</div>
                        </div>
                        <div>
                            <div class="text-fine uppercase tracking-wider text-gray-500">All Types</div>
                            <div class="text-base font-bold text-gray-300 tabular-nums">{{ group.type_count.toLocaleString('en-US') }}</div>
                        </div>
                        <div>
                            <div class="text-fine uppercase tracking-wider text-gray-500">Group ID</div>
                            <div class="text-base font-bold text-gray-300 tabular-nums">{{ group.group_id }}</div>
                        </div>
                        <div v-if="group.category_id">
                            <div class="text-fine uppercase tracking-wider text-gray-500">Category</div>
                            <div class="text-base font-bold text-blue-400">{{ group.category_name }}</div>
                        </div>
                    </EntityStatGrid>
                </template>
            </EntityHeader>

            <div class="glass-panel p-3.5">
                <div class="flex items-center gap-2 mb-3 pb-2 border-b border-white/[0.04]">
                    <Icon name="lucide:boxes" class="w-4 h-4 text-blue-400" />
                    <h2 class="text-xs font-semibold text-gray-300">Published Types</h2>
                    <span class="text-fine text-gray-600">({{ publishedTypes.length }})</span>
                </div>

                <div v-if="publishedTypes.length" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-2">
                    <NuxtLink v-for="type in publishedTypes" :key="type.type_id" :to="`/item/${type.type_id}`"
                        class="flex items-center gap-3 p-2 rounded-md border border-white/[0.05] hover:border-blue-500/30 hover:bg-blue-500/[0.06] transition-colors min-w-0">
                        <img :src="`/images/types/${type.type_id}/icon?size=64`" :alt="type.name ?? ''"
                            class="w-12 h-12 rounded shrink-0" loading="lazy">
                        <div class="min-w-0">
                            <div class="text-sm text-gray-200 truncate">{{ type.name ?? `Type #${type.type_id}` }}</div>
                            <div class="text-fine text-gray-600 truncate">
                                <span v-if="type.meta_group_name">{{ type.meta_group_name }} · </span>#{{ type.type_id }}
                            </div>
                        </div>
                    </NuxtLink>
                </div>
                <div v-else class="py-8 text-center text-sm text-gray-500">This group has no published types.</div>

                <details v-if="unpublishedTypes.length" class="mt-4 pt-3 border-t border-white/[0.04]">
                    <summary class="text-xs text-gray-500 hover:text-gray-300 cursor-pointer">Show {{ unpublishedTypes.length }} unpublished types</summary>
                    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-2 mt-3">
                        <NuxtLink v-for="type in unpublishedTypes" :key="type.type_id" :to="`/item/${type.type_id}`"
                            class="flex items-center gap-2 p-2 rounded-md text-xs text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.04] transition-colors">
                            <img :src="`/images/types/${type.type_id}/icon?size=32`" :alt="type.name ?? ''" class="w-8 h-8 rounded opacity-60" loading="lazy">
                            <span class="truncate">{{ type.name ?? `Type #${type.type_id}` }}</span>
                        </NuxtLink>
                    </div>
                </details>
            </div>
        </div>
    </div>
</template>
