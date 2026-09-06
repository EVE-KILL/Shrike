import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { ref } from 'vue'

// Exercise the actual client-only composable with a deterministic browser
// transport. Nuxt normally replaces import.meta.client/server at build time.
const source = readFileSync(new URL('../app/composables/useRelaySocket.ts', import.meta.url), 'utf8')
const compiled = new Bun.Transpiler({ loader: 'ts', define: {
    'import.meta.client': 'true', 'import.meta.server': 'false',
} }).transformSync(source)

test('visibility-paused feeds close, AIID-style feeds stay open, both dispose', async () => {
    const globals = globalThis as any
    const saved = Object.fromEntries(['ref', 'window', 'document', 'WebSocket'].map(key => [key, Object.getOwnPropertyDescriptor(globals, key)]))
    const handlers = new Set<() => void>()
    class Socket {
        static instances: Socket[] = []
        readyState = 1
        closed = false
        constructor(public url: string) { Socket.instances.push(this) }
        close() { this.closed = true; this.readyState = 3 }
    }
    const document = {
        hidden: false,
        addEventListener: (_: string, handler: () => void) => handlers.add(handler),
        removeEventListener: (_: string, handler: () => void) => handlers.delete(handler),
    }
    try {
        Object.assign(globals, { ref, window: { location: { origin: 'https://example.test' } }, document, WebSocket: Socket })
        const { createRelaySocket } = await import(`data:text/javascript;base64,${Buffer.from(compiled).toString('base64')}`)
        const options = { wsUrl: 'https://example.test', path: '/killlist', onMessage: () => {} }
        const ordinary = createRelaySocket({ ...options, visibilityPause: true })
        const background = createRelaySocket({ ...options, visibilityPause: false })
        ordinary.connect(); background.connect()
        document.hidden = true
        for (const handler of handlers) handler()
        expect(Socket.instances[0]!.closed).toBe(true)
        expect(Socket.instances[1]!.closed).toBe(false)
        ordinary.connect()
        expect(Socket.instances.length).toBe(2)
        document.hidden = false
        for (const handler of handlers) handler()
        expect(Socket.instances.length).toBe(3)
        ordinary.dispose(); background.dispose()
        expect(Socket.instances.every(socket => socket.closed)).toBe(true)
        expect(handlers.size).toBe(0)
    } finally {
        for (const [key, descriptor] of Object.entries(saved)) {
            if (descriptor) Object.defineProperty(globals, key, descriptor)
            else delete globals[key]
        }
    }
})
