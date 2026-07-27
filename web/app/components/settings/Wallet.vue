<script setup lang="ts">
defineOptions({ name: 'SettingsWallet' })

interface EkWalletTransaction {
    id: number
    type: number
    amount: string
    balance_after: string
    description: string
    external_reference: string
    related_transaction_id: number | null
    campaign_id: string | null
    metadata: Record<string, unknown>
    created_at: string
}

interface EkWalletReservation {
    id: number
    external_reference: string
    transaction_type: number
    amount: string
    description: string
    expires_at: string | null
    created_at: string
}

interface EkWalletData {
    character: {
        character_id: number
        character_name: string
    }
    corporation: {
        corporation_id: number
        name: string
        ticker: string
    }
    balance: string
    totalBalance: string
    reservedBalance: string
    availableBalance: string
    lastSynced: string | null
    depositsEnabledAt: string | null
    transactions: EkWalletTransaction[]
    reservations: EkWalletReservation[]
    page: number
    hasMore: boolean
    pageSize: number
}

const page = ref(1)
const copied = ref(false)
const query = computed(() => ({ page: page.value }))
const { data, pending, error } = useApiFetch<EkWalletData>('/api/user/wallet', { query })

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

const transactionLabel = (type: number) =>
    ['Deposit', 'Purchase', 'Refund', 'Campaign contribution', 'Campaign prize', 'Adjustment'][type]
    ?? 'Transaction'

const copyCorporationName = async () => {
    const name = data.value?.corporation.name
    if (!name) return
    await navigator.clipboard.writeText(name)
    copied.value = true
    window.setTimeout(() => { copied.value = false }, 2000)
}
</script>

