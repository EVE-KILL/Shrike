import type { KillmailsResponse } from '#shared/api'
import { createServerApiClient } from '#shared/utils/serverApi'

const RSS_USER_AGENT = 'eve-kill-nuxt-rss/1.0 (contact: admin@eve-kill.com)'

function escapeXML(value: string): string {
    return value
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&apos;')
}

function formatISK(value: number): string {
    if (value >= 1e9) return `${(value / 1e9).toFixed(2)}B`
    if (value >= 1e6) return `${(value / 1e6).toFixed(2)}M`
    if (value >= 1e3) return `${(value / 1e3).toFixed(2)}K`
    return value.toFixed(0)
}

/**
 * RSS remains a renderer-owned representation, but its data comes exclusively
 * from the canonical Go API. Nitro never connects to Postgres directly.
 */
export default defineEventHandler(async (event) => {
    const config = useRuntimeConfig(event)
    const incoming = getRequestHeaders(event)
    const host = getRequestHost(event, { xForwardedHost: true })
    const headers = new Headers({
        'user-agent': incoming['user-agent'] || RSS_USER_AGENT,
    })
    if (host) {
        headers.set('host', host)
        headers.set('x-forwarded-host', host)
    }
    if (incoming['x-forwarded-proto']) {
        headers.set('x-forwarded-proto', incoming['x-forwarded-proto'])
    }

    const api = createServerApiClient({
        socket: config.apiSocket,
        origin: config.apiOrigin,
        headers,
    })
    const response = await api<KillmailsResponse>(
        '/api/killmails',
        {
            headers,
            query: {
                type: 'latest',
                limit: 50,
                // The canonical endpoint sorts descending when a before cursor
                // is supplied. This cursor remains exact in JavaScript and is
                // comfortably above any EVE killmail ID.
                before: Number.MAX_SAFE_INTEGER,
            },
        },
    )
    const kills = response.data

    const items = kills.map((kill) => {
        const who = kill.victim_character_name
            || kill.victim_corporation_name
            || 'Unknown'
        const ship = kill.ship_name || 'Unknown'
        const system = kill.solar_system_name || 'Unknown'
        const isk = formatISK(kill.total_value)
        const link = `https://eve-kill.com/kill/${kill.killmail_id}`
        const imageURL = kill.ship_type_id
            ? `https://eve-kill.com/images/types/${kill.ship_type_id}/overlayrender?size=128`
            : ''

        return `  <item>
    <title>${escapeXML(`${ship} | ${who} | ${isk} ISK`)}</title>
    <link>${link}</link>
    <guid isPermaLink="true">${link}</guid>
    <pubDate>${new Date(kill.killmail_time).toUTCString()}</pubDate>
    <description>${escapeXML(`${who} lost a ${ship} worth ${isk} ISK in ${system}.`)}</description>${imageURL ? `\n    <enclosure url="${escapeXML(imageURL)}" type="image/png" length="0"/>` : ''}
  </item>`
    })

    const lastBuild = kills[0]
        ? new Date(kills[0].killmail_time).toUTCString()
        : new Date().toUTCString()
    const rss = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
<channel>
  <title>EVE-KILL — Latest Kills</title>
  <link>https://eve-kill.com</link>
  <description>Real-time killmail feed from EVE Online — ship destructions, ISK values, and combat data from New Eden.</description>
  <language>en</language>
  <lastBuildDate>${lastBuild}</lastBuildDate>
  <atom:link href="https://eve-kill.com/feed.xml" rel="self" type="application/rss+xml"/>
  <image>
    <url>https://eve-kill.com/icon.png</url>
    <title>EVE-KILL</title>
    <link>https://eve-kill.com</link>
  </image>
${items.join('\n')}
</channel>
</rss>`

    setResponseHeaders(event, {
        'content-type': 'application/rss+xml; charset=utf-8',
        'cache-control': 'public, s-maxage=60, stale-while-revalidate=120',
    })
    return rss
})
