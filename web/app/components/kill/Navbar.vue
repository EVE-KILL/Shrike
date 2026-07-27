<script setup lang="ts">
const props = defineProps<{
    killmailId: number
    killmailHash: string
    systemName: string | null
    systemId: number
    constellationId: number | null
    constellationName: string | null
    regionName: string | null
    regionId: number | null
    shipTypeId: number | null
    shipGroupId: number | null
    shipGroupName: string | null
    shipName: string | null
    characterId: number | null
    characterName: string | null
    corporationId: number | null
    corporationName: string | null
    allianceId: number | null
    allianceName: string | null
    finalBlowShipTypeId: number | null
    finalBlowShipName: string | null
    topDamageShipTypeId: number | null
    topDamageShipName: string | null
    siblings: {
        killmail_id: number
        ship_type_id: number | null
        ship_group_id: number | null
        ship_name: string | null
        total_value: number
    }[]
}>()

interface ToolLink {
    name: string
    icon: string
    items: { label: string; desc?: string | null; url: string; external?: boolean; disabled?: boolean }[]
}

const toolLinks = computed<ToolLink[]>(() => {
    const k = props
    const links: ToolLink[] = []

    // DOTLAN
    links.push({
        name: 'DOTLAN', icon: '/remotes/dotlan.png',
        items: [
            { label: 'System', desc: k.systemName, url: `https://evemaps.dotlan.net/system/${encodeURIComponent(k.systemName || '')}`, external: true, disabled: !k.systemName },
            { label: 'Constellation', desc: k.constellationName, url: `https://evemaps.dotlan.net/map/${encodeURIComponent(k.regionName || '')}/${encodeURIComponent(k.constellationName || '')}`, external: true, disabled: !k.constellationName },
            { label: 'Region', desc: k.regionName, url: `https://evemaps.dotlan.net/region/${encodeURIComponent(k.regionName || '')}`, external: true, disabled: !k.regionName },
        ],
    })

    // EVEEye
    links.push({
        name: 'EVEEye', icon: '/remotes/eveeye.svg',
        items: [
            { label: 'Region', desc: k.regionName, url: `https://eveeye.com/?m=${encodeURIComponent(k.regionName || '')}`, external: true, disabled: !k.regionName },
        ],
    })

    // EVE Missioneer
    links.push({
        name: 'Missioneer', icon: '/remotes/evemissioneer.png',
        items: [
            { label: 'System', desc: k.systemName, url: `https://evemissioneer.com/s/${k.systemId}`, external: true },
            { label: 'Region', desc: k.regionName, url: `https://evemissioneer.com/r/${k.regionId}`, external: true, disabled: !k.regionId },
        ],
    })

    // EveShip.fit
    links.push({
        name: 'EveShip.fit', icon: '/remotes/eveship-fit.png',
        items: [
            { label: 'Fitting', desc: k.shipName, url: `https://eveship.fit/fit/${k.killmailId}`, external: true, disabled: !k.shipTypeId },
        ],
    })

    // EVERef
    links.push({
        name: 'EVERef', icon: '/remotes/everef.png',
        items: [
            ...(k.shipGroupId ? [{ label: 'Ship Group', desc: k.shipGroupName, url: `https://everef.net/group/${k.shipGroupId}`, external: true }] : []),
            ...(k.shipTypeId ? [{ label: 'Ship', desc: k.shipName, url: `https://everef.net/type/${k.shipTypeId}`, external: true }] : []),
        ],
    })

    // Jita.Space
    links.push({
        name: 'Jita.Space', icon: '/remotes/jita-space.png',
        items: [
            { label: 'System', desc: k.systemName, url: `https://www.jita.space/system/${k.systemId}`, external: true },
            { label: 'Constellation', desc: k.constellationName, url: `https://www.jita.space/constellation/${k.constellationId}`, external: true, disabled: !k.constellationId },
            { label: 'Region', desc: k.regionName, url: `https://www.jita.space/region/${k.regionId}`, external: true, disabled: !k.regionId },
        ],
    })

    // EVEWho
    links.push({
        name: 'EVEWho', icon: '/remotes/evewho.png',
        items: [
            { label: 'Character', desc: k.characterName, url: `https://evewho.com/character/${k.characterId}`, external: true, disabled: !k.characterId },
            { label: 'Corporation', desc: k.corporationName, url: `https://evewho.com/corporation/${k.corporationId}`, external: true, disabled: !k.corporationId },
            { label: 'Alliance', desc: k.allianceName, url: `https://evewho.com/alliance/${k.allianceId}`, external: true, disabled: !k.allianceId },
        ],
    })

    // zKillboard
    links.push({
        name: 'zKillboard', icon: '/remotes/zkillboard.png',
        items: [
            { label: 'Killmail', desc: `#${k.killmailId}`, url: `https://zkillboard.com/kill/${k.killmailId}/`, external: true },
            { label: 'System', desc: k.systemName, url: `https://zkillboard.com/system/${k.systemId}/`, external: true },
            { label: 'Constellation', desc: k.constellationName, url: `https://zkillboard.com/constellation/${k.constellationId}/`, external: true, disabled: !k.constellationId },
            { label: 'Region', desc: k.regionName, url: `https://zkillboard.com/region/${k.regionId}/`, external: true, disabled: !k.regionId },
            { label: 'Ship Group', desc: k.shipGroupName, url: `https://zkillboard.com/group/${k.shipGroupId}/`, external: true, disabled: !k.shipGroupId },
            { label: 'Ship', desc: k.shipName, url: `https://zkillboard.com/ship/${k.shipTypeId}/`, external: true, disabled: !k.shipTypeId },
            { label: 'Character', desc: k.characterName, url: `https://zkillboard.com/character/${k.characterId}/`, external: true, disabled: !k.characterId },
            { label: 'Corporation', desc: k.corporationName, url: `https://zkillboard.com/corporation/${k.corporationId}/`, external: true, disabled: !k.corporationId },
            { label: 'Alliance', desc: k.allianceName, url: `https://zkillboard.com/alliance/${k.allianceId}/`, external: true, disabled: !k.allianceId },
        ],
    })

    // kb.evetools.org
    links.push({
        name: 'kb.evetools', icon: '/remotes/evetools.png',
        items: [
            { label: 'Killmail', desc: `#${k.killmailId}`, url: `https://kb.evetools.org/kill/${k.killmailId}`, external: true },
            { label: 'Victim Ship', desc: k.shipName, url: `https://kb.evetools.org/ship/${k.shipTypeId}`, external: true, disabled: !k.shipTypeId },
            { label: 'Final Blow Ship', desc: k.finalBlowShipName, url: `https://kb.evetools.org/ship/${k.finalBlowShipTypeId}`, external: true, disabled: !k.finalBlowShipTypeId },
            { label: 'Top Damage Ship', desc: k.topDamageShipName, url: `https://kb.evetools.org/ship/${k.topDamageShipTypeId}`, external: true, disabled: !k.topDamageShipTypeId },
            { label: 'Character', desc: k.characterName, url: `https://kb.evetools.org/character/${k.characterId}`, external: true, disabled: !k.characterId },
            { label: 'Corporation', desc: k.corporationName, url: `https://kb.evetools.org/corporation/${k.corporationId}`, external: true, disabled: !k.corporationId },
            { label: 'Alliance', desc: k.allianceName, url: `https://kb.evetools.org/alliance/${k.allianceId}`, external: true, disabled: !k.allianceId },
            { label: 'System', desc: k.systemName, url: `https://kb.evetools.org/system/${k.systemId}`, external: true, disabled: !k.systemId },
            { label: 'Region', desc: k.regionName, url: `https://kb.evetools.org/region/${k.regionId}`, external: true, disabled: !k.regionId },
        ],
    })

    // SocketKill
    links.push({
        name: 'SocketKill', icon: '/remotes/socketkill.png',
        items: [
            { label: 'Killmail', desc: `#${k.killmailId}`, url: `https://socketkill.com/kill/${k.killmailId}`, external: true },
        ],
    })

    // ESI
    links.push({
        name: 'ESI', icon: '',
        items: [
            { label: 'Raw Killmail', desc: 'JSON from CCP', url: `https://esi.evetech.net/latest/killmails/${k.killmailId}/${k.killmailHash}/`, external: true },
        ],
    })

    return links
        .map(t => ({ ...t, items: t.items.filter(i => !i.disabled) }))
        .filter(t => t.items.length > 0)
})

