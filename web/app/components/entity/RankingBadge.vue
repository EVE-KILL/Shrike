<script setup lang="ts">
defineProps<{
    ranking?: {
        eve_kill_rating: number
        overall_rank: number
        population: number
        combat_points: number
    } | null
}>()
</script>

<template>
    <NuxtLink v-if="ranking" to="/rankings"
        class="block min-w-[132px] rounded-lg border border-blue-500/20 bg-blue-500/[0.07] px-3 py-2 text-right hover:border-blue-400/40 transition-colors">
        <div class="text-fine uppercase tracking-[0.16em] text-blue-400">EVE-KILL Rating</div>
        <div class="text-xl font-bold tabular-nums text-white">{{ ranking.eve_kill_rating.toLocaleString('en-US') }}</div>
        <div class="text-fine text-gray-500 tabular-nums">#{{ ranking.overall_rank.toLocaleString('en-US') }} of {{ ranking.population.toLocaleString('en-US') }}</div>
    </NuxtLink>
</template>
