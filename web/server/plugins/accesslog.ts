import { isbot } from 'isbot'

function formatDate(d: Date): string {
    const pad = (n: number) => String(n).padStart(2, '0')
    const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']
    const offset = -d.getTimezoneOffset()
    const sign = offset >= 0 ? '+' : '-'
    const absOff = Math.abs(offset)
    const tz = `${sign}${pad(Math.floor(absOff / 60))}${pad(absOff % 60)}`
    return `${pad(d.getDate())}/${months[d.getMonth()]}/${d.getFullYear()}:${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())} ${tz}`
}

export default defineNitroPlugin((nitroApp) => {
    nitroApp.hooks.hook('request', (event) => {
        event.context.requestStartTime = Date.now()
    })

    nitroApp.hooks.hook('beforeResponse', (event) => {
        // Skip internal SSR fetches ($fetch / useRequestFetch during SSR).
        // These go through the h3 app in-process and have no real socket.
        // In dev we still want to see them, so only skip in production.
        if (!import.meta.dev && !event.node.req.socket?.remoteAddress) return

        const method = event.node.req.method || 'GET'
        const url = event.node.req.url || ''
        const status = event.node.res.statusCode
        const ip = getRequestIP(event, { xForwardedFor: true }) || '-'
        const ua = event.node.req.headers['user-agent'] || '-'
        const referer = event.node.req.headers.referer || '-'
        const duration = event.context.requestStartTime
            ? (Date.now() - event.context.requestStartTime)
            : 0
        const size = event.node.res.getHeader('content-length') || '-'
        const bot = isbot(ua) ? ' [BOT]' : ''
        const ts = formatDate(new Date())

        console.log(`${ip} - - [${ts}] "${method} ${url} HTTP/1.1" ${status} ${size} "${referer}" "${ua}" ${duration}ms${bot}`)
    })
})
