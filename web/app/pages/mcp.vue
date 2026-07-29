<script setup lang="ts">
interface ToolSchema {
    type: 'object'
    properties: Record<string, any>
    required?: string[]
    additionalProperties?: boolean
}

interface Tool {
    name: string
    description: string
    inputSchema: ToolSchema
    outputSchema?: Record<string, any>
}

interface ToolsListResponse {
    jsonrpc: '2.0'
    id: number
    result?: { tools: Tool[] }
    error?: { code: number; message: string }
}

const config = useRuntimeConfig()
const baseUrl = ref<string>(config.public.publicMcpUrl || 'https://mcp.eve-kill.com')

const tools = ref<Tool[]>([])
const loading = ref(true)
const loadError = ref<string | null>(null)

useHead({ title: 'MCP Server' })
useSeoMeta({
    description: 'EVE-KILL MCP server — AI-friendly tools for searching, analyzing, and narrating EVE Online killmail data over the Model Context Protocol.',
    ogTitle: 'MCP Server — EVE-KILL',
    ogDescription: 'Plug ChatGPT, Claude, or any MCP client into EVE-KILL. Tools covering killmails, characters, corps, intel, battles, war reports, and routing.',
})

const loadTools = async () => {
    loading.value = true
    loadError.value = null
    try {
        const res = await apiFetch<ToolsListResponse>(`${baseUrl.value}/mcp`, {
            method: 'POST',
            headers: { 'content-type': 'application/json' },
            body: { jsonrpc: '2.0', id: 1, method: 'tools/list' },
        })
        if (res.error) throw new Error(res.error.message)
        tools.value = res.result?.tools ?? []
    } catch (e: any) {
        loadError.value = e?.message ?? 'Failed to reach MCP server'
    } finally {
        loading.value = false
    }
}

onMounted(loadTools)

// Group tools by derived category. We don't ship category metadata from
// the server — prefix-matching covers everything except a small manual
// override map for names that don't self-classify.
const CATEGORY_ORDER = ['Discovery', 'Entity', 'Killmails & Battles', 'Fittings & Dogma', 'SDE Lookups', 'Intel (Memgraph)', 'Intel (Derived)', 'Routing', 'Fun & Narrative', 'Me (self-serve)'] as const
type Category = typeof CATEGORY_ORDER[number]

function categorize(name: string): Category {
    if (name.startsWith('me_')) return 'Me (self-serve)'
    if (name === 'search') return 'Discovery'
    if (name === 'route_danger') return 'Routing'
    if (['killmail_fitting', 'dogma_eval', 'fit_compare', 'killmail_forensics', 'doctrine_detect', 'meta_pulse'].includes(name)) return 'Fittings & Dogma'
    if (['ship_info', 'item_info', 'system_info', 'ship_compare'].includes(name)) return 'SDE Lookups'
    if (['killmail', 'battle_report', 'find_battles', 'war_report'].includes(name)) return 'Killmails & Battles'
    if (['flies_with', 'hunts_in', 'hunted_by', 'preys_on', 'compare'].includes(name)) return 'Intel (Memgraph)'
    if (['coalition_graph', 'character_history', 'pilot_efficiency', 'kills_with', 'ships_used'].includes(name)) return 'Intel (Derived)'
    if (['system_pulse', 'global_pulse', 'expensive_losses', 'capsuleer_dossier', 'killmail_story'].includes(name)) return 'Fun & Narrative'
    return 'Entity'
}

const grouped = computed(() => {
    const map = new Map<Category, Tool[]>()
    for (const cat of CATEGORY_ORDER) map.set(cat, [])
    for (const t of tools.value) {
        map.get(categorize(t.name))!.push(t)
    }
    for (const cat of CATEGORY_ORDER) {
        map.get(cat)!.sort((a, b) => a.name.localeCompare(b.name))
    }
    return CATEGORY_ORDER
        .filter(c => map.get(c)!.length > 0)
        .map(c => ({ category: c, tools: map.get(c)! }))
})

const totalTools = computed(() => tools.value.length)

const search = ref('')
const filteredGroups = computed(() => {
    if (!search.value.trim()) return grouped.value
    const q = search.value.toLowerCase()
    return grouped.value
        .map(g => ({
            category: g.category,
            tools: g.tools.filter(t =>
                t.name.toLowerCase().includes(q)
                || t.description.toLowerCase().includes(q),
            ),
        }))
        .filter(g => g.tools.length > 0)
})

const expanded = ref<Record<string, boolean>>({})
const toggle = (name: string) => { expanded.value[name] = !expanded.value[name] }

