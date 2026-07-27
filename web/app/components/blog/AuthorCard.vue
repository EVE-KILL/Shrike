<script setup lang="ts">
defineProps<{
    authorId: number
    authorName: string
    corporationId?: number | null
    corporationName?: string | null
    allianceId?: number | null
    allianceName?: string | null
    publishedAt?: string | null
}>()

function fmtDate(iso: string | null | undefined): string {
    if (!iso) return ''
    return new Date(iso).toLocaleDateString(undefined, { year: 'numeric', month: 'long', day: 'numeric' })
}
</script>

<template>
    <div class="rounded-xl bg-white/[0.03] border border-white/[0.08] p-5">
        <div class="flex items-start gap-4">
            <NuxtLink
                :to="`/character/${authorId}`"
                class="flex-shrink-0"
            >
                <img
                    :src="`/images/characters/${authorId}/portrait?size=128`"
                    :alt="authorName"
                    class="w-16 h-16 rounded-lg border border-white/[0.08]"
                    loading="lazy"
                >
            </NuxtLink>
            <div class="min-w-0 flex-1">
                <div class="text-[10px] uppercase tracking-wider text-gray-500 mb-0.5">Written by</div>
                <NuxtLink
                    :to="`/character/${authorId}`"
                    class="text-base font-semibold text-white hover:text-blue-300 transition-colors"
                >
                    {{ authorName }}
                </NuxtLink>

                <div v-if="corporationId || allianceId" class="mt-2 space-y-1 text-sm">
                    <div v-if="corporationId" class="flex items-center gap-2 text-gray-400">
                        <img
                            :src="`/images/corporations/${corporationId}/logo?size=32`"
                            :alt="corporationName || 'Corporation'"
                            class="w-4 h-4 rounded"
                            loading="lazy"
                        >
                        <NuxtLink
                            :to="`/corporation/${corporationId}`"
                            class="hover:text-white transition-colors truncate"
                        >
                            {{ corporationName || `Corporation ${corporationId}` }}
                        </NuxtLink>
                    </div>
                    <div v-if="allianceId" class="flex items-center gap-2 text-gray-400">
                        <img
                            :src="`/images/alliances/${allianceId}/logo?size=32`"
                            :alt="allianceName || 'Alliance'"
                            class="w-4 h-4 rounded"
                            loading="lazy"
                        >
                        <NuxtLink
                            :to="`/alliance/${allianceId}`"
                            class="hover:text-white transition-colors truncate"
                        >
                            {{ allianceName || `Alliance ${allianceId}` }}
                        </NuxtLink>
                    </div>
                </div>

                <div v-if="publishedAt" class="mt-2 text-xs text-gray-500">
                    Posted {{ fmtDate(publishedAt) }}
                </div>
            </div>
        </div>
    </div>
</template>
