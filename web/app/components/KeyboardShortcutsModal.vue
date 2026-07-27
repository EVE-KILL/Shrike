<script setup lang="ts">
const { showHelp } = useKeyboardShortcuts()

const isMac = ref(false)
onMounted(() => {
    isMac.value = navigator.platform.toUpperCase().includes('MAC')
})

const modKey = computed(() => isMac.value ? '⌘' : 'Ctrl')

const sections = computed(() => [
    {
        title: 'General',
        shortcuts: [
            { keys: ['?'], description: 'Show keyboard shortcuts' },
            { keys: [modKey.value, 'K'], description: 'Open search' },
            { keys: ['Esc'], description: 'Close dialog / menu' },
        ],
    },
    {
        title: 'Search',
        shortcuts: [
            { keys: ['↑', '↓'], description: 'Navigate results' },
            { keys: ['↵'], description: 'Select result' },
            { keys: ['Esc'], description: 'Close search' },
        ],
    },
    {
        title: 'Comments',
        shortcuts: [
            { keys: [modKey.value, '↵'], description: 'Submit comment' },
            { keys: [modKey.value, 'G'], description: 'Open GIF picker' },
        ],
    },
])
</script>

<template>
    <Modal v-model="showHelp" title="Keyboard Shortcuts">
        <div class="space-y-5">
            <div v-for="section in sections" :key="section.title">
                <div class="px-2 pb-2 mb-2 text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 border-b border-white/[0.08]">
                    {{ section.title }}
                </div>
                <div class="space-y-1">
                    <div
                        v-for="shortcut in section.shortcuts"
                        :key="shortcut.description"
                        class="flex items-center justify-between px-2 py-1.5 rounded-md hover:bg-blue-500/[0.04]"
                    >
                        <span class="text-sm text-gray-400">{{ shortcut.description }}</span>
                        <div class="flex items-center gap-1">
                            <template v-for="(key, idx) in shortcut.keys" :key="idx">
                                <span v-if="idx > 0" class="text-gray-600 text-fine">+</span>
                                <kbd class="inline-flex items-center justify-center min-w-[1.5rem] h-5 px-1.5 rounded text-xs font-medium text-gray-400 bg-white/[0.06] border border-white/[0.1]">
                                    {{ key }}
                                </kbd>
                            </template>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <template #footer>
            <div class="text-center">
                <span class="text-xs text-gray-600">Press <kbd class="px-1 py-0.5 rounded text-fine bg-white/[0.06] border border-white/[0.1] text-gray-500">?</kbd> to toggle</span>
            </div>
        </template>
    </Modal>
</template>
