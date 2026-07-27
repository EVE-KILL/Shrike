<script setup lang="ts">
useHead({ title: 'EVE-KILL Wallet' })
useSeoMeta({
    description: 'The public EVE-KILL.com corporation wallet balance and journal.',
    ogTitle: 'EVE-KILL Wallet',
    ogDescription: 'View the public EVE-KILL.com corporation wallet balance and journal.',
})

interface WalletJournalEntry {
    division: number
    journal_id: number
    date: string
    ref_type: string
    description: string
    amount: string | null
    balance: string | null
    first_party_id: number | null
    second_party_id: number | null
    context_id: number | null
    context_id_type: string | null
    reason: string | null
    tax: string | null
    tax_receiver_id: number | null
}

interface WalletData {
    corporation: {
        corporation_id: number
        name: string
        ticker: string
    }
    totalBalance: string
    lastSynced: string | null
    journal: WalletJournalEntry[]
    page: number
    division: number | null
    hasMore: boolean
    pageSize: number
}

const page = ref(1)
const division = ref('')
const query = computed(() => ({
    page: page.value,
    division: division.value || undefined,
}))

const { data, pending, error } = useApiFetch<WalletData>('/api/wallet', { query })

watch(division, () => {
    page.value = 1
})

const formatIsk = (value: string | number | null | undefined) => {
    if (value === null || value === undefined || value === '') return '—'
    const amount = Number(value)
    if (!Number.isFinite(amount)) return '—'
    return `${new Intl.NumberFormat('en-US', {
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
    }).format(amount)} ISK`
}

const formatDate = (value: string | null | undefined) => {
    if (!value) return 'Not yet'
    return new Intl.DateTimeFormat('en-GB', {
        dateStyle: 'medium',
        timeStyle: 'short',
    }).format(new Date(value))
}

const formatReferenceType = (value: string) =>
    value
        .split('_')
        .map(part => part.charAt(0).toUpperCase() + part.slice(1))
        .join(' ')
</script>

