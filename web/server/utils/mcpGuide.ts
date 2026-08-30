/**
 * Shared generator for the plain-text MCP usage guide. Served at both
 * /mcp.txt (thematic) and /llms.txt (answer.ai convention for AI-readable
 * site docs). Generated live from the MCP server's own tools/list so new
 * tools appear automatically without a frontend deploy.
 */

interface Tool {
    name: string
    description: string
    inputSchema: {
        type: 'object'
        properties?: Record<string, any>
        required?: string[]
    }
    outputSchema?: Record<string, any>
}

interface ToolsListResponse {
    jsonrpc: '2.0'
    id: number
    result?: { tools: Tool[] }
    error?: { code: number; message: string }
}

const TOOL_ORDER_HINT = [
    'search',
    'entity_overview', 'entity_kills', 'entity_top', 'entity_timeline',
    'killmail', 'killmail_story',
    'flies_with', 'hunts_in', 'hunted_by', 'preys_on', 'compare', 'kills_with', 'ships_used',
    'battle_report', 'find_battles',
    'system_pulse', 'expensive_losses', 'capsuleer_dossier', 'route_danger',
    'me_dossier', 'me_overview', 'me_kills', 'me_flies_with',
    'me_hunts_in', 'me_hunted_by', 'me_preys_on', 'me_kills_with', 'me_ships_used', 'me_timeline',
]

function toolRank(name: string): number {
    const idx = TOOL_ORDER_HINT.indexOf(name)
    return idx === -1 ? TOOL_ORDER_HINT.length + name.charCodeAt(0) : idx
}

function formatParam(name: string, schema: any, required: boolean): string {
    const type = Array.isArray(schema.type) ? schema.type.join(' | ') : (schema.type ?? 'any')
    const enumList = schema.enum ? ` (one of: ${schema.enum.map((v: any) => JSON.stringify(v)).join(', ')})` : ''
    const def = schema.default !== undefined ? `, default ${JSON.stringify(schema.default)}` : ''
    const reqLabel = required ? '(required)' : '(optional)'
    const desc = schema.description ? ` — ${schema.description}` : ''
    return `  - ${name} ${reqLabel} [${type}${enumList}${def}]${desc}`
}

function renderTool(t: Tool): string {
    const req = new Set(t.inputSchema?.required ?? [])
    const props = t.inputSchema?.properties ?? {}
    const params = Object.entries(props).map(([k, v]) => formatParam(k, v, req.has(k)))
    const paramBlock = params.length > 0 ? `\nArguments:\n${params.join('\n')}` : '\nArguments: none'
    const outputType = t.outputSchema?.title || t.outputSchema?.type || (t.outputSchema ? 'typed object' : 'unspecified')
    return `### ${t.name}\n${t.description}${paramBlock}\nReturns: ${outputType} (see tools/list outputSchema or /api/docs for the full schema)`
}

