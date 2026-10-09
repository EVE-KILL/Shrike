export interface WalletAuthorization {
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

export interface WalletBalance {
	division: number;
	balance: string;
	updated_at: string;
}

export interface WalletJournalEntry {
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

export interface PrizeSettlementRow {
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

export interface WalletReference {
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

export interface WalletData {
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

export const adminWalletDivisions = (balances: WalletBalance[] = []) => {
  const byDivision = new Map(balances.map(wallet => [wallet.division, wallet]))
  return Array.from({ length: 7 }, (_, index) => byDivision.get(index + 1) ?? { division: index + 1, balance: '0.00', updated_at: '' })
}
export const adminWalletSettlementGroups = (rows: PrizeSettlementRow[] = []) => {
  const groups = new Map<string, Pick<PrizeSettlementRow, 'campaign_id' | 'campaign_name' | 'pool_status' | 'funded_total' | 'finalized_at'> & { results: PrizeSettlementRow[] }>()
  for (const row of rows) {
    const group = groups.get(row.campaign_id) ?? { campaign_id: row.campaign_id, campaign_name: row.campaign_name, pool_status: row.pool_status, funded_total: row.funded_total, finalized_at: row.finalized_at, results: [] }
    if (row.rank != null) group.results.push(row)
    groups.set(row.campaign_id, group)
  }
  return [...groups.values()]
}
export const adminWalletIsk = (value: string | number | null | undefined) => {
  if (value === null || value === undefined || value === '') return '—'
  const amount = Number(value)
  if (!Number.isFinite(amount)) return '—'
  return `${new Intl.NumberFormat('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(amount)} ISK`
}
export const adminWalletDate = (value: string | null | undefined) => value ? new Intl.DateTimeFormat('en-GB', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) : 'Never'
export const adminWalletReferenceType = (value: string) => value.split('_').map(part => part.charAt(0).toUpperCase() + part.slice(1)).join(' ')
export const adminWalletReferenceStatus = (status: number) => ['Matched', 'Unmatched', 'Late', 'Invalid'][status] ?? 'Unknown'
export const adminWalletAuthorizationBanner = (query: Record<string, unknown>) => {
  if (query.wallet_auth === 'ok') return { kind: 'success', text: typeof query.message === 'string' ? query.message : 'Wallet authorization saved. The first journal import has been queued.' }
  if (query.wallet_auth === 'error') return { kind: 'error', text: typeof query.message === 'string' ? query.message : 'Wallet authorization failed.' }
  return null
}
export const adminWalletPaymentPrompt = (row: Pick<PrizeSettlementRow, 'character_name' | 'character_id'>) => `Optional payment note for ${row.character_name || row.character_id}:`
export const adminWalletPaidMessage = (row: PrizeSettlementRow) => `Marked ${row.character_name || `rank ${row.rank}`} as paid.`
