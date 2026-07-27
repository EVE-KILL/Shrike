<script setup lang="ts">
/**
 * Sidebar section nav for the admin and settings shells.
 *
 * Both pages carried their own copy of this — mobile strip and desktop list,
 * ~40 lines each, identical down to the hover colours.
 *
 * Two changes came out of merging them:
 *
 * - Mobile used to render `<span :class="active ? '' : 'hidden'">`, so every
 *   inactive tab collapsed to a bare icon. On /admin that is ten unlabelled
 *   glyphs in a scroller and no way to tell Moderation from Comments without
 *   tapping. Labels now always show.
 * - Sections can carry a badge, so a queue with work waiting in it says so from
 *   the nav instead of only once you open it.
 */
export interface NavSection {
    id: string
    label: string
    icon: string
    /** Count of items needing attention. Hidden when zero or absent. */
    badge?: number
}

defineProps<{
    sections: readonly NavSection[]
    active: string
}>()

const emit = defineEmits<{ select: [id: string] }>()

const itemClass = (isActive: boolean) => isActive
    ? 'bg-blue-500/10 text-blue-400'
    : 'text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.04]'
</script>

<template>
    <nav class="md:w-48 flex-shrink-0">
        <!-- Mobile: horizontal strip -->
        <div class="flex md:hidden overflow-x-auto gap-1 pb-2 -mx-1 px-1 scrollbar-hide">
            <button
                v-for="s in sections" :key="s.id"
                class="flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium whitespace-nowrap transition-colors cursor-pointer"
                :class="itemClass(active === s.id)"
                @click="emit('select', s.id)"
            >
                <Icon :name="s.icon" class="text-base" />
                {{ s.label }}
                <span
                    v-if="s.badge"
                    class="px-1.5 rounded-full bg-yellow-500/20 text-yellow-400 text-fine font-bold tabular-nums"
                >{{ s.badge }}</span>
            </button>
        </div>

        <!-- Desktop: vertical list -->
        <div class="hidden md:flex flex-col gap-0.5">
            <button
                v-for="s in sections" :key="s.id"
                class="flex items-center gap-2.5 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors text-left cursor-pointer"
                :class="itemClass(active === s.id)"
                @click="emit('select', s.id)"
            >
                <Icon :name="s.icon" class="text-base flex-shrink-0" />
                <span class="flex-1 min-w-0 truncate">{{ s.label }}</span>
                <span
                    v-if="s.badge"
                    class="px-1.5 rounded-full bg-yellow-500/20 text-yellow-400 text-fine font-bold tabular-nums flex-shrink-0"
                >{{ s.badge }}</span>
            </button>

            <div v-if="$slots.footer" class="mt-4 pt-4 border-t border-white/[0.06]">
                <slot name="footer" />
            </div>
        </div>
    </nav>
</template>
