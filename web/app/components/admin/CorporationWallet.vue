<script setup lang="ts">
defineOptions({ name: "AdminCorporationWallet" });

interface WalletAuthorization {
	authorized_character_id: number;
	authorized_character_name: string;
	authorized_by_admin_character_id: number;
	token_expiry: string;
	scopes: string[];
	disabled: boolean;
	last_balance_sync: string | null;
	last_journal_sync: string | null;
	last_error: string | null;
	created_at: string;
	updated_at: string;
}

interface WalletBalance {
	division: number;
	balance: string;
	updated_at: string;
}

interface WalletJournalEntry {
	division: number;
	journal_id: number;
	date: string;
	ref_type: string;
	description: string;
	amount: string | null;
	balance: string | null;
	first_party_id: number | null;
	second_party_id: number | null;
	context_id: number | null;
	context_id_type: string | null;
	reason: string | null;
	tax: string | null;
	tax_receiver_id: number | null;
}

interface PrizeSettlementRow {
	campaign_id: string;
	campaign_name: string;
	pool_status: number;
	funded_total: string;
	finalized_at: string | null;
	rank: number | null;
	character_id: number | null;
	character_name: string | null;
	metric_value: string | null;
	payout_percentage: number | null;
	payout_amount: string | null;
	claimed_at: string | null;
	paid_at: string | null;
	payment_note: string | null;
}

interface WalletReference {
	division: number;
	journal_id: number;
	reference_type: string;
	reference_id: string;
	status: number;
	amount: string | null;
	note: string | null;
	created_at: string;
	date: string;
	first_party_id: number | null;
	reason: string | null;
}

interface WalletData {
	corporation: { corporation_id: number; name: string; ticker: string };
	authorization: WalletAuthorization | null;
	requiredScopes: string[];
	balances: WalletBalance[];
	totalBalance: string;
	journal: WalletJournalEntry[];
	page: number;
	division: number | null;
	hasMore: boolean;
	pageSize: number;
	prizeSettlements: PrizeSettlementRow[];
	walletReferences: WalletReference[];
}

const route = useRoute();
const page = ref(1);
const division = ref("");
const authorizing = ref(false);
const syncing = ref(false);
const markingPaid = ref("");
const actionMessage = ref("");
const actionError = ref("");

const query = computed(() => ({
	page: page.value,
	division: division.value || undefined,
}));
const { data, pending, error, refresh } = useApiFetch<WalletData>(
	"/api/admin/wallet",
	{ query },
);

watch(division, () => {
	page.value = 1;
});

const authorizationBanner = computed(() => {
	if (route.query.wallet_auth === "ok") {
		const message =
			typeof route.query.message === "string"
				? route.query.message
				: "Wallet authorization saved. The first journal import has been queued.";
		return {
			kind: "success",
			text: message,
		};
	}
	if (route.query.wallet_auth === "error") {
		const message =
			typeof route.query.message === "string"
				? route.query.message
				: "Wallet authorization failed.";
		return { kind: "error", text: message };
	}
	return null;
});

const divisions = computed(() => {
	const byDivision = new Map(
		(data.value?.balances ?? []).map((wallet) => [wallet.division, wallet]),
	);
	return Array.from({ length: 7 }, (_, index) => {
		const divisionNumber = index + 1;
		return (
			byDivision.get(divisionNumber) ?? {
				division: divisionNumber,
				balance: "0.00",
				updated_at: "",
			}
		);
	});
});

const prizeSettlementGroups = computed(() => {
	const groups = new Map<
		string,
		{
			campaign_id: string;
			campaign_name: string;
			pool_status: number;
			funded_total: string;
			finalized_at: string | null;
			results: PrizeSettlementRow[];
		}
	>();
	for (const row of data.value?.prizeSettlements ?? []) {
		const group = groups.get(row.campaign_id) ?? {
			campaign_id: row.campaign_id,
			campaign_name: row.campaign_name,
			pool_status: row.pool_status,
			funded_total: row.funded_total,
			finalized_at: row.finalized_at,
			results: [],
		};
		if (row.rank != null) group.results.push(row);
		groups.set(row.campaign_id, group);
	}
	return [...groups.values()];
});

const formatIsk = (value: string | number | null | undefined) => {
	if (value === null || value === undefined || value === "") return "—";
	const amount = Number(value);
	if (!Number.isFinite(amount)) return "—";
	return `${new Intl.NumberFormat("en-US", {
		minimumFractionDigits: 2,
		maximumFractionDigits: 2,
	}).format(amount)} ISK`;
};

