<script setup lang="ts">
import { eveTools } from '#shared/utils/eveTools'
</script>

<template>
    <footer class="w-full mt-auto pt-8 pb-3 text-sm px-2 md:px-10">
        <!-- Project / legal -->
        <div class="rounded-t-lg border border-white/[0.08] bg-white/[0.04] px-4 py-4">
            <div class="grid grid-cols-1 md:grid-cols-2 items-center gap-3 md:gap-6">
                <div class="flex justify-center md:justify-start">
                    <a
                        href="https://github.com/EVE-KILL/Shrike"
                        target="_blank"
                        rel="noopener noreferrer"
                        class="group inline-flex items-center gap-3 text-gray-300 hover:text-blue-300 transition-colors"
                    >
                        <Icon name="lucide:github" class="text-lg text-gray-200 group-hover:text-blue-300 transition-colors" />
                        <span class="font-medium">Shrike on GitHub</span>
                        <span class="h-4 w-px bg-white/[0.12]"></span>
                        <span class="text-xs font-normal text-gray-500 group-hover:text-gray-400 transition-colors">Open source</span>
                    </a>
                </div>

                <div class="text-center md:text-right text-gray-500 text-xs">
                    EVE Online and all related materials &copy;
                    <a href="https://www.ccpgames.com/" target="_blank" rel="noopener noreferrer" class="text-gray-300 hover:text-blue-300 transition-colors">CCP Games</a>
                </div>
            </div>
        </div>

        <!-- EVE community tools -->
        <div class="flex flex-wrap justify-center rounded-b-lg border-x border-b border-white/[0.08] bg-white/[0.04] px-2 py-4 md:px-3 xl:flex-nowrap">
            <a
                v-for="tool in eveTools"
                :key="tool.name"
                :href="tool.url"
                target="_blank"
                rel="noopener noreferrer"
                v-tooltip="tool.title"
                class="group flex min-w-0 items-center justify-center gap-2 px-3 py-2 text-xs text-gray-400 hover:text-blue-300 transition-colors w-1/2 sm:w-1/3 md:w-1/4 lg:w-1/5 xl:w-auto xl:flex-1 xl:gap-1.5 xl:px-1"
            >
                <NuxtImg
                    v-if="tool.icon"
                    :src="tool.icon"
                    alt=""
                    width="18"
                    height="18"
                    loading="lazy"
                    decoding="async"
                    class="w-[18px] h-[18px] flex-shrink-0 object-contain grayscale opacity-70 group-hover:grayscale-0 group-hover:opacity-100 transition-all duration-300"
                />
                <Icon
                    v-else
                    name="lucide:radio"
                    class="w-[18px] h-[18px] flex-shrink-0 opacity-70 group-hover:opacity-100 transition-opacity"
                />
                <span class="whitespace-nowrap md:text-fine">{{ tool.label || tool.name }}</span>
            </a>
        </div>
    </footer>
</template>
