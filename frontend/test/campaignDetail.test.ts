// @vitest-environment happy-dom
import {expect, test} from 'vitest'
import {createVictimSideLookup} from '../src/browser/killlist-sides'
import {campaignContributionError, campaignContributionMessage} from '../src/browser/campaign-prize'

test('campaign victim assignment preserves character, corporation, alliance precedence', () => {
    const lookup = createVictimSideLookup([
        {characterIds: [1], corporationIds: [10], allianceIds: [100]},
        {characterIds: [2], corporationIds: [10, 11], allianceIds: [101]},
    ])
    expect(lookup({characterId: 1, corporationId: 10, allianceId: 101})).toBe(0)
    expect(lookup({characterId: 3, corporationId: 10, allianceId: 100})).toBe(1)
    expect(lookup({characterId: 3, corporationId: 12, allianceId: 100})).toBe(0)
    expect(lookup({characterId: 3, corporationId: 12, allianceId: 102})).toBeUndefined()
})

test('campaign prize contributions accept decimal ISK and reject invalid or excessive amounts', () => {
    for (const amount of ['1', '0.01', '100.50']) expect(campaignContributionError(amount, 100.5)).toBe('')
    for (const amount of ['', '0', '0.00', '-1', '1e3', '1.001', '1,000'])
        expect(campaignContributionError(amount, 100.5)).toContain('positive ISK')
    expect(campaignContributionError('100.51', 100.5)).toContain('available EK Wallet')
    // API amounts are exact decimal strings, while the source formatter accepts numbers.
    expect(campaignContributionMessage('100.25')).toBe('100 added to the prize pool.')
    expect(campaignContributionMessage('10000.50')).toBe('10.0k added to the prize pool.')
})
