<script setup lang="ts">
import type { ApiEndpoint, ApiParam, ApiBodyField } from '~/composables/apiDocs'
import { buildUrl, buildCurl } from '~/composables/apiDocs'

const props = defineProps<{
    endpoint: ApiEndpoint
    baseUrl: string
}>()

const expanded = ref(false)

// Per-param state — initialised lazily on first expand so we don't allocate
// for every card up front (there are ~80 endpoints on the page).
const pathValues = reactive<Record<string, string | number>>({})
const queryValues = reactive<Record<string, string | number>>({})
const bodyValueText = ref('')

const response = ref<{ status: number; timeMs: number; data: unknown; headers?: Record<string, string> } | null>(null)
const error = ref<string | null>(null)
const loading = ref(false)

// Initialise defaults when the card opens for the first time.
const initDefaults = () => {
    for (const p of props.endpoint.pathParams ?? []) {
        if (pathValues[p.name] == null && p.example != null) pathValues[p.name] = p.example
    }
    for (const p of props.endpoint.query ?? []) {
        if (queryValues[p.name] == null && p.default != null) queryValues[p.name] = p.default
    }
    if (props.endpoint.body && !bodyValueText.value) {
        bodyValueText.value = JSON.stringify(
            props.endpoint.body.example ?? buildBodyTemplate(props.endpoint.body.fields),
            null, 2,
        )
    }
}

const buildBodyTemplate = (fields: Record<string, ApiBodyField>): Record<string, unknown> => {
    const out: Record<string, unknown> = {}
    for (const [name, field] of Object.entries(fields)) {
        if (field.default != null) out[name] = field.default
        else if (field.type === 'int[]') out[name] = []
        else if (field.type === 'string[]') out[name] = []
        else if (field.type === 'int') out[name] = 0
        else out[name] = ''
    }
    return out
}

const toggle = () => {
    expanded.value = !expanded.value
    if (expanded.value) initDefaults()
}

// Computed preview URL (updates as inputs change)
const previewUrl = computed(() =>
    buildUrl(props.baseUrl, props.endpoint, pathValues, queryValues),
)

const curlCommand = computed(() => {
    let body: unknown = undefined
    try { body = bodyValueText.value ? JSON.parse(bodyValueText.value) : undefined } catch { /* ignore */ }
    return buildCurl(props.baseUrl, props.endpoint, pathValues, queryValues, body)
})

// Convert a parsed JSON response into a pretty-printed string, capped at a
// sensible length so huge payloads don't blow up the DOM.
const prettyResponse = computed(() => {
    if (!response.value) return ''
    try {
        const json = JSON.stringify(response.value.data, null, 2)
        if (json.length > 200_000) return `${json.slice(0, 200_000)}\n\n… response truncated (${json.length.toLocaleString()} bytes) …`
        return json
    } catch {
        return String(response.value.data)
    }
})

const hasMissingPathParams = computed(() =>
    (props.endpoint.pathParams ?? []).some(p => p.required && (pathValues[p.name] == null || pathValues[p.name] === '')),
)

// Load an example into the form without firing the request.
const loadExample = (example: { path?: string; body?: unknown }) => {
    if (example.path) {
        // Parse /characters/90000001/stats?type=weekly into path + query
        const [pathOnly, qs] = example.path.split('?')
        // Map positional segments of the example back to template names.
        const templateSegments = props.endpoint.path.split('/')
        const exampleSegments = pathOnly?.split('/') ?? []
        templateSegments.forEach((seg, i) => {
            if (seg.startsWith(':')) {
                const name = seg.slice(1)
                const v = exampleSegments[i]
                if (v) pathValues[name] = /^\d+$/.test(v) ? Number(v) : v
            }
        })
        // Query
        for (const k of Object.keys(queryValues)) delete queryValues[k]
        if (qs) {
            const params = new URLSearchParams(qs)
            for (const [k, v] of params) queryValues[k] = v
        }
    }
    if (example.body !== undefined) {
        bodyValueText.value = JSON.stringify(example.body, null, 2)
    }
}

