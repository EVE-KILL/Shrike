<script setup lang="ts">
const { toasts, remove } = useToast()
</script>

<template>
    <Teleport to="body">
        <div class="fixed bottom-4 left-1/2 -translate-x-1/2 z-[10000] flex flex-col gap-2 pointer-events-none">
            <TransitionGroup
                enter-active-class="transition duration-200 ease-out"
                enter-from-class="opacity-0 translate-y-2"
                enter-to-class="opacity-100 translate-y-0"
                leave-active-class="transition duration-150 ease-in"
                leave-from-class="opacity-100"
                leave-to-class="opacity-0"
            >
                <div
                    v-for="t in toasts"
                    :key="t.id"
                    class="pointer-events-auto flex items-center gap-2 px-4 py-2.5 rounded-lg border backdrop-blur-md text-sm shadow-xl"
                    :class="t.type === 'success' ? 'bg-green-500/10 border-green-400/30 text-green-300'
                          : t.type === 'error'   ? 'bg-red-500/10 border-red-400/30 text-red-300'
                          :                        'bg-blue-500/10 border-blue-400/30 text-blue-300'"
                    @click="remove(t.id)"
                >
                    <Icon
                        :name="t.type === 'success' ? 'lucide:check-circle' : t.type === 'error' ? 'lucide:alert-circle' : 'lucide:info'"
                        class="text-base flex-shrink-0"
                    />
                    <span>{{ t.message }}</span>
                </div>
            </TransitionGroup>
        </div>
    </Teleport>
</template>
