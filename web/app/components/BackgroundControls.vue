<script setup lang="ts">
const { viewerMode, toggleViewer, setRedditBackground } = useSiteBackground()
const loading = ref(false)

const shuffle = async () => {
    loading.value = true
    await setRedditBackground()
    loading.value = false
}
</script>

<template>
    <div class="fixed bottom-4 left-4 z-[9999] flex items-end gap-2">
        <!-- Eye button — toggle viewer mode -->
        <button
            class="w-9 h-9 rounded-lg flex items-center justify-center transition-all"
            :class="viewerMode ? 'bg-blue-500/30 text-blue-400' : 'bg-black/50 text-gray-400 hover:text-blue-400 hover:bg-black/70'"
            @click="toggleViewer"
            v-tooltip="'Toggle background view'"
        >
            <Icon :name="viewerMode ? 'lucide:eye-off' : 'lucide:eye'" class="w-4 h-4" />
        </button>

        <!-- Shuffle button — random Reddit -->
        <button
            class="w-9 h-9 rounded-lg flex items-center justify-center bg-black/50 text-gray-400 hover:text-blue-400 hover:bg-black/70 transition-all"
            @click="shuffle"
            :disabled="loading"
            v-tooltip="'Random background from r/eveporn'"
        >
            <Icon name="lucide:shuffle" class="w-4 h-4" :class="{ 'animate-spin': loading }" />
        </button>
    </div>
</template>