const openStates = ref<Record<string, boolean>>({})
const siblingsOpen = ref(false)

// Split tool links into two rows of 5 so the external-link clutter stacks
// neatly under itself instead of hogging a single wide line.
const toolLinksRow1 = computed(() => toolLinks.value.slice(0, 5))
const toolLinksRow2 = computed(() => toolLinks.value.slice(5))

// Always offer the Battle Report link — the /battle/<id>?killmail=<id> page
// auto-redirects to a saved battle when one exists and renders an empty-state
// otherwise.
const battleLink = computed(() => `/battle/${props.killmailId}?killmail=${props.killmailId}`)
</script>

<template>
    <!-- ===== DESKTOP navbar =====
         Left column stacks two rows of tool-link dropdowns. Battle Report +
         Related sit in the right column, aligned to the top so they line up
         with row 1. -->
    <div class="hidden md:flex items-start justify-between gap-4 mb-4 py-2 border-b border-white/[0.08]">
        <!-- Left: External tool links in two stacked rows -->
        <div class="flex flex-col gap-0.5 min-w-0">
            <div class="flex flex-wrap items-center gap-0.5">
                <div v-for="tool in toolLinksRow1" :key="tool.name">
                    <Dropdown v-model="openStates[tool.name]">
                        <template #trigger>
                            <button class="flex items-center gap-1 px-1.5 py-1 rounded-md text-sm text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors">
                                <NuxtImg v-if="tool.icon" :src="tool.icon" alt="" width="14" height="14" class="w-3.5 h-3.5 object-contain grayscale opacity-60" />
                                <Icon v-else name="lucide:code" class="text-xs opacity-60" />
                                <span>{{ tool.name }}</span>
                            </button>
                        </template>
                        <template #default="{ close }">
                            <a
                                v-for="item in tool.items"
                                :key="item.url"
                                :href="item.url"
                                :target="item.external ? '_blank' : undefined"
                                rel="noopener noreferrer"
                                class="flex flex-col px-3 py-1.5 rounded-md hover:bg-blue-500/[0.08] transition-colors"
                                @click="close()"
                            >
                                <span class="text-xs text-gray-300">{{ item.label }}</span>
                                <span v-if="item.desc" class="text-fine text-gray-600">{{ item.desc }}</span>
                            </a>
                        </template>
                    </Dropdown>
                </div>
            </div>
            <div class="flex flex-wrap items-center gap-0.5">
                <div v-for="tool in toolLinksRow2" :key="tool.name">
                    <Dropdown v-model="openStates[tool.name]">
                        <template #trigger>
                            <button class="flex items-center gap-1 px-1.5 py-1 rounded-md text-sm text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors">
                                <NuxtImg v-if="tool.icon" :src="tool.icon" alt="" width="14" height="14" class="w-3.5 h-3.5 object-contain grayscale opacity-60" />
                                <Icon v-else name="lucide:code" class="text-xs opacity-60" />
                                <span>{{ tool.name }}</span>
                            </button>
                        </template>
                        <template #default="{ close }">
                            <a
                                v-for="item in tool.items"
                                :key="item.url"
                                :href="item.url"
                                :target="item.external ? '_blank' : undefined"
                                rel="noopener noreferrer"
                                class="flex flex-col px-3 py-1.5 rounded-md hover:bg-blue-500/[0.08] transition-colors"
                                @click="close()"
                            >
                                <span class="text-xs text-gray-300">{{ item.label }}</span>
                                <span v-if="item.desc" class="text-fine text-gray-600">{{ item.desc }}</span>
                            </a>
                        </template>
                    </Dropdown>
                </div>
            </div>
        </div>

        <!-- Right: Battle Report + Sibling Killmails -->
        <div class="flex items-center gap-1 flex-shrink-0">
            <NuxtLink
                v-if="battleLink"
                :to="battleLink"
                class="flex items-center gap-1 px-2 py-1 rounded-md text-sm text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors"
            >
                <Icon name="lucide:swords" class="text-xs" />
                <span>Battle Report</span>
            </NuxtLink>

            <template v-if="siblings.length === 1">
                <NuxtLink
                    :to="`/kill/${siblings[0]!.killmail_id}`"
                    class="flex items-center gap-1.5 px-2 py-1 rounded-md text-sm text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors"
                >
                    <div v-if="siblings[0]!.ship_type_id" class="w-4 h-4 rounded overflow-hidden bg-white/[0.04] flex-shrink-0">
                        <img :src="`/images/types/${siblings[0]!.ship_type_id}/icon?size=64`" :alt="siblings[0]!.ship_name || 'Ship'" class="w-full h-full object-cover">
                    </div>
                    <span>{{ siblings[0]!.ship_name || 'Related Kill' }}</span>
                    <span class="text-xs text-isk/70 tabular-nums">{{ formatIsk(siblings[0]!.total_value) }}</span>
                </NuxtLink>
            </template>

            <template v-else-if="siblings.length > 1">
                <Dropdown v-model="siblingsOpen" align="right">
                    <template #trigger>
                        <button class="flex items-center gap-1 px-2 py-1 rounded-md text-sm text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors">
                            <Icon name="lucide:link" class="text-xs" />
                            <span>Related ({{ siblings.length }})</span>
                        </button>
                    </template>
                    <template #default="{ close }">
                        <NuxtLink
                            v-for="sib in siblings"
                            :key="sib.killmail_id"
                            :to="`/kill/${sib.killmail_id}`"
                            class="flex items-center gap-2 px-3 py-2 rounded-md text-xs text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.08] transition-colors whitespace-nowrap"
                            @click="close()"
                        >
                            <div class="w-5 h-5 rounded overflow-hidden bg-white/[0.04] flex-shrink-0">
                                <img :src="`/images/types/${sib.ship_type_id}/icon?size=64`" :alt="sib.ship_name || 'Ship'" class="w-full h-full object-cover">
                            </div>
                            <span class="flex-1">{{ sib.ship_name || `Kill #${sib.killmail_id}` }}</span>
                            <span class="text-isk/70 tabular-nums">{{ formatIsk(sib.total_value) }}</span>
                        </NuxtLink>
                    </template>
                </Dropdown>
            </template>
        </div>
    </div>

</template>