export async function buildMcpGuide(mcpBase: string): Promise<string> {
    let tools: Tool[] = []
    let fetchError: string | null = null

    try {
        const res = await $fetch<ToolsListResponse>(`${mcpBase}/mcp`, {
            method: 'POST',
            headers: { 'content-type': 'application/json' },
            body: { jsonrpc: '2.0', id: 1, method: 'tools/list' },
            timeout: 5000,
        })
        if (res.error) throw new Error(res.error.message)
        tools = res.result?.tools ?? []
    } catch (e: any) {
        fetchError = e?.message ?? 'unknown error'
    }

    tools.sort((a, b) => toolRank(a.name) - toolRank(b.name))

    const lines: string[] = []
    lines.push('# EVE-KILL MCP Server')
    lines.push('')
    lines.push(`- [EVE-KILL](https://eve-kill.com/) — human-friendly killboard`)
    lines.push(`- [REST API documentation](https://eve-kill.com/docs) — OpenAPI reference`)
    lines.push(`- [MCP endpoint](${mcpBase}/mcp) — JSON-RPC over streamable HTTP`)
    lines.push('')
    lines.push(`Endpoint: ${mcpBase}/mcp  (JSON-RPC over streamable HTTP)`)
    lines.push(`Protocol: MCP 2025-06-18`)
    lines.push(`Rate limit: 20 requests/second per IP (no auth, no API key)`)
    lines.push('')
    lines.push('## What this server does')
    lines.push('')
    lines.push('Exposes EVE-KILL killmail and intel data to AI agents as a set of narrow, purpose-built tools.')
    lines.push('Covers: entity lookup, killmail detail, character/corp/alliance stats, relationship graphs,')
    lines.push('battle reports, system activity heat, route danger, and personalized (self-service) queries.')
    lines.push('')
    lines.push('Back-end datasources: Postgres (killmails, breakdowns, stats), Memgraph (FLEW_WITH, KILLED,')
    lines.push('OPERATED_IN edges over the last 90 days), solar-system graph (stargate BFS/Dijkstra).')
    lines.push('')
    lines.push('## Connecting a client')
    lines.push('')
    lines.push('There are two shapes depending on what speaks streamable-HTTP natively.')
    lines.push('')
    lines.push('### Claude Code, ChatGPT connectors, and other native HTTP clients')
    lines.push('')
    lines.push('    {')
    lines.push('      "mcpServers": {')
    lines.push('        "evekill": {')
    lines.push('          "type": "http",')
    lines.push(`          "url": "${mcpBase}/mcp"`)
    lines.push('        }')
    lines.push('      }')
    lines.push('    }')
    lines.push('')
    lines.push(`Or via CLI: \`claude mcp add --scope user --transport http evekill ${mcpBase}/mcp\``)
    lines.push('')
    lines.push('### Claude Desktop — requires the `mcp-remote` adapter')
    lines.push('')
    lines.push('Claude Desktop only supports stdio transport natively, so remote HTTP servers go through')
    lines.push('the `mcp-remote` package (proxies stdio <-> streamable-HTTP). Add to')
    lines.push('`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS (or')
    lines.push('`%APPDATA%\\Claude\\claude_desktop_config.json` on Windows), then fully quit + relaunch:')
    lines.push('')
    lines.push('    {')
    lines.push('      "mcpServers": {')
    lines.push('        "evekill": {')
    lines.push('          "command": "npx",')
    lines.push(`          "args": ["-y", "mcp-remote", "${mcpBase}/mcp"]`)
    lines.push('        }')
    lines.push('      }')
    lines.push('    }')
    lines.push('')
    lines.push('Editing via the Desktop UI (Settings -> Developer -> Edit Config) works too, but the file')
    lines.push('is the source of truth.')
    lines.push('')
    lines.push('## How to call it')
    lines.push('')
    lines.push('Standard MCP JSON-RPC. Example:')
    lines.push('')
    lines.push('    POST /mcp  {"jsonrpc":"2.0","id":1,"method":"initialize",')
    lines.push('                "params":{"protocolVersion":"2025-06-18","capabilities":{},')
    lines.push('                          "clientInfo":{"name":"my-agent","version":"0"}}}')
    lines.push('')
    lines.push('    POST /mcp  {"jsonrpc":"2.0","id":2,"method":"tools/list"}')
    lines.push('')
    lines.push('    POST /mcp  {"jsonrpc":"2.0","id":3,"method":"tools/call",')
    lines.push('                "params":{"name":"search","arguments":{"query":"Jita"}}}')
    lines.push('')
    lines.push('All responses are JSON. Tool results include both MCP `structuredContent` and a')
    lines.push('JSON text block for compatibility. Every tool advertises inputSchema and outputSchema.')
    lines.push('')
    lines.push('## Usage tips for LLMs')
    lines.push('')
    lines.push('1. When the user mentions any named entity (character, corp, alliance, ship, system, region),')
    lines.push('   call `search` first to resolve it. Pass the `type` hint when you know it; otherwise the')
    lines.push('   server picks the best candidate by fuzzy match + type rank.')
    lines.push('2. If the user self-identifies ("I\'m Karbowiak", "my stats are..."), prefer the `me_*` tools')
    lines.push('   and pass the character name via the `me` argument. Stateless — pass `me` every call.')
    lines.push('3. For a rich one-shot character profile, call `capsuleer_dossier` / `me_dossier` — it combines')
    lines.push('   lifetime stats, archetype tags, playstyle, top ships, top systems, and wingmates in one go.')
    lines.push('4. For "who is hunting me" or "who do I fly with" questions, use the Memgraph-backed intel tools')
    lines.push('   (`flies_with`, `hunted_by`, `preys_on`). The graph window is 90 days.')
    lines.push('   For precise pair-counts like "how many kills did X and I share?" or "how many ships did I')
    lines.push('   kill with Y flying a Deluge?", use `kills_with` / `me_kills_with` — it counts shared')
    lines.push('   killmails with optional ship/victim/location/time filters and supports breakdowns.')
    lines.push('   For per-ship questions like "how many kills did I get in a Deluge?" or "what ships have I')
    lines.push('   killed/lost in over the last 30 days?", use `ships_used` / `me_ships_used` — char/corp/')
    lines.push('   alliance scoped, supports custom time windows, single-ship focus, and kills/losses combined.')
    lines.push('5. For "what\'s happening in system X right now" use `system_pulse`. For recent large fights use')
    lines.push('   `find_battles`; drill into one via `battle_report`.')
    lines.push('6. `route_danger` supports `prefer: "shortest"` (fewest jumps) or `prefer: "safest"` (EVE')
    lines.push('   autopilot-style, stays in highsec when possible). Stargate-only — no wormholes.')
    lines.push('7. Every entity-shaped response includes a canonical eve-kill.com URL under `url` — use them')
    lines.push('   when citing sources to the user.')
    lines.push('')
    lines.push('## Tools')
    lines.push('')

    if (fetchError) {
        lines.push(`(Live tool list unavailable: ${fetchError}. Try POST ${mcpBase}/mcp directly.)`)
    } else if (tools.length === 0) {
        lines.push('(No tools registered.)')
    } else {
        for (const t of tools) {
            lines.push(renderTool(t))
            lines.push('')
        }
    }

    lines.push('## Notes and limits')
    lines.push('')
    lines.push('- Stats tables are refreshed continuously from the killmail ingest pipeline. Daily granularity')
    lines.push('  retains 365 days; monthly + yearly retained forever.')
    lines.push('- Memgraph edges (FLEW_WITH, KILLED, OPERATED_IN) are rebuilt on a rolling 90-day window —')
    lines.push('  older associations fall out.')
    lines.push('- ISK values reflect EVE-KILL\'s custom price overlay when set, otherwise current Jita prices.')
    lines.push('- No authentication. No per-user state. No writes. Pure read-only over public killboard data.')
    lines.push('- Human-friendly site: https://eve-kill.com  · REST API: https://eve-kill.com/api')
    lines.push('')

    return lines.join('\n')
}