<template>
    <div class="space-y-4">
        <div v-if="error" class="glass-panel p-5 text-sm text-red-400">
            Could not load your EK Wallet: {{ error.message }}
        </div>

        <template v-if="data">
            <div class="glass-panel p-5">
                <div class="flex flex-col sm:flex-row sm:items-start justify-between gap-4">
                    <div class="flex items-start gap-3">
                        <div class="w-11 h-11 rounded-xl bg-yellow-500/15 flex items-center justify-center flex-shrink-0">
                            <Icon name="lucide:wallet-cards" class="text-xl text-yellow-400" />
                        </div>
                        <div>
                            <h2 class="text-base font-semibold text-white">Your EK Wallet</h2>
                            <p class="text-xs text-gray-500 mt-0.5">
                                A per-character balance for {{ data.character.character_name }}.
                            </p>
                        </div>
                    </div>
                    <div class="sm:text-right">
                        <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500">Available balance</div>
                        <div class="mt-1 text-2xl font-bold text-yellow-400 tabular-nums">
                            {{ formatIsk(data.availableBalance) }}
                        </div>
                        <div v-if="Number(data.reservedBalance) > 0" class="mt-1 text-fine text-gray-500 tabular-nums">
                            {{ formatIsk(data.reservedBalance) }} held · {{ formatIsk(data.totalBalance) }} total
                        </div>
                    </div>
                </div>
            </div>

            <div v-if="data.reservations.length" class="glass-panel overflow-hidden">
                <div class="p-4 border-b border-white/[0.06]">
                    <div class="flex items-center gap-2">
                        <Icon name="lucide:lock-keyhole" class="text-yellow-400" />
                        <h2 class="text-sm font-semibold text-white">Pending holds</h2>
                    </div>
                    <p class="text-fine text-gray-600 mt-0.5">
                        Reserved funds are excluded from your available balance until captured, released, or expired.
                    </p>
                </div>
                <div class="divide-y divide-white/[0.05]">
                    <div
                        v-for="reservation in data.reservations"
                        :key="reservation.id"
                        class="flex items-start justify-between gap-4 px-4 py-3"
                    >
                        <div class="min-w-0">
                            <div class="text-xs text-gray-300">{{ reservation.description }}</div>
                            <div class="mt-0.5 text-fine text-gray-600 truncate">
                                {{ reservation.external_reference }}
                                <template v-if="reservation.expires_at">
                                    · expires {{ formatDate(reservation.expires_at) }}
                                </template>
                            </div>
                        </div>
                        <div class="text-xs font-medium text-yellow-400 tabular-nums whitespace-nowrap">
                            {{ formatIsk(reservation.amount) }}
                        </div>
                    </div>
                </div>
            </div>

            <div class="glass-panel overflow-hidden">
                <div class="p-5 border-b border-white/[0.06]">
                    <div class="flex items-center gap-2">
                        <Icon name="lucide:arrow-down-to-line" class="text-green-400" />
                        <h2 class="text-sm font-semibold text-white">Deposit ISK</h2>
                    </div>
                    <p class="text-xs text-gray-500 mt-1">
                        Send ISK from the signed-in character to the EVE-KILL.com corporation.
                        ESI identifies the sender, so no special wallet reference is required.
                    </p>
                </div>

                <div class="p-5 space-y-4">
                    <div class="grid sm:grid-cols-2 gap-3">
                        <div class="rounded-lg border border-white/[0.07] bg-white/[0.025] p-4">
                            <div class="text-fine font-bold uppercase tracking-[0.12em] text-gray-600">Send from</div>
                            <div class="flex items-center gap-2 mt-2">
                                <img
                                    :src="`/images/characters/${data.character.character_id}/portrait?size=64`"
                                    :alt="data.character.character_name"
                                    class="w-8 h-8 rounded"
                                >
                                <div>
                                    <div class="text-sm text-white">{{ data.character.character_name }}</div>
                                    <div class="text-fine text-gray-600">{{ data.character.character_id }}</div>
                                </div>
                            </div>
                        </div>

                        <div class="rounded-lg border border-white/[0.07] bg-white/[0.025] p-4">
                            <div class="text-fine font-bold uppercase tracking-[0.12em] text-gray-600">Send to</div>
                            <div class="flex items-center justify-between gap-3 mt-2">
                                <NuxtLink
                                    :to="`/corporation/${data.corporation.corporation_id}`"
                                    class="min-w-0 text-sm text-white hover:text-blue-300"
                                >
                                    <span class="block truncate">{{ data.corporation.name }}</span>
                                    <span class="block text-fine text-gray-600">
                                        {{ data.corporation.ticker }} · {{ data.corporation.corporation_id }}
                                    </span>
                                </NuxtLink>
                                <button
                                    class="px-2.5 py-1.5 rounded bg-white/[0.05] text-fine text-gray-400 hover:text-blue-300 hover:bg-blue-500/10"
                                    @click="copyCorporationName"
                                >
                                    <Icon :name="copied ? 'lucide:check' : 'lucide:copy'" class="mr-1" />
                                    {{ copied ? 'Copied' : 'Copy name' }}
                                </button>
                            </div>
                        </div>
                    </div>

                    <div class="rounded-lg border border-blue-500/15 bg-blue-500/[0.06] px-4 py-3 text-xs text-gray-400 leading-relaxed">
                        Deposits are imported automatically each hour. The last wallet journal sync was
                        <span class="text-gray-300">{{ formatDate(data.lastSynced) }}</span>.
                        A reason beginning with <code class="text-yellow-400">campaign:</code> funds that campaign directly
                        instead of crediting your personal EK Wallet.
                    </div>

                    <div class="rounded-lg border border-yellow-500/20 bg-yellow-500/[0.08] px-4 py-3 text-xs text-yellow-400/80 leading-relaxed">
                        EK Wallet is currently deposit-only while its ledger is being established.
                        The balance cannot be spent, transferred, or withdrawn yet.
                    </div>
                </div>
            </div>

            <div class="glass-panel overflow-hidden">
                <div class="p-4 border-b border-white/[0.06]">
                    <h2 class="text-sm font-semibold text-white">Transaction history</h2>
                    <p class="text-fine text-gray-600 mt-0.5">Immutable entries for this character, newest first.</p>
                </div>

                <div v-if="data.transactions.length" class="overflow-x-auto">
                    <table class="w-full text-xs">
                        <thead>
                            <tr class="text-left text-gray-600 border-b border-white/[0.06]">
                                <th class="px-4 py-2.5 font-medium whitespace-nowrap">Date</th>
                                <th class="px-4 py-2.5 font-medium">Type</th>
                                <th class="px-4 py-2.5 font-medium min-w-52">Description</th>
                                <th class="px-4 py-2.5 font-medium text-right">Amount</th>
                                <th class="px-4 py-2.5 font-medium text-right">Balance</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr
                                v-for="transaction in data.transactions"
                                :key="transaction.id"
                                class="border-b border-white/[0.04] last:border-0"
                            >
                                <td class="px-4 py-3 text-gray-500 whitespace-nowrap">{{ formatDate(transaction.created_at) }}</td>
                                <td class="px-4 py-3 text-gray-300 whitespace-nowrap">{{ transactionLabel(transaction.type) }}</td>
                                <td class="px-4 py-3 text-gray-400">
                                    <NuxtLink
                                        v-if="transaction.campaign_id"
                                        :to="`/campaign/${transaction.campaign_id}`"
                                        class="hover:text-blue-300"
                                    >
                                        {{ transaction.description }}
                                    </NuxtLink>
                                    <template v-else>{{ transaction.description }}</template>
                                </td>
                                <td
                                    class="px-4 py-3 text-right font-medium tabular-nums whitespace-nowrap"
                                    :class="Number(transaction.amount) > 0 ? 'text-green-400' : 'text-red-400'"
                                >
                                    {{ Number(transaction.amount) > 0 ? '+' : '' }}{{ formatIsk(transaction.amount) }}
                                </td>
                                <td class="px-4 py-3 text-right text-gray-400 tabular-nums whitespace-nowrap">
                                    {{ formatIsk(transaction.balance_after) }}
                                </td>
                            </tr>
                        </tbody>
                    </table>
                </div>
                <div v-else class="p-8 text-center text-sm text-gray-500">
                    No EK Wallet transactions yet.
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
            Loading your EK Wallet…
        </div>
    </div>
</template>
