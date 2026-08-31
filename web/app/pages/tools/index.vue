<script setup lang="ts">
useSeoMeta({
    title: 'Tools',
    description: 'EVE Online utilities on EVE-KILL — Local Scan analyzer, D-Scan parser, Battle Generator, Campaign Creator, Advanced Search, Ship Fittings, Legacy Kill archive.',
})

interface Tool {
    label: string
    href: string
    icon: string
    description: string
}

const tools: Tool[] = [
    {
        label: 'Killmail Labels',
        href: '/labels',
        icon: 'lucide:tags',
        description: 'Browse killmail classifications by space, engagement, value, hull class, and technology.',
    },
    {
        label: 'Advanced Search',
        href: '/advancedsearch',
        icon: 'lucide:search',
        description: 'Filter killmails across characters, corps, alliances, systems, ships, and time ranges.',
    },
    {
        label: 'Battle Generator',
        href: '/battlegenerator',
        icon: 'lucide:shield',
        description: 'Paste killmail IDs to build a shareable battle report on the fly.',
    },
    {
        label: 'Campaign Creator',
        href: '/campaigncreator',
        icon: 'lucide:flag',
        description: 'Bundle killmails by attackers, victims, systems or timeframe into a named campaign.',
    },
    {
        label: 'Comments',
        href: '/comments',
        icon: 'lucide:message-square',
        description: 'Recent community comments across killmails, battles, and entities.',
    },
    {
        label: 'Local Scan',
        href: '/tools/localscan',
        icon: 'lucide:scan-search',
        description: 'Paste an in-game Local list to instantly break it down by corp, alliance, and ship composition.',
    },
    {
        label: 'D-Scan',
        href: '/tools/dscan',
        icon: 'lucide:radar',
        description: 'Parse a directional scan export into a clean ship-by-ship summary.',
    },
    {
        label: 'Ship Fittings',
        href: '/fits',
        icon: 'lucide:wrench',
        description: 'Browse and search community-submitted ship fittings with modules and stats.',
    },
    {
        label: 'Legacy Kills',
        href: '/legacy',
        icon: 'lucide:archive',
        description: 'Pre-API killmail archive from 2003–2007 with search and top-loss lists.',
    },
]
</script>

<template>
    <div class="max-w-5xl mx-auto px-4 py-8">
        <header class="mb-8">
            <h1 class="text-3xl font-bold text-white mb-2">Tools</h1>
            <p class="text-gray-400 text-sm">
                Utilities for EVE Online pilots — scan analyzers, battle tooling, and search helpers.
            </p>
        </header>

        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            <NuxtLink
                v-for="tool in tools"
                :key="tool.href"
                :to="tool.href"
                class="group flex flex-col gap-2 rounded-lg border border-white/[0.08] bg-white/[0.02] p-4 hover:border-blue-500/30 hover:bg-blue-500/[0.04] transition-colors"
            >
                <div class="flex items-center gap-2">
                    <Icon :name="tool.icon" class="text-xl text-blue-400/70 group-hover:text-blue-400" />
                    <h2 class="text-base font-semibold text-white">{{ tool.label }}</h2>
                </div>
                <p class="text-sm text-gray-400 group-hover:text-gray-300">{{ tool.description }}</p>
            </NuxtLink>
        </div>
    </div>
</template>
