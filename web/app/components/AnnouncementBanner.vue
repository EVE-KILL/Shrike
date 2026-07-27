<script setup lang="ts">
const { bannerItems, dismiss } = useAnnouncements()

const visibleBanners = computed(() => bannerItems.value.slice(0, 2))

const colorMap: Record<string, { bg: string; border: string; text: string; icon: string }> = {
    info: {
        bg: 'bg-blue-500/10',
        border: 'border-blue-400/30',
        text: 'text-blue-300',
        icon: 'lucide:info',
    },
    warning: {
        bg: 'bg-amber-500/10',
        border: 'border-amber-400/30',
        text: 'text-amber-300',
        icon: 'lucide:alert-triangle',
    },
    danger: {
        bg: 'bg-red-500/10',
        border: 'border-red-400/30',
        text: 'text-red-300',
        icon: 'lucide:alert-circle',
    },
    success: {
        bg: 'bg-green-500/10',
        border: 'border-green-400/30',
        text: 'text-green-300',
        icon: 'lucide:check-circle',
    },
}

function getStyle(color: string) {
    return colorMap[color] ?? colorMap.info!
}
</script>

<template>
    <div v-if="visibleBanners.length" class="space-y-2 mb-4">
        <TransitionGroup
            enter-active-class="transition duration-200 ease-out"
            enter-from-class="opacity-0 -translate-y-2"
            enter-to-class="opacity-100 translate-y-0"
            leave-active-class="transition duration-150 ease-in"
            leave-from-class="opacity-100"
            leave-to-class="opacity-0 -translate-y-2"
        >
            <div
                v-for="item in visibleBanners"
                :key="item.id"
                class="relative rounded-lg border px-4 py-3 text-sm backdrop-blur-sm"
                :class="[getStyle(item.color).bg, getStyle(item.color).border]"
            >
                <div class="flex items-start gap-3">
                    <Icon
                        :name="item.icon || getStyle(item.color).icon"
                        class="text-base flex-shrink-0 mt-0.5"
                        :class="getStyle(item.color).text"
                    />

                    <div class="flex-1 min-w-0">
                        <div class="font-medium" :class="getStyle(item.color).text">
                            {{ item.title }}
                        </div>
                        <div
                            v-if="item.body_html"
                            class="mt-1 text-xs text-gray-400 prose prose-invert prose-xs max-w-none"
                            v-html="item.body_html"
                        />
                        <NuxtLink
                            v-if="item.link_url"
                            :to="item.link_url"
                            class="inline-flex items-center gap-1 mt-2 text-xs font-medium hover:underline"
                            :class="getStyle(item.color).text"
                        >
                            {{ item.link_label || 'Learn more' }}
                            <Icon name="lucide:arrow-right" class="text-[10px]" />
                        </NuxtLink>
                    </div>

                    <button
                        class="flex items-center justify-center w-6 h-6 rounded-md text-gray-500 hover:text-white hover:bg-white/[0.06] transition-colors flex-shrink-0 cursor-pointer"
                        v-tooltip="'Dismiss'"
                        @click="dismiss(item.id)"
                    >
                        <Icon name="lucide:x" class="text-sm" />
                    </button>
                </div>
            </div>
        </TransitionGroup>
    </div>
</template>
