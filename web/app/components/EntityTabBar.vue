<script setup lang="ts">
defineProps<{
    tabs: readonly { id: string; label: string; icon: string }[]
    activeId: string
    accent?: { accent: string } | null
}>()

defineEmits<{ select: [id: string] }>()
</script>

<template>
    <div class="flex overflow-x-auto border-b border-white/[0.08] mb-4 scrollbar-hide">
        <button v-for="tab in tabs" :key="tab.id"
            class="flex items-center gap-2 px-3 md:px-4 py-3 text-sm font-medium transition-colors border-b-2 whitespace-nowrap"
            :class="activeId === tab.id ? 'text-white border-blue-400' : 'text-gray-500 border-transparent hover:text-blue-400'"
            :style="activeId === tab.id && accent ? { borderBottomColor: accent.accent } : undefined"
            @click="$emit('select', tab.id)">
            <Icon :name="tab.icon" class="text-base" />
            <span :class="activeId === tab.id ? '' : 'hidden md:inline'">{{ tab.label }}</span>
        </button>
    </div>
</template>
