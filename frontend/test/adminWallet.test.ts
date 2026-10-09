import { expect, test } from 'vitest'
import { adminWalletDivisions, adminWalletSettlementGroups, adminWalletIsk, adminWalletAuthorizationBanner, adminWalletReferenceStatus, type PrizeSettlementRow } from '../src/utils/adminWallet'

test('wallet display preserves absent divisions and nullable amounts', () => {
  const divisions = adminWalletDivisions([{ division: 3, balance: '1234.50', updated_at: '' }])
  expect(divisions.map(row => row.division)).toEqual([1, 2, 3, 4, 5, 6, 7])
  expect(divisions[2]?.balance).toBe('1234.50')
  expect(divisions[0]?.balance).toBe('0.00')
  expect(adminWalletIsk(null)).toBe('—')
  expect(adminWalletIsk('NaN')).toBe('—')
  expect(adminWalletIsk('-1234.50')).toBe('-1,234.50 ISK')
  expect(adminWalletReferenceStatus(2)).toBe('Late')
})
test('left-joined empty prize pools keep their card without a phantom winner', () => {
  const row = (campaign_id: string, rank: number | null) => ({ campaign_id, rank, campaign_name: campaign_id, pool_status: 1, funded_total: '100.00', finalized_at: null }) as PrizeSettlementRow
  const groups = adminWalletSettlementGroups([row('ready', 1), row('empty', null), row('ready', 2)])
  expect(groups.map(group => group.campaign_id)).toEqual(['ready', 'empty'])
  expect(groups[0]?.results.map(result => result.rank)).toEqual([1, 2])
  expect(groups[1]?.results).toEqual([])
  expect(adminWalletAuthorizationBanner({ wallet_auth: 'ok', message: '<script>plain callback</script>' })?.text).toBe('<script>plain callback</script>')
  expect(adminWalletAuthorizationBanner({ wallet_auth: ['ok', 'error'] })).toBeNull()
})
