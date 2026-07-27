<script setup lang="ts">
import type { DomainConfig } from '~/composables/useDomainConfig'

const props = defineProps<{
    config: DomainConfig
    /**
     * Render the header without a banner image: no <img>, no dark scrim, and
     * a taller top pad so the site background shows through the strip the way
     * an image normally would. The content panel's own top fade (see
     * .content--bleed) is what stops it looking like a hole.
     */
    transparent?: boolean
}>()

const showLogo = computed(() => props.config.theme.showLogoInBanner !== false && props.config.theme.logoUrl)
const showName = computed(() => props.config.theme.showNameInBanner !== false && props.config.siteName)
const showDescription = computed(() => props.config.theme.showDescriptionInBanner !== false && props.config.siteDescription)
const showOverlay = computed(() => showLogo.value || showName.value || showDescription.value)
</script>

<template>
    <!-- Transparent: height comes from the padding, since there is no image to
         size to, and the top pad has to clear the fixed 4rem navbar. -->
    <div
        class="relative w-full overflow-hidden rounded-t-lg"
        :class="transparent ? 'pt-20 pb-4' : 'h-48 md:h-64'"
    >
        <img
            v-if="!transparent"
            :src="config.theme.bannerUrl"
            :alt="config.siteName || 'Banner'"
            class="w-full h-full object-cover"
        >
        <template v-if="showOverlay">
            <!-- The scrim exists to lift text off a busy banner image. With no
                 image the background is already dimmed by .content-blur, and a
                 second scrim would just grey out the bleed-through. -->
            <div v-if="!transparent" class="absolute inset-0 bg-gradient-to-t from-black/80 via-black/20 to-transparent"></div>
            <!-- Transparent mode drops the vertical padding: the strip's own
                 pt/pb set its height, and doubling up made it far taller than
                 the header needs. -->
            <div
                class="flex items-end gap-4 max-w-[80rem] mx-auto"
                :class="transparent
                    ? 'relative px-4 md:px-6'
                    : 'absolute bottom-0 left-0 right-0 p-4 md:p-6'"
            >
                <img
                    v-if="showLogo"
                    :src="config.theme.logoUrl"
                    :alt="config.siteName || 'Logo'"
                    class="w-12 h-12 md:w-16 md:h-16 rounded-lg object-cover border border-white/20"
                    :class="transparent ? 'shadow-lg shadow-black/50' : ''"
                >
                <!-- Drop shadows do the scrim's job locally: whatever the
                     visitor's background image is, the text stays readable
                     without dimming the whole strip. -->
                <div v-if="showName || showDescription" :class="transparent ? '[text-shadow:0_2px_8px_rgb(0_0_0_/_0.9)]' : ''">
                    <h1 v-if="showName" class="text-xl md:text-2xl font-bold text-white">{{ config.siteName }}</h1>
                    <p v-if="showDescription" class="text-sm text-gray-300 mt-1 line-clamp-2">{{ config.siteDescription }}</p>
                </div>
            </div>
        </template>
    </div>
</template>