const tryIt = async () => {
    if (hasMissingPathParams.value) {
        error.value = 'Fill in the required path parameters first.'
        return
    }
    error.value = null
    response.value = null
    loading.value = true

    const url = previewUrl.value
    const start = performance.now()

    try {
        const init: RequestInit = { method: props.endpoint.method }
        // Any method that declares a JSON body sends one, not POST alone —
        // the document also carries PUT and PATCH routes.
        if (props.endpoint.body) {
            init.headers = { 'Content-Type': 'application/json' }
            init.body = bodyValueText.value || '{}'
        }
        const res = await fetch(url, init)
        const text = await res.text()
        let data: unknown = text
        try { data = JSON.parse(text) } catch { /* non-JSON response */ }
        response.value = { status: res.status, timeMs: Math.round(performance.now() - start), data }
    } catch (e: any) {
        error.value = e?.message ?? 'Request failed'
    } finally {
        loading.value = false
    }
}

const copyCurl = async () => {
    try { await navigator.clipboard.writeText(curlCommand.value) } catch { /* ignore */ }
}

const copyUrl = async () => {
    try { await navigator.clipboard.writeText(previewUrl.value) } catch { /* ignore */ }
}

// One colour per verb, so a wall of endpoints is scannable by shape. Deletes
// are red because they are the ones worth pausing over before Try it.
const METHOD_CLASSES: Record<string, string> = {
    GET: 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20',
    POST: 'bg-blue-500/10 text-blue-400 border-blue-500/20',
    PUT: 'bg-amber-500/10 text-amber-400 border-amber-500/20',
    PATCH: 'bg-violet-500/10 text-violet-400 border-violet-500/20',
    DELETE: 'bg-red-500/10 text-red-400 border-red-500/20',
}

// Dates get a real date picker rather than free text, so a malformed value
// never reaches Try it.
const inputType = (type: string) =>
    type === 'int' ? 'number' : type === 'date' ? 'date' : 'text'

const methodClass = computed(() =>
    METHOD_CLASSES[props.endpoint.method] ?? METHOD_CLASSES.GET!,
)

// Render path with named params highlighted.
const pathSegments = computed(() => {
    const parts: { type: 'literal' | 'param'; value: string }[] = []
    const regex = /(:[A-Za-z0-9_]+)|([^:]+)/g
    let m: RegExpExecArray | null
    while ((m = regex.exec(props.endpoint.path)) !== null) {
        if (m[1]) parts.push({ type: 'param', value: m[1] })
        else if (m[2]) parts.push({ type: 'literal', value: m[2] })
    }
    return parts
})
</script>

