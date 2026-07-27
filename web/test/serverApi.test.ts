import { afterAll, describe, expect, test } from 'bun:test'
import { unlink } from 'node:fs/promises'
import { createServerApiClient } from '../shared/utils/serverApi'

const socket = `/tmp/evekill-server-api-${process.pid}.sock`
const server = Bun.serve({
    unix: socket,
    fetch(request) {
        const url = new URL(request.url)
        return Response.json({
            path: url.pathname,
            query: url.search,
            host: request.headers.get('host'),
            cookie: request.headers.get('cookie'),
        })
    },
})

afterAll(async () => {
    await server.stop(true)
    await unlink(socket).catch(() => undefined)
})

describe('SSR API transport', () => {
    test('uses the Go Unix socket while preserving tenant request identity', async () => {
        const api = createServerApiClient({
            socket,
            headers: {
                host: 'boring.eve-kill.com',
                cookie: 'ek_auth=session',
            },
        })

        const response = await api<{
            path: string
            query: string
            host: string
            cookie: string
        }>('/api/killmails', {
            query: { limit: 5 },
        })

        expect(response).toEqual({
            path: '/api/killmails',
            query: '?limit=5',
            host: 'boring.eve-kill.com',
            cookie: 'ek_auth=session',
        })
    })
})