const formatDate = (value: string | null | undefined) => {
	if (!value) return "Never";
	return new Intl.DateTimeFormat("en-GB", {
		dateStyle: "medium",
		timeStyle: "short",
	}).format(new Date(value));
};

const formatReferenceType = (value: string) =>
	value
		.split("_")
		.map((part) => part.charAt(0).toUpperCase() + part.slice(1))
		.join(" ");

const authorize = async () => {
	authorizing.value = true;
	actionError.value = "";
	try {
		const result = await apiFetch<{ url: string }>("/api/admin/wallet/authorize");
		window.location.assign(result.url);
	} catch (requestError: any) {
		actionError.value =
			requestError?.data?.message ||
			requestError?.message ||
			"Could not start wallet authorization";
		authorizing.value = false;
	}
};

const syncNow = async () => {
	syncing.value = true;
	actionError.value = "";
	actionMessage.value = "";
	try {
		await apiFetch("/api/admin/wallet/sync", { method: "POST" });
		actionMessage.value =
			"Wallet sync queued. The journal will update shortly.";
		window.setTimeout(() => refresh(), 2500);
	} catch (requestError: any) {
		actionError.value =
			requestError?.data?.message ||
			requestError?.message ||
			"Could not queue wallet sync";
	} finally {
		syncing.value = false;
	}
};

const markPrizePaid = async (row: PrizeSettlementRow) => {
	if (row.rank == null || row.paid_at || !row.claimed_at) return;
	const key = `${row.campaign_id}:${row.rank}`;
	const note = window.prompt(
		`Optional payment note for ${row.character_name || row.character_id}:`,
		"",
	);
	if (note === null) return;
	markingPaid.value = key;
	actionError.value = "";
	try {
		await apiFetch(
			`/api/admin/campaign-prizes/${row.campaign_id}/${row.rank}/paid`,
			{ method: "POST", body: { note } },
		);
		actionMessage.value = `Marked ${row.character_name || `rank ${row.rank}`} as paid.`;
		await refresh();
	} catch (requestError: any) {
		actionError.value =
			requestError?.data?.message ||
			requestError?.message ||
			"Could not mark prize as paid";
	} finally {
		markingPaid.value = "";
	}
};

const referenceStatus = (status: number) =>
	["Matched", "Unmatched", "Late", "Invalid"][status] ?? "Unknown";
</script>