<template>
    <div class="rounded-lg border border-white/[0.08] bg-white/[0.02] hover:bg-white/[0.03] transition-colors">
        <!-- Header (always visible, click to expand) -->
        <button
            class="w-full flex items-start gap-3 text-left p-4"
            @click="toggle"
        >
            <span
                class="font-mono text-[10px] font-semibold px-2 py-0.5 rounded border uppercase flex-shrink-0 mt-0.5"
                :class="methodClass"
            >
                {{ endpoint.method }}
            </span>
            <div class="flex-1 min-w-0">
                <div class="font-mono text-sm text-white truncate">
                    <template v-for="(seg, i) in pathSegments" :key="i">
                        <span v-if="seg.type === 'literal'">{{ seg.value }}</span>
                        <span v-else class="text-blue-400">{{ seg.value }}</span>
                    </template>
                </div>
                <div class="text-xs text-gray-400 mt-0.5">{{ endpoint.summary }}</div>
            </div>
            <Icon
                :name="expanded ? 'lucide:chevron-up' : 'lucide:chevron-down'"
                class="text-gray-500 text-sm flex-shrink-0 mt-1"
            />
        </button>

        <!-- Expanded body -->
        <div v-if="expanded" class="border-t border-white/[0.06] p-4 space-y-4">
            <!-- Description -->
            <p v-if="endpoint.description" class="text-sm text-gray-300 leading-relaxed">
                {{ endpoint.description }}
            </p>

            <!-- Returns -->
            <div v-if="endpoint.returns" class="text-xs">
                <span class="text-gray-500 uppercase tracking-wider mr-2">Returns</span>
                <code class="text-emerald-300 bg-emerald-500/[0.06] border border-emerald-500/[0.15] px-1.5 py-0.5 rounded">{{ endpoint.returns }}</code>
            </div>

            <!-- Notes -->
            <div v-if="endpoint.notes" class="text-xs text-amber-300/90 bg-amber-500/[0.05] border border-amber-500/[0.15] rounded-md px-3 py-2 flex gap-2">
                <Icon name="lucide:info" class="text-[14px] flex-shrink-0 mt-0.5" />
                <span>{{ endpoint.notes }}</span>
            </div>

            <!-- Path params -->
            <div v-if="(endpoint.pathParams ?? []).length > 0">
                <div class="text-[10px] uppercase tracking-wider text-gray-500 mb-2">Path parameters</div>
                <div class="space-y-2">
                    <div v-for="p in endpoint.pathParams" :key="p.name" class="grid grid-cols-[160px_1fr] gap-3 items-start">
                        <div class="text-xs">
                            <code class="text-blue-400">:{{ p.name }}</code>
                            <span v-if="p.required" class="text-red-400 ml-0.5">*</span>
                            <div class="text-[10px] text-gray-600 mt-0.5">{{ p.type }}{{ p.entityType ? ` (${p.entityType})` : '' }}</div>
                        </div>
                        <div>
                            <ApiDocsEntityPicker
                                v-if="p.type === 'entity' && p.entityType"
                                v-model="pathValues[p.name]"
                                :entity-type="p.entityType"
                                :base-url="baseUrl"
                                :placeholder="`Search ${p.entityType}…`"
                            />
                            <input
                                v-else
                                v-model="pathValues[p.name]"
                                :type="inputType(p.type)"
                                :placeholder="p.pattern || p.example?.toString() || p.name"
                                class="w-full bg-white/[0.04] border border-white/[0.10] rounded-md px-2.5 py-1.5 text-xs text-white placeholder-gray-600 outline-none focus:border-blue-500/40"
                            >
                            <div v-if="p.description" class="text-[10px] text-gray-500 mt-1">{{ p.description }}</div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Query params -->
            <div v-if="(endpoint.query ?? []).length > 0">
                <div class="text-[10px] uppercase tracking-wider text-gray-500 mb-2">Query parameters</div>
                <div class="space-y-2">
                    <div v-for="p in endpoint.query" :key="p.name" class="grid grid-cols-[160px_1fr] gap-3 items-start">
                        <div class="text-xs">
                            <code class="text-gray-300">{{ p.name }}</code>
                            <span v-if="p.required" class="text-red-400 ml-0.5">*</span>
                            <div class="text-[10px] text-gray-600 mt-0.5">
                                {{ p.type }}<span v-if="p.default != null"> · default {{ p.default }}</span>
                            </div>
                        </div>
                        <div>
                            <select
                                v-if="p.type === 'enum'"
                                v-model="queryValues[p.name]"
                                class="w-full bg-white/[0.04] border border-white/[0.10] rounded-md px-2.5 py-1.5 text-xs text-white outline-none focus:border-blue-500/40"
                            >
                                <option value="">(not set)</option>
                                <option v-for="v in p.values" :key="v" :value="v">{{ v }}</option>
                            </select>
                            <input
                                v-else
                                v-model="queryValues[p.name]"
                                :type="inputType(p.type)"
                                :placeholder="p.pattern || (p.default != null ? String(p.default) : p.name)"
                                class="w-full bg-white/[0.04] border border-white/[0.10] rounded-md px-2.5 py-1.5 text-xs text-white placeholder-gray-600 outline-none focus:border-blue-500/40"
                            >
                            <div v-if="p.description" class="text-[10px] text-gray-500 mt-1">{{ p.description }}</div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Body -->
            <div v-if="endpoint.body">
                <div class="flex items-center justify-between mb-2">
                    <div class="text-[10px] uppercase tracking-wider text-gray-500">Request body</div>
                    <div class="text-[10px] text-gray-600">{{ endpoint.body.contentType }}</div>
                </div>
                <div class="grid gap-3 mb-2 text-[10px]">
                    <div v-for="(field, name) in endpoint.body.fields" :key="name" class="flex gap-2 items-baseline">
                        <code class="text-gray-300">{{ name }}</code>
                        <span v-if="field.required" class="text-red-400">*</span>
                        <span class="text-gray-600">{{ field.type }}</span>
                        <span v-if="field.maxItems" class="text-gray-600">(max {{ field.maxItems }})</span>
                        <span v-if="field.description" class="text-gray-500">— {{ field.description }}</span>
                    </div>
                </div>
                <textarea
                    v-model="bodyValueText"
                    rows="8"
                    spellcheck="false"
                    class="w-full bg-black/40 border border-white/[0.10] rounded-md px-3 py-2 text-xs font-mono text-white placeholder-gray-600 outline-none focus:border-blue-500/40 resize-y"
                />
            </div>

            <!-- Examples -->
            <div v-if="(endpoint.examples ?? []).length > 0">
                <div class="text-[10px] uppercase tracking-wider text-gray-500 mb-2">Examples</div>
                <div class="flex flex-wrap gap-2">
                    <button
                        v-for="ex in endpoint.examples"
                        :key="ex.label"
                        class="text-[11px] px-2.5 py-1 rounded border border-white/[0.10] bg-white/[0.04] text-gray-300 hover:border-blue-500/40 hover:text-blue-300 transition-colors"
                        @click="loadExample(ex)"
                    >
                        {{ ex.label }}
                    </button>
                </div>
            </div>

            <!-- URL preview + actions -->
            <div>
                <div class="text-[10px] uppercase tracking-wider text-gray-500 mb-2">Request</div>
                <div class="rounded-md bg-black/40 border border-white/[0.10] overflow-hidden">
                    <div class="flex items-center gap-2 px-3 py-2 text-[11px] font-mono text-gray-300 break-all">
                        <span class="text-emerald-400/80">{{ endpoint.method }}</span>
                        <span class="flex-1 min-w-0">{{ previewUrl }}</span>
                    </div>
                </div>
                <div class="flex flex-wrap gap-2 mt-3">
                    <button
                        :disabled="loading"
                        class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-blue-500/20 hover:bg-blue-500/30 text-blue-300 border border-blue-500/30 text-xs font-medium transition-colors disabled:opacity-50"
                        @click="tryIt"
                    >
                        <Icon v-if="loading" name="lucide:loader-2" class="text-[12px] animate-spin" />
                        <Icon v-else name="lucide:play" class="text-[12px]" />
                        Try it
                    </button>
                    <button
                        class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-white/[0.04] hover:bg-white/[0.08] text-gray-300 border border-white/[0.10] text-xs font-medium transition-colors"
                        @click="copyUrl"
                    >
                        <Icon name="lucide:link" class="text-[12px]" />
                        Copy URL
                    </button>
                    <button
                        class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-white/[0.04] hover:bg-white/[0.08] text-gray-300 border border-white/[0.10] text-xs font-medium transition-colors"
                        @click="copyCurl"
                    >
                        <Icon name="lucide:terminal" class="text-[12px]" />
                        Copy curl
                    </button>
                </div>
            </div>

            <!-- Error -->
            <div v-if="error" class="rounded-md bg-red-500/[0.08] border border-red-500/[0.20] px-3 py-2 text-xs text-red-300 flex gap-2">
                <Icon name="lucide:alert-circle" class="text-[14px] flex-shrink-0 mt-0.5" />
                <span>{{ error }}</span>
            </div>

            <!-- Response -->
            <div v-if="response">
                <div class="flex items-center justify-between mb-2">
                    <div class="text-[10px] uppercase tracking-wider text-gray-500">Response</div>
                    <div class="flex items-center gap-3 text-[10px]">
                        <span
                            class="font-mono font-semibold px-1.5 py-0.5 rounded"
                            :class="response.status < 300
                                ? 'bg-emerald-500/10 text-emerald-400'
                                : response.status < 500
                                    ? 'bg-amber-500/10 text-amber-400'
                                    : 'bg-red-500/10 text-red-400'"
                        >
                            {{ response.status }}
                        </span>
                        <span class="text-gray-500">{{ response.timeMs }}ms</span>
                    </div>
                </div>
                <pre class="bg-black/40 border border-white/[0.10] rounded-md px-3 py-2 text-[11px] font-mono text-gray-200 overflow-auto max-h-96 whitespace-pre-wrap break-words">{{ prettyResponse }}</pre>
            </div>
        </div>
    </div>
</template>
