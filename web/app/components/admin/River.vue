<script setup lang="ts">
type Depth = Record<'available' | 'running' | 'scheduled' | 'retryable' | 'discarded' | 'cancelled' | 'completed', number>
type RiverQueue = { name: string; depth: Depth; cron: boolean; concurrency?: number; description?: string; paused_at?: string | null; worker_updated_at?: string | null; worker_active?: boolean }
type RiverJob = { id: number; queue: string; kind: string; state: string; attempt: number; max_attempts: number; priority: number; created_at: string; scheduled_at: string; attempted_at?: string | null; finalized_at?: string | null; attempted_by: string[]; args: unknown; errors: Array<{ error?: string; at?: string; attempt?: number }>; output?: unknown; metadata: unknown }

const toast = useToast()
const queueFilter = ref('')
const stateFilter = ref('')
const beforeId = ref<number | undefined>()
const selected = ref<RiverJob | null>(null)
const busy = ref('')

const { data: overview, refresh: refreshOverview } = await useApiFetch<{ queues: RiverQueue[] }>('/api/admin/river', { lazy: true })
const { data: jobs, refresh: refreshJobs } = await useApiFetch<{ jobs: RiverJob[]; next_before_id: number }>('/api/admin/river/jobs', {
    query: { queue: queueFilter, state: stateFilter, before_id: beforeId, limit: 100 },
    lazy: true,
    watch: [queueFilter, stateFilter, beforeId],
})

watch([queueFilter, stateFilter], () => { beforeId.value = undefined })

const stateOptions = ['', 'available', 'running', 'scheduled', 'retryable', 'discarded', 'cancelled', 'completed']
const queueOptions = computed(() => overview.value?.queues ?? [])
const pending = (q: RiverQueue) => q.depth.available + q.depth.running + q.depth.scheduled + q.depth.retryable
const stamp = (value?: string | null) => value ? new Date(value).toLocaleString() : '—'
const json = (value: unknown) => JSON.stringify(value, null, 2)
const stateClass = (state: string) => ({ completed: 'text-green-400', running: 'text-blue-400', discarded: 'text-red-400', retryable: 'text-yellow-400', cancelled: 'text-gray-500', available: 'text-cyan-400', scheduled: 'text-purple-400' }[state] ?? 'text-gray-400')

async function reload() { await Promise.all([refreshOverview(), refreshJobs()]) }

async function queueAction(q: RiverQueue, action: 'pause' | 'resume') {
    busy.value = `${action}:${q.name}`
    try {
        await apiFetch(`/api/admin/river/queues/${encodeURIComponent(q.name)}/action`, { method: 'POST', body: { action } })
        await reload()
    } catch (err: any) { toast.error(extractFetchError(err, `Failed to ${action} queue`)) }
    finally { busy.value = '' }
}

const { pendingId: clearConfirm, confirm: confirmClear } = useConfirmTwice<string>()
async function clearQueue(q: RiverQueue) {
    if (!confirmClear(q.name)) return
    busy.value = `clear:${q.name}`
    try {
        const result = await apiFetch<{ deleted: number }>(`/api/admin/river/queues/${encodeURIComponent(q.name)}/clear`, {
            method: 'POST', body: { states: ['completed', 'cancelled', 'discarded'], limit: 10000 },
        })
        toast.success(`Deleted ${result.deleted} finalized jobs from ${q.name}`)
        await reload()
    } catch (err: any) { toast.error(extractFetchError(err, 'Failed to clear queue')) }
    finally { busy.value = '' }
}

const { pendingId: deleteConfirm, confirm: confirmDelete } = useConfirmTwice()
async function jobAction(job: RiverJob, action: 'cancel' | 'retry' | 'delete') {
    if (action === 'delete' && !confirmDelete(job.id)) return
    busy.value = `${action}:${job.id}`
    try {
        await apiFetch(`/api/admin/river/jobs/${job.id}/action`, { method: 'POST', body: { action } })
        selected.value = null
        await reload()
    } catch (err: any) { toast.error(extractFetchError(err, `Failed to ${action} job`)) }
    finally { busy.value = '' }
}
</script>

