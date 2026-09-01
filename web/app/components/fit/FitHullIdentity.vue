<script setup lang="ts">
const props = withDefaults(defineProps<{
    compact?: boolean;
}>(), { compact: false });

const { currentFit } = useCurrentFit();
const { sde } = useEveData();
const { issues, isValid } = useFitValidity();

const hull = computed(() => {
    const fit = currentFit.value;
    if (!fit?.shipTypeId || !sde.value) return null;
    const type = sde.value.types.get(fit.shipTypeId);
    if (!type) return null;
    return {
        typeId: fit.shipTypeId,
        name: type.name,
        group: sde.value.groups.get(type.groupID)?.name ?? "Ship",
        tech: type.metaGroupID === 2 ? "Tech II" : type.metaGroupID === 3 ? "Storyline" : type.metaGroupID === 4 ? "Faction" : type.metaGroupID === 14 ? "Tech III" : "Tech I",
        fitName: fit.name,
    };
});
</script>

<template>
    <div v-if="hull && compact" class="ml-auto flex min-w-0 items-center gap-2 rounded-md border border-white/[0.06] bg-white/[0.025] py-0.5 pl-1 pr-2">
        <img :src="`https://images.evetech.net/types/${hull.typeId}/render?size=64`" :alt="hull.name" class="h-7 w-7 rounded object-contain" />
        <div class="min-w-0 text-right leading-tight">
            <div class="truncate text-fine font-semibold text-gray-300">{{ hull.name }}</div>
            <div class="truncate text-[9px] text-gray-600">{{ hull.group }}</div>
        </div>
    </div>

    <div v-else-if="hull" class="glass-panel mb-2 flex shrink-0 items-center gap-3 overflow-hidden p-2.5">
        <div class="relative h-14 w-14 shrink-0 overflow-hidden rounded-lg border border-white/[0.07] bg-black/30">
            <img :src="`https://images.evetech.net/types/${hull.typeId}/render?size=128`" :alt="hull.name" class="h-full w-full object-contain" />
            <div class="absolute inset-x-0 bottom-0 h-5 bg-gradient-to-t from-black/80 to-transparent" />
        </div>
        <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
                <div class="truncate text-sm font-semibold text-gray-100">{{ hull.name }}</div>
                <span class="shrink-0 rounded bg-blue-400/10 px-1.5 py-0.5 text-[9px] font-bold uppercase tracking-wider text-blue-300">{{ hull.tech }}</span>
            </div>
            <div class="mt-0.5 truncate text-fine text-gray-500">{{ hull.group }} · {{ hull.fitName }}</div>
            <div class="mt-1 flex items-center gap-1 text-[9px] font-bold uppercase tracking-[0.1em]" :class="isValid ? 'text-emerald-400' : 'text-red-400'" :title="issues.map(issue => issue.label).join(' · ')">
                <Icon :name="isValid ? 'lucide:circle-check' : 'lucide:triangle-alert'" class="h-3 w-3" />
                {{ isValid ? "Fit valid" : `${issues.length} fitting issues` }}
            </div>
        </div>
    </div>
</template>