<template>
    <div class="max-w-6xl mx-auto py-4 space-y-4">
        <div class="glass-panel p-5 md:p-6">
            <div class="flex flex-col md:flex-row md:items-center justify-between gap-5">
                <div class="flex items-start gap-4">
                    <div class="w-12 h-12 rounded-xl bg-amber-500/15 flex items-center justify-center flex-shrink-0">
                        <Icon name="lucide:wallet-cards" class="text-2xl text-amber-400" />
                    </div>
                    <div>
                        <h1 class="text-2xl md:text-3xl font-bold text-white">
                            {{ data?.corporation.name || 'EVE-KILL.com' }} wallet
                        </h1>
                        <p class="text-sm text-gray-500 mt-1">
                            The corporation wallet used for campaign prizes and other EVE-KILL.com activity.
                            Its balance and journal are public.
                        </p>
                    </div>
                </div>
                <NuxtLink
                    :to="`/corporation/${data?.corporation.corporation_id || 98779905}`"
                    class="inline-flex items-center gap-2 text-xs text-gray-500 hover:text-blue-300 whitespace-nowrap"
                >
                    Corporation {{ data?.corporation.corporation_id || 98779905 }}
                    <Icon name="lucide:arrow-up-right" />
                </NuxtLink>
            </div>
        </div>

        <div v-if="error" class="glass-panel p-5 text-sm text-red-400">
            Could not load the corporation wallet: {{ error.message }}
        </div>

        <template v-if="data">
            <div class="grid sm:grid-cols-2 gap-3">
                <div class="glass-panel p-5">
                    <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500">Total balance</div>
                    <div class="mt-1 text-2xl font-bold text-amber-300 tabular-nums">
                        {{ formatIsk(data.totalBalance) }}
                    </div>
                </div>
                <div class="glass-panel p-5">
                    <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500">Last journal sync</div>
                    <div class="mt-1 text-lg font-semibold text-white">
                        {{ formatDate(data.lastSynced) }}
                    </div>
                    <div class="text-fine text-gray-600 mt-1">Imported automatically from ESI every hour.</div>
                </div>
            </div>

            <div class="glass-panel overflow-hidden">
                <div class="p-4 border-b border-white/[0.06] flex flex-col sm:flex-row sm:items-center gap-3">
                    <div class="flex-1">
                        <h2 class="text-sm font-semibold text-white">Wallet journal</h2>
                        <p class="text-fine text-gray-600 mt-0.5">Latest entries first, exactly as imported from ESI.</p>
                    </div>
                    <select
                        v-model="division"
                        aria-label="Wallet division"
                        class="px-3 py-2 rounded-md bg-gray-900 border border-white/[0.08] text-xs text-gray-300"
                    >
                        <option value="">All divisions</option>
                        <option v-for="number in 7" :key="number" :value="String(number)">
                            Division {{ number }}
                        </option>
                    </select>
                </div>

                <div class="overflow-x-auto">
                    <table class="w-full text-xs">
                        <thead>
                            <tr class="text-left text-gray-600 border-b border-white/[0.06]">
                                <th class="px-4 py-2.5 font-medium whitespace-nowrap">Date</th>
                                <th class="px-4 py-2.5 font-medium">Division</th>
                                <th class="px-4 py-2.5 font-medium">Type</th>
                                <th class="px-4 py-2.5 font-medium min-w-64">Description</th>
                                <th class="px-4 py-2.5 font-medium text-right">Amount</th>
                                <th class="px-4 py-2.5 font-medium text-right">Balance</th>
                                <th class="px-4 py-2.5 font-medium">Parties</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr
                                v-for="entry in data.journal"
                                :key="`${entry.division}-${entry.journal_id}`"
                                class="border-b border-white/[0.04] last:border-0 align-top"
                            >
                                <td class="px-4 py-3 text-gray-400 whitespace-nowrap">{{ formatDate(entry.date) }}</td>
                                <td class="px-4 py-3 text-gray-400">{{ entry.division }}</td>
                                <td class="px-4 py-3 text-gray-300 whitespace-nowrap">{{ formatReferenceType(entry.ref_type) }}</td>
                                <td class="px-4 py-3">
                                    <div class="text-gray-300">{{ entry.description || '—' }}</div>
                                    <div v-if="entry.reason" class="text-fine text-gray-600 mt-1">{{ entry.reason }}</div>
                                </td>
                                <td
                                    class="px-4 py-3 text-right tabular-nums whitespace-nowrap font-medium"
                                    :class="Number(entry.amount) > 0
                                        ? 'text-green-400'
                                        : Number(entry.amount) < 0 ? 'text-red-400' : 'text-gray-500'"
                                >
                                    {{ formatIsk(entry.amount) }}
                                </td>
                                <td class="px-4 py-3 text-right text-gray-400 tabular-nums whitespace-nowrap">
                                    {{ formatIsk(entry.balance) }}
                                </td>
                                <td class="px-4 py-3 text-fine text-gray-600 whitespace-nowrap">
                                    <div v-if="entry.first_party_id">From {{ entry.first_party_id }}</div>
                                    <div v-if="entry.second_party_id">To {{ entry.second_party_id }}</div>
                                    <div v-if="!entry.first_party_id && !entry.second_party_id">—</div>
                                </td>
                            </tr>
                        </tbody>
                    </table>
                </div>

                <div v-if="pending && !data.journal.length" class="p-8 text-center text-sm text-gray-500">
                    Loading wallet journal…
                </div>
                <div v-else-if="!data.journal.length" class="p-8 text-center text-sm text-gray-500">
                    No wallet journal entries have been imported yet.
                </div>
            </div>

            <div v-if="page > 1 || data.hasMore" class="flex items-center justify-center gap-2">
                <button
                    class="glass-panel px-3 py-1.5 text-xs text-gray-400 disabled:opacity-30"
                    :disabled="page <= 1 || pending"
                    @click="page--"
                >
                    Previous
                </button>
                <span class="text-fine text-gray-600">Page {{ page }}</span>
                <button
                    class="glass-panel px-3 py-1.5 text-xs text-gray-400 disabled:opacity-30"
                    :disabled="!data.hasMore || pending"
                    @click="page++"
                >
                    Next
                </button>
            </div>
        </template>

        <div v-else-if="pending" class="glass-panel p-10 text-center text-sm text-gray-500">
            Loading public wallet…
        </div>
    </div>
</template>
