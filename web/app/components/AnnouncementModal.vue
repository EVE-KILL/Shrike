<script setup lang="ts">
const { modalItems, dismiss } = useAnnouncements()

// Queue: show one at a time
const currentIndex = ref(0)
const current = computed(() => modalItems.value[currentIndex.value] ?? null)
const isOpen = computed(() => !!current.value)

async function acknowledge() {
    if (!current.value) return
    await dismiss(current.value.id)
    // After dismissal reactivity removes it from modalItems,
    // currentIndex stays at 0, next item (if any) slides in.
    currentIndex.value = 0
}

const colorMap: Record<string, { accent: string; button: string }> = {
    info: { accent: 'text-blue-400', button: 'bg-blue-600 hover:bg-blue-500' },
    warning: { accent: 'text-amber-400', button: 'bg-amber-600 hover:bg-amber-500' },
    danger: { accent: 'text-red-400', button: 'bg-red-600 hover:bg-red-500' },
    success: { accent: 'text-green-400', button: 'bg-green-600 hover:bg-green-500' },
}

function getColor(color: string) {
    return colorMap[color] || colorMap.info
}
</script>

<template>
    <Teleport to="body">
        <Transition
            enter-active-class="transition duration-200 ease-out"
            enter-from-class="opacity-0"
            enter-to-class="opacity-100"
            leave-active-class="transition duration-150 ease-in"
            leave-from-class="opacity-100"
            leave-to-class="opacity-0"
        >
            <div
                v-if="isOpen && current"
                class="fixed inset-0 z-[10001] flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm"
            >
                <div class="w-full max-w-md rounded-xl border border-white/[0.10] bg-[#141414] shadow-2xl shadow-black/60 overflow-hidden">
                    <!-- Header -->
                    <div class="flex items-center gap-3 px-5 py-4 border-b border-white/[0.06]">
                        <Icon
                            name="lucide:alert-circle"
                            class="text-lg flex-shrink-0"
                            :class="getColor(current.color).accent"
                        />
                        <h2 class="text-sm font-semibold text-white">{{ current.title }}</h2>
                    </div>

                    <!-- Body -->
                    <div class="px-5 py-4 max-h-[60vh] overflow-y-auto">
                        <div
                            v-if="current.body_html"
                            class="text-sm text-gray-300 prose prose-invert prose-sm max-w-none"
                            v-html="current.body_html"
                        />
                        <p v-else class="text-sm text-gray-400">
                            {{ current.title }}
                        </p>

                        <NuxtLink
                            v-if="current.link_url"
                            :to="current.link_url"
                            class="inline-flex items-center gap-1 mt-3 text-sm font-medium hover:underline"
                            :class="getColor(current.color).accent"
                        >
                            {{ current.link_label || 'Learn more' }}
                            <Icon name="lucide:external-link" class="text-xs" />
                        </NuxtLink>
                    </div>

                    <!-- Footer -->
                    <div class="px-5 py-3 border-t border-white/[0.06] flex items-center justify-between">
                        <span v-if="modalItems.length > 1" class="text-xs text-gray-500">
                            {{ modalItems.length }} alert{{ modalItems.length > 1 ? 's' : '' }} remaining
                        </span>
                        <span v-else />
                        <button
                            class="px-4 py-2 rounded-lg text-sm font-medium text-white transition-colors cursor-pointer"
                            :class="getColor(current.color).button"
                            @click="acknowledge"
                        >
                            I Acknowledge
                        </button>
                    </div>
                </div>
            </div>
        </Transition>
    </Teleport>
</template>
