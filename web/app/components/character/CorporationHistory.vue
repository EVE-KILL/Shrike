<script setup lang="ts">
const props = defineProps<{
    history: any[]
    queued?: boolean
}>()

const entries = computed(() => {
    return props.history.map((entry: any, idx: number) => {
        const start = new Date(entry.start_date)
        const end = idx > 0 ? new Date(props.history[idx - 1].start_date) : new Date()
        const totalDays = Math.floor((end.getTime() - start.getTime()) / 86400000)
        const years = Math.floor(totalDays / 365)
        const months = Math.floor((totalDays % 365) / 30)
        const days = totalDays % 30
        let duration = ''
        if (years > 0) duration += `${years}y `
        if (months > 0) duration += `${months}m `
        if (years === 0 && months === 0) duration = `${days}d`
        const isCurrent = idx === 0
        return { ...entry, start, end, duration: duration.trim(), totalDays, isCurrent }
    })
})
</script>

<template>
    <div>
        <div v-if="entries.length > 0" class="space-y-2">
            <NuxtLink v-for="entry in entries" :key="entry.start_date"
                :to="`/corporation/${entry.corporation_id}`"
                class="flex items-center gap-4 p-3 rounded-lg border transition-colors hover:bg-blue-500/[0.04]"
                :class="entry.isCurrent ? 'border-blue-500/20 bg-blue-500/[0.04]' : 'border-white/[0.08]'">
                <div class="flex-shrink-0">
                    <img :src="`/images/corporations/${entry.corporation_id}/logo?size=128`"
                        :alt="entry.corporation_name"
                        class="w-12 h-12 rounded-lg"
                        loading="lazy">
                </div>
                <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-2">
                        <span class="text-xs text-gray-300 truncate">{{ entry.corporation_name }}</span>
                        <span class="text-fine text-gray-600">[{{ entry.corporation_ticker }}]</span>
                        <span v-if="entry.isCurrent" class="text-fine uppercase tracking-wider px-1.5 py-0.5 rounded bg-blue-500/20 text-blue-400 font-medium">Current</span>
                    </div>
                    <div class="text-fine text-gray-500 mt-0.5">
                        {{ formatDate(entry.start_date) }}
                        <span v-if="!entry.isCurrent"> &mdash; {{ formatDate(entry.end.toISOString()) }}</span>
                        <span v-else> &mdash; Present</span>
                    </div>
                </div>
                <div v-if="entry.kills > 0 || entry.losses > 0" class="flex-shrink-0 flex items-center gap-3 text-fine tabular-nums">
                    <span class="text-green-400">{{ entry.kills.toLocaleString('en-US') }} <span class="text-gray-600">kills</span></span>
                    <span class="text-red-400">{{ entry.losses.toLocaleString('en-US') }} <span class="text-gray-600">losses</span></span>
                </div>
                <div class="flex-shrink-0 text-right">
                    <div class="text-fine text-gray-400 tabular-nums">{{ entry.duration }}</div>
                    <div class="text-fine text-gray-600">{{ entry.totalDays.toLocaleString('en-US') }} days</div>
                </div>
            </NuxtLink>
        </div>

        <div v-else-if="queued" class="text-sm text-center py-8">
            <p class="text-gray-400">Corporation history has been queued for retrieval.</p>
            <p class="text-gray-600 mt-1">Refresh in a moment to see the data.</p>
        </div>

        <div v-else class="text-sm text-gray-600 text-center py-8">
            No corporation history available
        </div>
    </div>
</template>