<template>
    <div class="space-y-5">
        <div class="flex items-center justify-between">
            <div><h2 class="text-lg font-bold text-white">River Operations</h2><p class="text-xs text-gray-500 mt-1">Queue workers, cron runs, inputs, outcomes, retries, and controls.</p></div>
            <button class="px-3 py-2 rounded-lg bg-white/[0.05] text-xs text-gray-300 hover:text-blue-400 cursor-pointer" @click="reload"><Icon name="lucide:refresh-cw" class="mr-1" /> Refresh</button>
        </div>

        <div class="grid grid-cols-1 xl:grid-cols-2 gap-3">
            <div v-for="q in queueOptions" :key="q.name" class="glass-panel p-4">
                <div class="flex items-start justify-between gap-3">
                    <div class="min-w-0"><div class="flex items-center gap-2"><span class="font-mono text-sm text-white">{{ q.name }}</span><span v-if="q.cron" class="text-fine px-1.5 py-0.5 rounded bg-purple-500/15 text-purple-400">CRON</span><span :class="q.worker_active ? 'text-green-400' : 'text-red-400'" class="text-fine">● {{ q.worker_active ? 'worker live' : 'worker stale' }}</span></div><p class="text-xs text-gray-600 mt-1 truncate">{{ q.description }}</p></div>
                    <div class="flex gap-1">
                        <button class="px-2 py-1 rounded text-xs bg-white/[0.05] text-gray-400 hover:text-yellow-400 cursor-pointer" :disabled="!!busy" @click="queueAction(q, q.paused_at ? 'resume' : 'pause')">{{ q.paused_at ? 'Resume' : 'Pause' }}</button>
                        <button class="px-2 py-1 rounded text-xs bg-red-500/10 text-red-400 cursor-pointer" :disabled="!!busy" @click="clearQueue(q)">{{ clearConfirm === q.name ? 'Confirm clear' : 'Clear history' }}</button>
                    </div>
                </div>
                <div class="grid grid-cols-4 sm:grid-cols-7 gap-2 mt-4 text-center">
                    <div v-for="state in ['available','running','scheduled','retryable','discarded','cancelled','completed'] as const" :key="state"><div class="text-sm font-bold tabular-nums" :class="stateClass(state)">{{ q.depth[state] ?? 0 }}</div><div class="text-fine text-gray-600 uppercase">{{ state }}</div></div>
                </div>
                <div class="mt-3 text-fine text-gray-600">Concurrency {{ q.concurrency ?? '—' }} · Pending {{ pending(q) }} · Heartbeat {{ stamp(q.worker_updated_at) }}<span v-if="q.paused_at" class="text-yellow-400"> · Paused {{ stamp(q.paused_at) }}</span></div>
            </div>
        </div>

        <div class="glass-panel overflow-hidden">
            <div class="p-4 border-b border-white/[0.06] flex flex-wrap gap-2 items-center">
                <h3 class="text-sm font-bold text-white mr-auto">Jobs</h3>
                <select v-model="queueFilter" class="bg-zinc-900 border border-white/10 rounded px-2 py-1.5 text-xs text-gray-300"><option value="">All queues</option><option v-for="q in queueOptions" :key="q.name" :value="q.name">{{ q.name }}</option></select>
                <select v-model="stateFilter" class="bg-zinc-900 border border-white/10 rounded px-2 py-1.5 text-xs text-gray-300"><option v-for="s in stateOptions" :key="s" :value="s">{{ s || 'All states' }}</option></select>
            </div>
            <div class="overflow-x-auto"><table class="w-full text-xs"><thead class="text-gray-600 uppercase text-fine"><tr><th class="text-left p-3">ID</th><th class="text-left p-3">Queue / kind</th><th class="text-left p-3">State</th><th class="text-left p-3">Attempt</th><th class="text-left p-3">Created</th><th class="text-right p-3">Actions</th></tr></thead><tbody class="divide-y divide-white/[0.05]"><tr v-for="job in jobs?.jobs ?? []" :key="job.id" class="hover:bg-white/[0.02]"><td class="p-3 font-mono text-gray-400">{{ job.id }}</td><td class="p-3"><div class="text-white">{{ job.queue }}</div><div class="text-gray-600">{{ job.kind }}</div></td><td class="p-3 font-medium" :class="stateClass(job.state)">{{ job.state }}</td><td class="p-3 text-gray-400">{{ job.attempt }}/{{ job.max_attempts }}</td><td class="p-3 text-gray-500 whitespace-nowrap">{{ stamp(job.created_at) }}</td><td class="p-3"><div class="flex justify-end gap-1"><button class="px-2 py-1 text-blue-400 hover:bg-blue-500/10 rounded cursor-pointer" @click="selected = job">Inspect</button><button v-if="job.state === 'running' || ['available','scheduled','retryable'].includes(job.state)" class="px-2 py-1 text-yellow-400 hover:bg-yellow-500/10 rounded cursor-pointer" @click="jobAction(job, 'cancel')">Cancel</button><button v-else class="px-2 py-1 text-green-400 hover:bg-green-500/10 rounded cursor-pointer" @click="jobAction(job, 'retry')">Rerun</button><button v-if="job.state !== 'running'" class="px-2 py-1 text-red-400 hover:bg-red-500/10 rounded cursor-pointer" @click="jobAction(job, 'delete')">{{ deleteConfirm === job.id ? 'Confirm' : 'Delete' }}</button></div></td></tr></tbody></table></div>
            <div v-if="jobs?.next_before_id" class="p-3 border-t border-white/[0.06] text-center"><button class="text-xs text-blue-400 cursor-pointer" @click="beforeId = jobs!.next_before_id">Older jobs</button></div>
        </div>

        <Teleport to="body"><div v-if="selected" class="fixed inset-0 z-50 bg-black/70 p-4 overflow-y-auto" @click.self="selected = null"><div class="max-w-3xl mx-auto my-8 glass-panel p-5 space-y-4"><div class="flex justify-between"><div><h3 class="font-bold text-white">Job {{ selected.id }}</h3><p class="text-xs text-gray-500">{{ selected.queue }} / {{ selected.kind }}</p></div><button class="text-gray-500 cursor-pointer" @click="selected = null"><Icon name="lucide:x" /></button></div><div class="grid grid-cols-2 md:grid-cols-4 gap-3 text-xs"><div><span class="text-gray-600">State</span><div :class="stateClass(selected.state)">{{ selected.state }}</div></div><div><span class="text-gray-600">Attempt</span><div class="text-white">{{ selected.attempt }}/{{ selected.max_attempts }}</div></div><div><span class="text-gray-600">Scheduled</span><div class="text-white">{{ stamp(selected.scheduled_at) }}</div></div><div><span class="text-gray-600">Finalized</span><div class="text-white">{{ stamp(selected.finalized_at) }}</div></div></div><div><h4 class="text-fine uppercase text-gray-500 mb-1">Input arguments</h4><pre class="p-3 rounded bg-black/30 text-xs text-gray-300 overflow-auto">{{ json(selected.args) }}</pre></div><div><h4 class="text-fine uppercase text-gray-500 mb-1">Output</h4><pre class="p-3 rounded bg-black/30 text-xs text-gray-300 overflow-auto">{{ selected.output == null ? 'No durable output recorded' : json(selected.output) }}</pre></div><div v-if="selected.errors?.length"><h4 class="text-fine uppercase text-red-400 mb-1">Errors</h4><pre class="p-3 rounded bg-red-500/5 text-xs text-red-300 overflow-auto">{{ json(selected.errors) }}</pre></div><div><h4 class="text-fine uppercase text-gray-500 mb-1">Worker IDs</h4><div class="font-mono text-xs text-gray-400 break-all">{{ selected.attempted_by?.join(', ') || '—' }}</div></div></div></div></Teleport>
    </div>
</template>
