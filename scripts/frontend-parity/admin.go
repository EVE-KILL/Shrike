package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Optional, owned-database records for admin table paging and queue badges.
func seedAdminFixtures(ctx context.Context, pool *pgxpool.Pool) error {
	for _, query := range []string{
		`INSERT INTO users(character_id,character_name,character_owner_hash,session_id,last_login,created_at) SELECT 44000+i,'Fixture user '||to_char(i,'FM00'),'fixture',gen_random_uuid(),now()-i*interval '1 minute',now()-i*interval '1 day' FROM generate_series(1,54)i`,
		`UPDATE campaigns SET last_processing_error='Disposable fixture processing failure' WHERE campaign_id='FourSides00001'`,
		`INSERT INTO user_config(character_id,key,value) VALUES(4242,'fixture-object','{"title":"<script>escaped text</script>","2":true,"10":false}'),(4242,'fixture-array','["fixture",42,{"enabled":true}]')`,
		`ANALYZE killmails`,
		`INSERT INTO esi_request_logs(character_id,endpoint,source,success,status_code,error_message,items_returned,new_items,new_item_ids,request_duration_ms) VALUES
		 (4242,'/characters/4242/killmails/recent/','fixture-admin-esi',true,200,NULL,10,2,'[2000000001,2199999999]',4500),
		 (4242,'/corporations/10/killmails/recent/','fixture-admin-esi',false,304,NULL,0,0,'[]',50),
		 (4343,'/corporations/11/killmails/recent/','fixture-admin-esi',false,503,'<script>fixture upstream error</script>',0,0,'[]',5100),
		 (4343,'/characters/4343/assets/','fixture-admin-esi',false,400,'Fixture permission denied',0,0,'[]',500)`,
		`INSERT INTO announcements(tier,title,body_md,body_html,color,starts_at,expires_at,created_by,archived_at) VALUES
		 (1,'Fixture active ticker','Disposable announcement text','<p>Disposable announcement text</p>','info',now()-interval '1 hour',now()+interval '1 day',4242,NULL),
		 (2,'Fixture scheduled banner','**Scheduled** fixture with <script>plain text</script>','<p>Scheduled fixture</p>','warning',now()+interval '1 day',now()+interval '7 days',4242,NULL),
		 (3,'Fixture expired modal','Expired fixture','<p>Expired fixture</p>','danger',now()-interval '2 days',now()-interval '1 day',4242,NULL),
		 (1,'Fixture archived ticker','','','success',now()-interval '1 day',now()+interval '1 day',4242,now())`,
	} {
		if _, err := pool.Exec(ctx, query); err != nil {
			return err
		}
	}
	return nil
}
