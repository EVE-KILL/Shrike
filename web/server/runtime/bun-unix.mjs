import '#nitro-internal-pollyfills'
import { useNitroApp } from 'nitropack/runtime'
import { startScheduleRunner } from 'nitropack/runtime/internal'
import wsAdapter from 'crossws/adapters/bun'

const nitroApp = useNitroApp()
const ws = import.meta._websocket
    ? wsAdapter(nitroApp.h3App.websocket)
    : undefined
const socket = process.env.NITRO_UNIX_SOCKET

const listen = socket
    ? { unix: socket }
    : {
        port: process.env.NITRO_PORT || process.env.PORT || 3000,
        host: process.env.NITRO_HOST || process.env.HOST,
    }

const server = Bun.serve({
    ...listen,
    idleTimeout: Number.parseInt(process.env.NITRO_BUN_IDLE_TIMEOUT || '') || undefined,
    websocket: import.meta._websocket ? ws.websocket : undefined,
    async fetch(request, bunServer) {
        if (
            import.meta._websocket
            && request.headers.get('upgrade') === 'websocket'
        ) {
            return ws.handleUpgrade(request, bunServer)
        }

        const url = new URL(request.url)
        const body = request.body ? await request.arrayBuffer() : undefined
        return nitroApp.localFetch(url.pathname + url.search, {
            host: url.hostname,
            protocol: url.protocol,
            headers: request.headers,
            method: request.method,
            redirect: request.redirect,
            body,
        })
    },
})

console.log(
    socket
        ? `Listening on unix socket ${socket}`
        : `Listening on ${server.url}`,
)

if (import.meta._tasks) {
    startScheduleRunner()
}

let stopping = false
async function shutdown() {
    if (stopping) return
    stopping = true
    await server.stop(false)
    await nitroApp.hooks.callHook('close').catch(console.error)
    process.exit(0)
}

process.once('SIGINT', shutdown)
process.once('SIGTERM', shutdown)
