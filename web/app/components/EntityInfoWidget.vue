<script setup lang="ts">
import type { DomainEntity } from '~/composables/useDomainConfig'

defineProps<{
    entities: readonly DomainEntity[]
}>()

const entityImage = (e: DomainEntity) => {
    const base = e.type === 'character' ? 'characters' : e.type === 'corporation' ? 'corporations' : 'alliances'
    const kind = e.type === 'character' ? 'portrait' : 'logo'
    return `/images/${base}/${e.id}/${kind}?size=64`
}

const entityLink = (e: DomainEntity) => {
    return `/${e.type === 'character' ? 'character' : e.type === 'corporation' ? 'corporation' : 'alliance'}/${e.id}`
}
</script>

<template>
    <div class="glass-panel p-4">
        <h3 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-3">Tracked Entities</h3>
        <div class="flex flex-wrap gap-2">
            <NuxtLink
                v-for="entity in entities"
                :key="`${entity.type}-${entity.id}`"
                :to="entityLink(entity)"
                class="flex items-center gap-2 px-3 py-2 rounded-lg bg-white/[0.04] border border-white/[0.06] hover:border-blue-500/30 transition-colors"
            >
                <EveImage :src="entityImage(entity)" :size="32" :alt="entity.name" class="w-8 h-8 rounded-full" />
                <div>
                    <div class="text-sm text-white">{{ entity.name }}</div>
                    <div class="text-fine text-gray-500 capitalize">{{ entity.type }}</div>
                </div>
            </NuxtLink>
        </div>
    </div>
</template>