function paramRow(name: string, schema: any, required: boolean) {
    const type = Array.isArray(schema.type) ? schema.type.join(' | ') : (schema.type ?? 'any')
    const enumList = schema.enum ? ` (${schema.enum.map((v: any) => JSON.stringify(v)).join(' | ')})` : ''
    const def = schema.default !== undefined ? ` default: ${JSON.stringify(schema.default)}` : ''
    return { name, required, type: `${type}${enumList}`, description: schema.description ?? '', default: def }
}

function toolParams(t: Tool) {
    const props = t.inputSchema?.properties ?? {}
    const req = new Set(t.inputSchema?.required ?? [])
    return Object.entries(props).map(([k, v]) => paramRow(k, v, req.has(k)))
}

function formattedOutputSchema(t: Tool) {
    return JSON.stringify(t.outputSchema ?? {}, null, 2)
}

// Native HTTP MCP form — works in Claude Code, ChatGPT connectors, and
// any client that speaks streamable-HTTP directly.
const httpConfig = computed(() => JSON.stringify({
    mcpServers: {
        evekill: {
            type: 'http',
            url: `${baseUrl.value}/mcp`,
        },
    },
}, null, 2))

// Claude Desktop-specific form. Desktop only supports stdio transport
// natively, so remote HTTP servers go through the `mcp-remote` adapter
// (npx package that proxies stdio ↔ streamable-HTTP).
const desktopConfig = computed(() => JSON.stringify({
    mcpServers: {
        evekill: {
            command: 'npx',
            args: ['-y', 'mcp-remote', `${baseUrl.value}/mcp`],
        },
    },
}, null, 2))

const curlExample = computed(() =>
`curl -s -X POST ${baseUrl.value}/mcp \\
  -H 'content-type: application/json' \\
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search","arguments":{"query":"Jita"}}}'`,
)

