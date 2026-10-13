package main

import (
	"context"

	"github.com/eve-kill/shrike/internal/queue"
	"github.com/jackc/pgx/v5/pgxpool"
)

// No workers are started: all River controls remain inside this disposable DB.
func seedAdminRiverFixtures(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := queue.Migrate(ctx, pool); err != nil {
		return err
	}
	for _, sql := range []string{
		`INSERT INTO river_queue(name,updated_at,paused_at) VALUES
		 ('achievements',now(),NULL),('campaign_processing',now()-interval '1 hour',now()-interval '2 hours'),('cron',now()-interval '10 minutes',NULL)`,
		`INSERT INTO river_job(id,queue,kind,state,attempt,max_attempts,priority,created_at,scheduled_at,attempted_at,finalized_at,attempted_by,args,metadata,errors)
		 SELECT 30000+n,CASE WHEN n%3=1 THEN 'campaign_processing' ELSE 'achievements' END,'fixture_report',
		 (ARRAY['available','running','scheduled','retryable','discarded','cancelled','completed'])[n%7+1]::river_job_state,
		 CASE WHEN n%7=0 THEN 0 ELSE 2 END,3,n%4+1,
		 now()-n*interval '1 minute',now()+n*interval '1 minute',CASE WHEN n%7=0 THEN NULL ELSE now()-interval '1 minute' END,
		 CASE WHEN n%7 IN(4,5,6) THEN now()-interval '10 seconds' ELSE NULL END,
		 CASE WHEN n%7=0 THEN ARRAY[]::text[] ELSE ARRAY['fixture-worker-a','fixture-worker-b'] END,
		 jsonb_build_object('killmail_id',2000000001,'label','Fixture input <script>plain text</script>','nested',jsonb_build_object('pilots',ARRAY[4242,4343]),'long_line',repeat('fixture ',80)),
		 CASE WHEN n%7=6 THEN jsonb_build_object('output',jsonb_build_object('processed',17,'message','Fixture output <img src=x onerror=alert(1)>')) ELSE '{}'::jsonb END,
		 CASE WHEN n%7 IN(3,4) THEN ARRAY[jsonb_build_object('error','Fixture retry <script>plain error</script>','at',now()-interval '30 seconds','attempt',2)] ELSE ARRAY[]::jsonb[] END
		 FROM generate_series(0,134) n`,
		`INSERT INTO river_job(id,queue,kind,max_attempts,args) VALUES(30135,'cron','fixture_cron',3,'{"window":"Fixture cron"}'),(30136,'fixture_unregistered','fixture_orphan',3,'{}')`,
		`SELECT setval(pg_get_serial_sequence('river_job','id'),(SELECT max(id) FROM river_job))`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			return err
		}
	}
	return nil
}
