<script setup lang="ts">
useHead({ title: 'Status' })
useSeoMeta({
    description: 'EVE-KILL system status — service health, queue processing, and backend connectivity.',
    ogTitle: 'System Status — EVE-KILL',
    robots: 'noindex, nofollow',
})

const config = useRuntimeConfig()
const data = ref<any>(null)
const connected = ref(false)

let ws: WebSocket | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let reconnectAttempts = 0
let hiddenByVisibility = false

const connectWs = () => {
    const wsUrl = config.public.wsUrl
    if (!wsUrl) return

    const base = new URL(wsUrl, window.location.origin)
    base.protocol = base.protocol === 'https:' ? 'wss:' : 'ws:'
    base.pathname = `${base.pathname.replace(/\/$/, '')}/status`
    ws = new WebSocket(base)

    ws.onopen = () => {
        connected.value = true
        reconnectAttempts = 0
    }

    ws.onmessage = (event) => {
        try {
            const msg = JSON.parse(event.data)
            if (msg.channel === 'status') {
                data.value = msg.data
            }
        } catch {}
    }

    ws.onclose = () => {
        connected.value = false
        ws = null
        if (!hiddenByVisibility) {
            const delay = Math.min(1000 * 2 ** reconnectAttempts, 30000)
            reconnectAttempts++
            reconnectTimer = setTimeout(connectWs, delay)
        }
    }

    ws.onerror = () => {}
}

const disconnectWs = () => {
    if (reconnectTimer) {
        clearTimeout(reconnectTimer)
        reconnectTimer = null
    }
    ws?.close()
    ws = null
    connected.value = false
}

const onVisibilityChange = () => {
    if (document.hidden) {
        hiddenByVisibility = true
        disconnectWs()
    } else {
        hiddenByVisibility = false
        connectWs()
    }
}

onMounted(() => {
    connectWs()
    document.addEventListener('visibilitychange', onVisibilityChange)
})

onUnmounted(() => {
    document.removeEventListener('visibilitychange', onVisibilityChange)
    hiddenByVisibility = false
    disconnectWs()
})
</script>

<template>
    <InfoPage
        title="System Status"
        subtitle="Queues, workers, databases and the killmail pipeline, live."
        icon="lucide:activity"
        wide
    >
        <template #actions>
            <div class="flex items-center gap-2">
                <div
                    class="w-2 h-2 rounded-full flex-shrink-0"
                    :class="connected ? 'bg-green-500' : 'bg-red-500'"
                    v-tooltip="connected ? 'Live — connected' : 'Disconnected'"
                ></div>
                <span class="text-xs text-gray-500">{{ connected ? 'Live' : 'Connecting...' }}</span>
            </div>
        </template>

        <div v-if="!data" class="flex items-center justify-center py-20">
            <Icon name="lucide:loader-2" class="w-6 h-6 text-gray-500 animate-spin" />
        </div>

        <div v-else class="space-y-4">
            <StatusSummary :data="data" />
            <StatusSystem :data="data" />
            <StatusCloudflare :data="data.cloudflare" />
            <StatusZkbIngest :data="data.zkb_ingest" />
            <StatusEsiOverview :coverage="data.coverage" :tokens="data.esi_tokens" />
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                <StatusQueues :data="data" />
                <StatusPostgres :data="data" />
            </div>
            <StatusImageServer :data="data.image_server" />
            <StatusWebSocketServer :data="data.websocket" />
            <StatusDatabaseTables :data="data" />
        </div>

        <div v-if="!data && !connected" class="text-center text-gray-500 py-20">
            Unable to connect to backend
        </div>
    </InfoPage>
</template>