const categoryAnchor = (c: string) => `cat-${c.replace(/[^a-z0-9]+/gi, '-').toLowerCase()}`
const scrollTo = (c: string) => {
    const el = document.getElementById(categoryAnchor(c))
    if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

const copied = ref<string | null>(null)
async function copy(text: string, key: string) {
    try {
        await navigator.clipboard.writeText(text)
        copied.value = key
        setTimeout(() => { if (copied.value === key) copied.value = null }, 1500)
    } catch {
        // clipboard unavailable — ignore silently
    }
}
</script>

<template>
    <div class="max-w-6xl mx-auto py-4 px-2">
        <!-- Header -->
        <div class="glass-panel backdrop-blur-sm p-6 md:p-8 mb-6">
            <div class="flex items-start justify-between gap-4 flex-wrap">
                <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-3 flex-wrap">
                        <h1 class="text-3xl md:text-4xl font-bold text-white">MCP Server</h1>
                        <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-purple-500/10 text-purple-300 border border-purple-500/20 text-[10px] font-medium uppercase tracking-wide">
                            Model Context Protocol
                        </span>
                    </div>
                    <p class="text-sm text-gray-400 mt-3 max-w-3xl leading-relaxed">
                        Plug any MCP-compatible AI client (Claude, ChatGPT's MCP integrations, custom agents) into
                        EVE-KILL. {{ totalTools || '45' }} purpose-built tools cover killmail lookup, character
                        intel, corp/alliance analytics, battle reports, Memgraph-backed relationship graphs, and
                        route danger assessment. No API key. No auth. JSON-RPC over streamable HTTP.
                    </p>
                </div>
                <div v-if="!loading && !loadError" class="text-xs text-gray-500 text-right">
                    <div>{{ totalTools }} tools</div>
                    <a :href="baseUrl" target="_blank" rel="noopener" class="font-mono text-blue-400/80 hover:text-blue-300 transition-colors">
                        {{ baseUrl }}
                    </a>
                </div>
            </div>
        </div>

        <!-- Connect section -->
        <div class="rounded-lg bg-white/[0.02] border border-white/[0.08] p-6 mb-6 space-y-5">
            <div>
                <div class="flex items-center gap-2 mb-2">
                    <Icon name="lucide:plug" class="text-purple-400" />
                    <h2 class="text-lg font-semibold text-white">Connect an AI client</h2>
                </div>
                <p class="text-xs text-gray-500 max-w-3xl">
                    MCP clients that speak <span class="font-mono">streamable-HTTP</span> can add EVE-KILL as a
                    server. Endpoint: <span class="font-mono text-blue-300">{{ baseUrl }}/mcp</span> (POST JSON-RPC,
                    optional SSE on GET). Rate limit: 20 req/s per IP.
                </p>
            </div>

            <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
                <div>
                    <div class="flex items-center justify-between mb-1.5 flex-wrap gap-1">
                        <div class="text-[11px] uppercase tracking-wider text-gray-500">
                            Claude Code / ChatGPT / native HTTP clients
                        </div>
                        <button
                            class="text-[10px] px-2 py-0.5 rounded bg-white/[0.04] hover:bg-white/[0.08] text-gray-400 hover:text-white border border-white/[0.08] transition-colors"
                            @click="copy(httpConfig, 'http')"
                        >
                            {{ copied === 'http' ? 'Copied!' : 'Copy' }}
                        </button>
                    </div>
                    <pre class="text-[11px] font-mono bg-black/40 border border-white/[0.06] rounded-md p-3 overflow-x-auto whitespace-pre text-gray-300">{{ httpConfig }}</pre>
                    <div class="text-[10px] text-gray-500 mt-1.5 leading-relaxed">
                        Or from the CLI:
                        <span class="font-mono text-blue-300">claude mcp add --scope user --transport http evekill {{ baseUrl }}/mcp</span>
                    </div>
                </div>

                <div>
                    <div class="flex items-center justify-between mb-1.5 flex-wrap gap-1">
                        <div class="text-[11px] uppercase tracking-wider text-gray-500">
                            Claude Desktop (via mcp-remote)
                        </div>
                        <button
                            class="text-[10px] px-2 py-0.5 rounded bg-white/[0.04] hover:bg-white/[0.08] text-gray-400 hover:text-white border border-white/[0.08] transition-colors"
                            @click="copy(desktopConfig, 'desktop')"
                        >
                            {{ copied === 'desktop' ? 'Copied!' : 'Copy' }}
                        </button>
                    </div>
                    <pre class="text-[11px] font-mono bg-black/40 border border-white/[0.06] rounded-md p-3 overflow-x-auto whitespace-pre text-gray-300">{{ desktopConfig }}</pre>
                    <div class="text-[10px] text-gray-500 mt-1.5 leading-relaxed">
                        Desktop only supports stdio transport — the
                        <span class="font-mono">mcp-remote</span> adapter proxies to HTTP.
                        Edit <span class="font-mono">~/Library/Application Support/Claude/claude_desktop_config.json</span>
                        (macOS) or
                        <span class="font-mono">%APPDATA%\Claude\claude_desktop_config.json</span> (Windows),
                        then fully quit + relaunch.
                    </div>
                </div>
            </div>

            <div>
                <div class="flex items-center justify-between mb-1.5">
                    <div class="text-[11px] uppercase tracking-wider text-gray-500">Raw cURL (sanity-check the server)</div>
                    <button
                        class="text-[10px] px-2 py-0.5 rounded bg-white/[0.04] hover:bg-white/[0.08] text-gray-400 hover:text-white border border-white/[0.08] transition-colors"
                        @click="copy(curlExample, 'curl')"
                    >
                        {{ copied === 'curl' ? 'Copied!' : 'Copy' }}
                    </button>
                </div>
                <pre class="text-[11px] font-mono bg-black/40 border border-white/[0.06] rounded-md p-3 overflow-x-auto whitespace-pre text-gray-300">{{ curlExample }}</pre>
            </div>

            <div class="text-[11px] text-gray-500 border-t border-white/[0.05] pt-3 flex items-center justify-between flex-wrap gap-2">
                <span>
                    Building an agent? Point it at
                    <a href="/mcp.txt" class="text-blue-400 hover:text-blue-300 font-mono">/mcp.txt</a>
                    — a machine-readable usage guide tailored for AI models.
                </span>
                <a href="https://modelcontextprotocol.io" target="_blank" rel="noopener" class="text-gray-500 hover:text-gray-300 inline-flex items-center gap-1">
                    <Icon name="lucide:external-link" class="text-[10px]" />
                    MCP spec
                </a>
            </div>
        </div>

        <!-- Loading -->
        <div v-if="loading" class="rounded-lg border border-white/[0.08] bg-white/[0.02] p-8 text-center">
            <Icon name="lucide:loader-2" class="text-2xl text-purple-400 animate-spin" />
            <div class="text-sm text-gray-500 mt-3">Fetching tool list from {{ baseUrl }}…</div>
        </div>

        <!-- Error -->
        <div v-else-if="loadError" class="rounded-lg border border-red-500/[0.20] bg-red-500/[0.05] p-6">
            <div class="flex gap-3 items-start">
                <Icon name="lucide:alert-triangle" class="text-red-400 text-xl flex-shrink-0 mt-0.5" />
                <div class="flex-1">
                    <div class="text-sm text-red-300 font-medium">Failed to load tool list</div>
                    <div class="text-xs text-red-300/70 mt-1">{{ loadError }}</div>
                    <button
                        class="mt-3 text-xs px-3 py-1.5 rounded bg-red-500/20 hover:bg-red-500/30 text-red-200 border border-red-500/30 transition-colors"
                        @click="loadTools"
                    >
                        Retry
                    </button>
                </div>
            </div>
        </div>

        <!-- Loaded -->
        <div v-else-if="tools.length > 0" class="grid grid-cols-1 lg:grid-cols-[220px_1fr] gap-6">
            <!-- Sidebar -->
            <aside class="lg:sticky lg:top-4 lg:self-start space-y-4">
                <div class="relative">
                    <Icon name="lucide:search" class="absolute left-2.5 top-1/2 -translate-y-1/2 text-xs text-gray-500 pointer-events-none" />
                    <input
                        v-model="search"
                        type="text"
                        placeholder="Filter tools…"
                        class="w-full bg-white/[0.04] border border-white/[0.10] rounded-md pl-7 pr-2.5 py-1.5 text-xs text-white placeholder-gray-600 outline-none focus:border-purple-500/40"
                    >
                </div>
                <nav class="space-y-1">
                    <div class="text-[10px] uppercase tracking-wider text-gray-600 px-2 mb-1">
                        {{ totalTools }} tools
                    </div>
                    <button
                        v-for="g in filteredGroups"
                        :key="g.category"
                        class="w-full flex items-center justify-between gap-2 px-2 py-1.5 rounded text-xs text-gray-400 hover:text-purple-300 hover:bg-purple-500/[0.06] transition-colors text-left"
                        @click="scrollTo(g.category)"
                    >
                        <span class="truncate">{{ g.category }}</span>
                        <span class="text-[10px] text-gray-600 font-mono flex-shrink-0">{{ g.tools.length }}</span>
                    </button>
                    <div v-if="filteredGroups.length === 0" class="text-xs text-gray-600 px-2 py-3 text-center">
                        No tools match
                    </div>
                </nav>
            </aside>

            <!-- Main -->
            <main class="space-y-8 min-w-0">
                <section
                    v-for="g in filteredGroups"
                    :id="categoryAnchor(g.category)"
                    :key="g.category"
                    class="scroll-mt-4"
                >
                    <div class="mb-3">
                        <h2 class="text-xl font-semibold text-white">{{ g.category }}</h2>
                    </div>
                    <div class="space-y-2">
                        <div
                            v-for="t in g.tools"
                            :key="t.name"
                            class="rounded-lg border border-white/[0.08] bg-white/[0.02] hover:bg-white/[0.03] transition-colors overflow-hidden"
                        >
                            <button
                                class="w-full flex items-start justify-between gap-3 p-3 text-left"
                                @click="toggle(t.name)"
                            >
                                <div class="flex-1 min-w-0">
                                    <div class="flex items-center gap-2 flex-wrap">
                                        <span class="font-mono text-sm text-purple-300">{{ t.name }}</span>
                                        <span
                                            v-if="(t.inputSchema?.required?.length ?? 0) > 0"
                                            class="text-[10px] text-gray-500"
                                        >
                                            requires {{ t.inputSchema.required!.join(', ') }}
                                        </span>
                                    </div>
                                    <p class="text-xs text-gray-400 mt-1 leading-relaxed">{{ t.description }}</p>
                                </div>
                                <Icon
                                    :name="expanded[t.name] ? 'lucide:chevron-up' : 'lucide:chevron-down'"
                                    class="text-gray-500 flex-shrink-0 mt-0.5"
                                />
                            </button>

                            <div v-if="expanded[t.name]" class="border-t border-white/[0.05] bg-black/20">
                                <div v-if="toolParams(t).length > 0" class="p-3 space-y-2">
                                    <div class="text-[10px] uppercase tracking-wider text-gray-600 mb-1">Arguments</div>
                                    <div
                                        v-for="p in toolParams(t)"
                                        :key="p.name"
                                        class="grid grid-cols-[140px_1fr] gap-3 items-start text-xs"
                                    >
                                        <div class="font-mono flex items-center gap-1.5">
                                            <span class="text-purple-300">{{ p.name }}</span>
                                            <span v-if="p.required" class="text-red-400/70 text-[9px]">required</span>
                                        </div>
                                        <div class="text-gray-400">
                                            <div class="font-mono text-[10px] text-gray-500">{{ p.type }}{{ p.default }}</div>
                                            <div v-if="p.description" class="mt-0.5">{{ p.description }}</div>
                                        </div>
                                    </div>
                                </div>
                                <div v-else class="p-3 text-xs text-gray-600 italic">No arguments.</div>
                                <div v-if="t.outputSchema" class="border-t border-white/[0.05] p-3">
                                    <div class="text-[10px] uppercase tracking-wider text-gray-600 mb-2">
                                        Output schema
                                    </div>
                                    <pre class="max-h-72 overflow-auto rounded bg-black/30 p-3 text-[10px] leading-relaxed text-gray-400">{{ formattedOutputSchema(t) }}</pre>
                                </div>
                            </div>
                        </div>
                    </div>
                </section>
            </main>
        </div>
    </div>
</template>
