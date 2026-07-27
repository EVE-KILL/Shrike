<script setup lang="ts">
const eveTools = [
    { name: 'DOTLAN', url: 'https://evemaps.dotlan.net/', icon: '/remotes/dotlan.png', title: 'EVE Maps & Intel' },
    { name: 'EVEEye', url: 'https://eveeye.com/', icon: '/remotes/eveeye.svg', title: 'Interactive Maps' },
    { name: 'Missioneer', url: 'https://evemissioneer.com/', icon: '/remotes/evemissioneer.png', title: 'Missions & PvE' },
    { name: 'EveShip.fit', url: 'https://eveship.fit/', icon: '/remotes/eveship-fit.png', title: 'Ship Fitting Tool' },
    { name: 'EVERef', url: 'https://everef.net/', icon: '/remotes/everef.png', title: 'Game Reference' },
    { name: 'Jita.Space', url: 'https://jita.space/', icon: '/remotes/jita-space.png', title: 'Market Analysis' },
    { name: 'EVEWho', url: 'https://evewho.com/', icon: '/remotes/evewho.png', title: 'Character Intel' },
    { name: 'zKillboard', url: 'https://zkillboard.com/', icon: '/remotes/zkillboard.png', title: 'Killmail Database' },
    { name: 'Socket.Kill', url: 'https://socketkill.com/', icon: '/remotes/socketkill.png', title: 'Live Killmail Stream' },
    { name: 'RIFT Intel Fusion', url: 'https://riftforeve.online/', icon: '/images/rift-intel-fusion-tool-256.png', title: 'Intel Fusion Tool' },
    { name: 'Eve Monthly', url: 'https://www.evemonthly.com/', icon: '/remotes/evemonthly.svg', title: 'Coalition Intelligence' },
    { name: 'EVE LKM', url: 'https://eve-lkm.capsuleer.life/', icon: '/remotes/eve-lkm.png', title: 'Last Known Mail' },
    { name: 'Evetools.org', url: 'https://br.evetools.org/', icon: '/remotes/evetools.png', title: 'Battle Reports' },
]
</script>

<template>
    <footer class="w-full pt-6 pb-2 mt-auto text-sm px-2 md:px-10">
        <!-- Main footer row -->
        <div class="border-t border-white/[0.08] pt-5 pb-4">
            <div class="flex flex-col sm:flex-row items-center justify-between gap-3">
                <!-- Left: GitHub -->
                <div class="flex-1">
                    <a
                        href="https://github.com/EVE-KILL"
                        target="_blank"
                        rel="noopener noreferrer"
                        class="inline-flex items-center gap-1.5 text-gray-500 hover:text-blue-400 transition-colors"
                    >
                        <Icon name="lucide:github" class="text-base" />
                        <span>GitHub</span>
                    </a>
                </div>

                <!-- Center: Creator -->
                <div class="flex-1 text-center text-gray-600">
                    Created by
                    <NuxtLink to="/character/268946627" class="text-blue-400/80 hover:text-blue-400 transition-colors">
                        Karbowiak
                    </NuxtLink>
                </div>

                <!-- Right: Legal -->
                <div class="flex-1 text-right text-gray-600 text-xs">
                    All materials &copy;
                    <a href="https://www.ccpgames.com/" target="_blank" rel="noopener noreferrer" class="text-blue-400/80 hover:text-blue-400 transition-colors">
                        CCP Games
                    </a>
                </div>
            </div>
        </div>

        <!-- EVE Community Tools -->
        <div class="border-t border-white/[0.04] pt-4 pb-2">
            <div class="flex flex-wrap justify-center w-full">
                <a
                    v-for="tool in eveTools"
                    :key="tool.name"
                    :href="tool.url"
                    target="_blank"
                    rel="noopener noreferrer"
                    v-tooltip="tool.title"
                    class="group flex flex-col items-center gap-1.5 py-3 px-2 rounded-lg text-xs text-gray-600 opacity-70 md:opacity-40 hover:opacity-100 hover:text-blue-400 hover:bg-blue-500/[0.04] transition-all duration-300 w-1/4 sm:w-1/6 md:w-auto md:flex-1"
                >
                    <NuxtImg
                        v-if="tool.icon"
                        :src="tool.icon"
                        alt=""
                        width="20"
                        height="20"
                        class="w-5 h-5 object-contain grayscale group-hover:grayscale-0 transition-all duration-300"
                    />
                    <Icon
                        v-else
                        name="lucide:radio"
                        class="w-5 h-5 transition-all duration-300"
                    />
                    <span>{{ tool.name }}</span>
                </a>
            </div>
        </div>
    </footer>
</template>
