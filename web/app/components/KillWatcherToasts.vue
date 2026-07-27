<script setup lang="ts">
const { notifications, dismiss } = useKillWatcher()

// Tick once a second to expire notifications whose dismissAt has passed.
// Mounted globally on every page, so the interval only runs while there
// are live toasts to expire.
let timer: ReturnType<typeof setInterval> | null = null
function stopTimer() {
    if (timer) { clearInterval(timer); timer = null }
}
onMounted(() => {
    watch(() => notifications.value.length, (len) => {
        if (len > 0 && !timer) {
            timer = setInterval(() => {
                const now = Date.now()
                for (const n of notifications.value) {
                    if (n.dismissAt <= now) dismiss(n.uid)
                }
            }, 1000)
        } else if (len === 0) {
            stopTimer()
        }
    }, { immediate: true })
})
onUnmounted(stopTimer)

function open(id: number, uid: number) {
    dismiss(uid)
    navigateTo(`/kill/${id}`)
}

</script>

<template>
    <Teleport to="body">
        <div class="fixed top-16 right-4 z-[10000] flex flex-col gap-2 pointer-events-none w-72 max-w-[calc(100vw-2rem)]">
            <TransitionGroup
                enter-active-class="transition duration-300 ease-out"
                enter-from-class="opacity-0 translate-x-4"
                enter-to-class="opacity-100 translate-x-0"
                leave-active-class="transition duration-200 ease-in"
                leave-from-class="opacity-100"
                leave-to-class="opacity-0 translate-x-4"
            >
                <button
                    v-for="n in notifications"
                    :key="n.uid"
                    class="pointer-events-auto group flex items-center gap-3 p-3 rounded-xl bg-black/80 backdrop-blur-md border border-blue-400/30 shadow-xl shadow-blue-500/10 text-left hover:bg-blue-500/10 transition-colors"
                    @click="open(n.killmail_id, n.uid)"
                >
                    <div v-if="n.ship_type_id" class="w-12 h-12 rounded overflow-hidden bg-black/40 flex-shrink-0">
                        <EveImage :src="`/images/types/${n.ship_type_id}/icon?size=64`" :alt="n.ship_name ?? ''" class="w-full h-full object-cover" loading="eager" />
                    </div>
                    <div class="flex-1 min-w-0">
                        <div class="text-xs font-bold uppercase tracking-wider text-blue-400 mb-0.5">Killmail Ready</div>
                        <div class="text-sm text-white truncate">
                            {{ n.victim_character_name || n.victim_corporation_name || `#${n.killmail_id}` }}
                        </div>
                        <div class="text-fine text-gray-400 truncate">
                            <span v-if="n.ship_name">{{ n.ship_name }}</span>
                            <span v-if="n.ship_name && n.total_value" class="text-gray-600"> · </span>
                            <span v-if="n.total_value" class="text-green-400">{{ formatIsk(n.total_value) }}</span>
                        </div>
                    </div>
                    <Icon name="lucide:external-link" class="text-sm text-gray-500 group-hover:text-blue-400 flex-shrink-0" />
                </button>
            </TransitionGroup>
        </div>
    </Teleport>
</template>