<template>
    <div class="space-y-4">
        <div
            v-if="authorizationBanner"
            class="rounded-lg border px-4 py-3 text-sm"
            :class="authorizationBanner.kind === 'success'
                ? 'border-green-500/20 bg-green-500/10 text-green-300'
                : 'border-red-500/20 bg-red-500/10 text-red-300'"
        >
            {{ authorizationBanner.text }}
        </div>

        <div v-if="actionMessage" class="rounded-lg border border-blue-500/20 bg-blue-500/10 px-4 py-3 text-sm text-blue-300">
            {{ actionMessage }}
        </div>
        <div v-if="actionError" class="rounded-lg border border-red-500/20 bg-red-500/10 px-4 py-3 text-sm text-red-300">
            {{ actionError }}
        </div>

        <div class="glass-panel p-5">
            <div class="flex flex-col lg:flex-row lg:items-start gap-5">
                <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-3">
                        <div class="w-10 h-10 rounded-lg bg-yellow-500/15 flex items-center justify-center">
                            <Icon name="lucide:wallet-cards" class="text-xl text-yellow-400" />
                        </div>
                        <div>
                            <h2 class="text-base font-semibold text-white">
                                {{ data?.corporation.name || 'EVE-KILL.com' }} wallet
                            </h2>
                            <p class="text-xs text-gray-500">
                                Corporation {{ data?.corporation.corporation_id || 98779905 }} · refreshed hourly
                            </p>
                        </div>
                    </div>

                    <div v-if="data?.authorization" class="mt-4 grid sm:grid-cols-2 gap-x-6 gap-y-2 text-xs">
                        <div>
                            <span class="text-gray-600">Authorized character</span>
                            <div class="text-gray-300 mt-0.5">
                                {{ data.authorization.authorized_character_name }}
                                <span class="text-gray-600">({{ data.authorization.authorized_character_id }})</span>
                            </div>
                        </div>
                        <div>
                            <span class="text-gray-600">Status</span>
                            <div
                                class="mt-0.5 font-medium"
                                :class="data.authorization.disabled ? 'text-red-400' : 'text-green-400'"
                            >
                                {{ data.authorization.disabled ? 'Authorization disabled' : 'Active' }}
                            </div>
                        </div>
                        <div>
                            <span class="text-gray-600">Last balance sync</span>
                            <div class="text-gray-300 mt-0.5">{{ formatDate(data.authorization.last_balance_sync) }}</div>
                        </div>
                        <div>
                            <span class="text-gray-600">Last journal sync</span>
                            <div class="text-gray-300 mt-0.5">{{ formatDate(data.authorization.last_journal_sync) }}</div>
                        </div>
                    </div>
                    <p v-else class="mt-4 text-sm text-gray-400">
                        Authorize an EVE-KILL.com character with Accountant or Junior Accountant to start importing the corporation wallet.
                    </p>

                    <div v-if="data?.authorization?.last_error" class="mt-4 rounded-md bg-red-500/10 px-3 py-2 text-xs text-red-300 break-words">
                        {{ data.authorization.last_error }}
                    </div>
                </div>

                <div class="flex flex-wrap gap-2">
                    <button
                        class="flex items-center gap-2 px-3 py-2 rounded-md bg-blue-500/15 text-sm font-medium text-blue-300 hover:bg-blue-500/25 disabled:opacity-50"
                        :disabled="authorizing"
                        @click="authorize"
                    >
                        <Icon name="lucide:key-round" />
                        {{ data?.authorization ? 'Re-authorize ESI' : 'Authorize ESI wallet' }}
                    </button>
                    <button
                        v-if="data?.authorization"
                        class="flex items-center gap-2 px-3 py-2 rounded-md bg-white/[0.05] text-sm text-gray-300 hover:bg-white/[0.09] disabled:opacity-50"
                        :disabled="syncing || data.authorization.disabled"
                        @click="syncNow"
                    >
                        <Icon name="lucide:refresh-cw" :class="{ 'animate-spin': syncing }" />
                        Sync now
                    </button>
                    <button
                        class="flex items-center gap-2 px-3 py-2 rounded-md bg-white/[0.05] text-sm text-gray-400 hover:bg-white/[0.09]"
                        :disabled="pending"
                        @click="() => refresh()"
                    >
                        <Icon name="lucide:rotate-cw" :class="{ 'animate-spin': pending }" />
                        Refresh
                    </button>
                </div>
            </div>
        </div>

        <div v-if="error" class="glass-panel p-5 text-sm text-red-400">
            Could not load the corporation wallet: {{ error.message }}
        </div>

        <template v-if="data">
            <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
                <div class="glass-panel p-4 col-span-2">
                    <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500">Total balance</div>
                    <div class="mt-1 text-xl font-bold text-yellow-400 tabular-nums">{{ formatIsk(data.totalBalance) }}</div>
                </div>
                <div
                    v-for="wallet in divisions"
                    :key="wallet.division"
                    class="glass-panel p-4"
                >
                    <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500">Division {{ wallet.division }}</div>
                    <div class="mt-1 text-sm font-semibold text-white tabular-nums">{{ formatIsk(wallet.balance) }}</div>
                </div>
            </div>

            <div v-if="prizeSettlementGroups.length" class="glass-panel overflow-hidden">
                <div class="p-4 border-b border-white/[0.06]">
                    <h2 class="text-sm font-semibold text-white">Campaign prize settlements</h2>
                    <p class="text-fine text-gray-600 mt-0.5">Verified claims waiting for manual in-game payment.</p>
                </div>
                <div class="divide-y divide-white/[0.06]">
                    <div v-for="pool in prizeSettlementGroups" :key="pool.campaign_id" class="p-4">
                        <div class="flex flex-wrap items-start justify-between gap-3 mb-3">
                            <div>
                                <NuxtLink :to="`/campaign/${pool.campaign_id}`" class="text-sm font-semibold text-blue-300 hover:text-blue-200">
                                    {{ pool.campaign_name }}
                                </NuxtLink>
                                <div class="text-fine text-gray-600 mt-0.5">
                                    {{ pool.campaign_id }} · finalized {{ formatDate(pool.finalized_at) }}
                                </div>
                            </div>
                            <div class="text-right">
                                <div class="text-sm font-semibold text-yellow-400 tabular-nums">{{ formatIsk(pool.funded_total) }}</div>
                                <div class="text-fine" :class="pool.pool_status === 2 ? 'text-green-400' : 'text-yellow-400'">
                                    {{ pool.pool_status === 2 ? 'Paid' : 'Ready for payout' }}
                                </div>
                            </div>
                        </div>
                        <div class="overflow-x-auto">
                            <table class="w-full text-xs">
                                <tbody>
                                    <tr v-for="(result, resultIndex) in pool.results" :key="result.rank ?? resultIndex" class="border-t border-white/[0.04]">
                                        <td class="py-2 pr-3 text-gray-500 w-10">#{{ result.rank }}</td>
                                        <td class="py-2 pr-3">
                                            <NuxtLink :to="`/character/${result.character_id}`" class="text-gray-300 hover:text-blue-300">
                                                {{ result.character_name || result.character_id }}
                                            </NuxtLink>
                                        </td>
                                        <td class="py-2 px-3 text-right text-gray-400 tabular-nums">
                                            {{ result.payout_percentage }}% · {{ formatIsk(result.payout_amount) }}
                                        </td>
                                        <td class="py-2 pl-3 text-right">
                                            <span v-if="result.paid_at" class="text-green-400">Paid {{ formatDate(result.paid_at) }}</span>
                                            <button v-else-if="result.claimed_at"
                                                class="px-2.5 py-1 rounded bg-green-500/15 text-green-300 hover:bg-green-500/25 disabled:opacity-50"
                                                :disabled="markingPaid === `${result.campaign_id}:${result.rank}`"
                                                @click="markPrizePaid(result)">
                                                Mark paid
                                            </button>
                                            <span v-else class="text-gray-600">Awaiting claim</span>
                                        </td>
                                    </tr>
                                </tbody>
                            </table>
                        </div>
                    </div>
                </div>
            </div>

            <div v-if="data.walletReferences.length" class="glass-panel overflow-hidden">
                <div class="p-4 border-b border-white/[0.06]">
                    <h2 class="text-sm font-semibold text-white">Wallet references</h2>
                    <p class="text-fine text-gray-600 mt-0.5">Matched and rejected campaign and EK Wallet references.</p>
                </div>
                <div class="overflow-x-auto">
                    <table class="w-full text-xs">
                        <thead>
                            <tr class="text-left text-gray-600 border-b border-white/[0.06]">
                                <th class="px-4 py-2.5 font-medium">Date</th>
                                <th class="px-4 py-2.5 font-medium">Reference</th>
                                <th class="px-4 py-2.5 font-medium">Status</th>
                                <th class="px-4 py-2.5 font-medium text-right">Amount</th>
                                <th class="px-4 py-2.5 font-medium">Note</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr v-for="reference in data.walletReferences" :key="`${reference.division}-${reference.journal_id}`"
                                class="border-b border-white/[0.04] last:border-0">
                                <td class="px-4 py-3 text-gray-500 whitespace-nowrap">{{ formatDate(reference.date) }}</td>
                                <td class="px-4 py-3 text-gray-300 font-mono">{{ reference.reference_type }}:{{ reference.reference_id }}</td>
                                <td class="px-4 py-3"
                                    :class="reference.status === 0 ? 'text-green-400' : reference.status === 2 ? 'text-yellow-400' : 'text-red-400'">
                                    {{ referenceStatus(reference.status) }}
                                </td>
                                <td class="px-4 py-3 text-right text-gray-300 tabular-nums whitespace-nowrap">{{ formatIsk(reference.amount) }}</td>
                                <td class="px-4 py-3 text-gray-600">{{ reference.note || '—' }}</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>

            <div class="glass-panel overflow-hidden">
                <div class="p-4 border-b border-white/[0.06] flex flex-col sm:flex-row sm:items-center gap-3">
                    <div class="flex-1">
                        <h2 class="text-sm font-semibold text-white">Wallet journal</h2>
                        <p class="text-fine text-gray-600 mt-0.5">The latest entries retained by ESI, newest first.</p>
                    </div>
                    <SelectMenu
                        v-model="division"
                        placeholder="All divisions"
                        :options="[
                            { value: '', label: 'All divisions' },
                            ...Array.from({ length: 7 }, (_, i) => ({ value: String(i + 1), label: `Division ${i + 1}` })),
                        ]"
                    />
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
                <button class="glass-panel px-3 py-1.5 text-xs text-gray-400 disabled:opacity-30" :disabled="page <= 1" @click="page--">Previous</button>
                <span class="text-fine text-gray-600">Page {{ page }}</span>
                <button class="glass-panel px-3 py-1.5 text-xs text-gray-400 disabled:opacity-30" :disabled="!data.hasMore" @click="page++">Next</button>
            </div>
        </template>
    </div>
</template>
