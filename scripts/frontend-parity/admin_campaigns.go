package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The same creation timestamp deliberately exercises stable admin pagination.
func seedAdminCampaignFixtures(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `INSERT INTO campaigns(campaign_id,name,description,created_by_character_id,status,visibility,start_time,end_time,location,stats,created_at,processing_paused,processing_note,estimated_killmails,last_processing_duration_ms,last_processing_killmails,last_processing_error)
	SELECT 'AdminCamp'||to_char(i,'FM00000'),'Fixture admin campaign '||to_char(i,'FM00'),'Disposable campaign administration data',
	CASE WHEN i%2=0 THEN 4343 ELSE 4242 END,i%3,i%3,now()-interval '30 days',now()+interval '2 days','{"systemIds":[30000142]}'::jsonb,
	jsonb_build_object('totals',jsonb_build_object('killCount',i*1234)),now()-interval '12 hours',i%7=0,
	CASE WHEN i%7=0 THEN 'Fixture workload limit' ELSE NULL END,i*123456,
	CASE WHEN i%3=0 THEN 725 WHEN i%3=1 THEN 2350 ELSE 126500 END,
	CASE WHEN i%4=0 THEN NULL WHEN i=1 THEN 0 ELSE i*10 END,
	CASE WHEN i%11=0 THEN '<script>fixture processing failure</script>' ELSE NULL END
	FROM generate_series(1,50) i`)
	return err
}
