<script setup lang="ts">
/**
 * Shared kill-tab layout (combined/kills/losses) for the entity pages:
 * context-aware top-list sidebar (with loading skeleton) + the killlist.
 * The parent page owns the `v-if` on the active tab.
 */
defineProps<{
    kind: 'character' | 'corporation' | 'alliance'
    entityId: number
    /** Current tab id — drives which sidebar top lists show */
    activeTab: string
    /** Top-lists payload from useEntityPage (null while loading → skeleton) */
    topLists: any
    killlistRole: 'any' | 'attacker' | 'victim'
    extraParams?: Record<string, any>
}>()
</script>

<template>
    <div class="grid grid-cols-1 md:grid-cols-[250px_1fr] gap-4">
        <!-- Sidebar — context-aware top lists (hidden on mobile) -->
        <div v-if="topLists" class="hidden md:block space-y-0">
            <template v-if="activeTab === 'combined' || activeTab === 'kills'">
                <TopBox v-if="topLists.killed?.characters?.length" title="Most Killed Characters" data-type="characters" :entries="topLists.killed.characters" count-color="text-isk/70" />
                <TopBox v-if="topLists.killed?.corporations?.length" title="Most Killed Corps" data-type="corporations" :entries="topLists.killed.corporations" count-color="text-isk/70" />
                <TopBox v-if="topLists.killed?.alliances?.length" title="Most Killed Alliances" data-type="alliances" :entries="topLists.killed.alliances" count-color="text-isk/70" />
            </template>
            <template v-if="activeTab === 'combined' || activeTab === 'losses'">
                <TopBox v-if="topLists.killedBy?.characters?.length" title="Most Killed By (Characters)" data-type="characters" :entries="topLists.killedBy.characters" count-color="text-red-400/70" />
                <TopBox v-if="topLists.killedBy?.corporations?.length" title="Most Killed By (Corps)" data-type="corporations" :entries="topLists.killedBy.corporations" count-color="text-red-400/70" />
                <TopBox v-if="topLists.killedBy?.alliances?.length" title="Most Killed By (Alliances)" data-type="alliances" :entries="topLists.killedBy.alliances" count-color="text-red-400/70" />
            </template>
        </div>
        <div v-else class="hidden md:block space-y-4">
            <div v-for="i in 3" :key="i" class="space-y-1">
                <div class="h-3 w-24 bg-white/[0.04] rounded animate-pulse mb-2"></div>
                <div v-for="j in 5" :key="j" class="h-6 bg-white/[0.04] rounded animate-pulse"></div>
            </div>
        </div>

        <div>
            <KillList :entity-endpoint="`/api/entity/${kind}/${entityId}/killlist`"
                :victim-entity-type="kind" :victim-entity-id="entityId" :entity-role="killlistRole"
                :extra-params="extraParams"
                :stream-topics="[`victim.${entityId}`, `attacker.${entityId}`]" />
        </div>
    </div>
</template>
