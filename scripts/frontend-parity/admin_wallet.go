package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Fake ESI tokens are never sent anywhere; the fixture starts no workers.
func seedAdminWalletFixtures(ctx context.Context, pool *pgxpool.Pool) error {
	for _, sql := range []string{
		`INSERT INTO corporations(corporation_id,name,ticker) VALUES(98779905,'EVE-KILL.com','EVEKI') ON CONFLICT DO NOTHING`,
		`INSERT INTO corporation_wallet_tokens(corporation_id,authorized_character_id,authorized_character_name,authorized_character_owner_hash,authorized_by_admin_character_id,access_token,refresh_token,token_expiry,scopes,last_balance_sync,last_journal_sync)
		 VALUES(98779905,4242,'Fixture Owner','fixture-owner',4242,'fixture-wallet-access','fixture-wallet-refresh',now()+interval '1 day',ARRAY['publicData','esi-wallet.read_corporation_wallets.v1'],now()-interval '1 hour',now()-interval '2 hours')`,
		`INSERT INTO corporation_wallet_balances(corporation_id,division,balance) VALUES(98779905,1,765432109.87),(98779905,2,0),(98779905,3,12000000.50),(98779905,5,432.01),(98779905,7,1234567890123.45)`,
		`INSERT INTO corporation_wallet_journal(corporation_id,division,journal_id,date,ref_type,description,amount,balance,first_party_id,second_party_id,reason)
		 SELECT 98779905,n%7+1,81000+n,now()-(n/3)*interval '1 minute',
		 CASE n%4 WHEN 0 THEN 'player_donation' WHEN 1 THEN 'corporation_account_withdrawal' WHEN 2 THEN 'market_transaction' ELSE 'bounty_prize' END,
		 'Fixture journal entry '||n||CASE WHEN n=6 THEN ' <script>plain description</script>' ELSE '' END,
		 CASE n%4 WHEN 0 THEN 1234.50 WHEN 1 THEN -2100.75 WHEN 2 THEN 0 ELSE NULL END,
		 CASE WHEN n%5=0 THEN NULL ELSE 123456789.50-n END,
		 CASE WHEN n%4=0 THEN 4242 ELSE NULL END,CASE WHEN n%4=1 THEN 4343 ELSE NULL END,
		 CASE WHEN n%3=0 THEN 'Fixture reason '||n||' <script>plain reason</script>' ELSE NULL END
		 FROM generate_series(1,115)n`,
		`INSERT INTO wallet_journal_references(corporation_id,division,journal_id,reference_type,reference_id,status,amount,note)
		 SELECT corporation_id,division,journal_id,CASE WHEN journal_id%2=0 THEN 'campaign' ELSE 'wallet' END,'fixture-reference-'||journal_id,journal_id%4,amount,
		 CASE WHEN journal_id%3=0 THEN NULL ELSE 'Fixture reference note <script>plain note</script>' END
		 FROM corporation_wallet_journal WHERE corporation_id=98779905 AND journal_id BETWEEN 81001 AND 81008`,
		`UPDATE campaign_prize_results SET claimed_at=now()-interval '30 minutes',claimed_by_character_id=character_id WHERE campaign_id='PrizeReady0001' AND rank IN(1,3)`,
		`UPDATE campaign_prize_results SET paid_at=now()-interval '10 minutes',paid_by_admin_character_id=4242,payment_note='Fixture prior payment' WHERE campaign_id='PrizeReady0001' AND rank=3`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			return err
		}
	}
	return nil
}
