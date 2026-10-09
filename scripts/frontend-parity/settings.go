package main

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func seedSettings(ctx context.Context, pool *pgxpool.Pool) error {
	for _, query := range []string{
		`INSERT INTO regions(region_id,name) VALUES(90000001,'Fixture Region') ON CONFLICT DO NOTHING`,
		`INSERT INTO constellations(constellation_id,constellation_name,region_id) VALUES(90000002,'Fixture Constellation',90000001) ON CONFLICT DO NOTHING`,
		`INSERT INTO solar_systems(solar_system_id,system_name,constellation_id,region_id,security) VALUES(90000003,'Fixture System',90000002,90000001,0.3) ON CONFLICT DO NOTHING`,
		`INSERT INTO blog_posts(id,slug,title,excerpt,body_md,body_html,status,author_id,author_name,published_at,tags) VALUES(91001,'fixture-settings-post','Disposable published post','A local frontend comparison post',E'## A local article\n\nDisposable comparison content.','<h2>A local article</h2><p>Disposable comparison content.</p>',1,4242,'Fixture Owner',now()-interval '1 day',ARRAY['fixture','settings']),(91002,'fixture-settings-draft','Disposable draft',NULL,'Draft content','<p>Draft content</p>',0,4242,'Fixture Owner',NULL,ARRAY['fixture'])`,
		`INSERT INTO comments(target_type,target_id,target_slug,body_md,body_html,character_id,character_name,corporation_id,corporation_name,created_at,edited_at,moderation_status) SELECT ((i-1)%10)+1,CASE ((i-1)%10)+1 WHEN 1 THEN 2000000001 WHEN 2 THEN 4242 WHEN 3 THEN 10 WHEN 4 THEN 70121 WHEN 5 THEN 30000142 WHEN 7 THEN 2000000001 WHEN 9 THEN 91001 ELSE 0 END,CASE ((i-1)%10)+1 WHEN 6 THEN 'about' WHEN 8 THEN 'PublicFit00001' WHEN 10 THEN 'PrizeOpen00001' ELSE NULL END,'Fixture comment '||i,'<p><strong>Fixture comment '||i||'</strong> with a <a href="https://eve-kill.com">link</a>.</p>',4242,'Fixture Owner',10,'Fixture Red',now()-i*interval '1 hour',CASE WHEN i%7=0 THEN now()-interval '30 minutes' ELSE NULL END,CASE WHEN i%13=0 THEN 2 ELSE 0 END FROM generate_series(1,36) i`,
		`INSERT INTO comments(target_type,target_id,body_md,body_html,character_id,character_name,corporation_id,corporation_name) VALUES(2,4343,'Reader-only fixture','<p>Reader-only fixture</p>',4343,'Fixture Reader',11,'Fixture Blue')`,
		`INSERT INTO comments(target_type,target_id,target_slug,parent_id,root_id,depth,body_md,body_html,character_id,character_name,corporation_id,corporation_name,created_at) SELECT target_type,target_id,target_slug,id,id,1,'Disposable reply to '||body_md,'<p>Disposable reply to '||body_md||'</p>',4343,'Fixture Reader',11,'Fixture Blue',now()-id*interval '1 minute' FROM comments WHERE id IN(1,6,8,9,10)`,
		`INSERT INTO comments(target_type,target_id,parent_id,root_id,depth,body_md,body_html,character_id,character_name,corporation_id,corporation_name,created_at) SELECT target_type,target_id,id,id,1,'Reader-specific reply','<p>Reader-specific reply</p>',4242,'Fixture Owner',10,'Fixture Red',now() FROM comments WHERE character_id=4343 AND parent_id IS NULL LIMIT 1`,
		`INSERT INTO entity_trackers(character_id,target_type,target_id,target_name,notifications_enabled) VALUES(4242,'character',4242,'Fixture Owner',true),(4242,'system',30000142,'Jita',true),(4343,'corporation',11,'Fixture Blue',false)`,
		`INSERT INTO entity_tracker_events(tracker_id,killmail_id,match_role,occurred_at) SELECT tracker_id,2000000001,CASE target_type WHEN 'system' THEN 'location' ELSE 'victim' END,now() FROM entity_trackers WHERE character_id=4242`,
		`INSERT INTO entity_tracker_notifications(event_id,character_id,created_at) SELECT event_id,4242,now()-interval '30 seconds' FROM entity_tracker_events`,
		`UPDATE corporations SET ceo_id=CASE corporation_id WHEN 10 THEN 4242 ELSE 4444 END,description='Fixture corporation bio' WHERE corporation_id IN (10,11)`,
		`UPDATE alliances SET executor_corporation_id=CASE alliance_id WHEN 70121 THEN 10 ELSE 11 END WHERE alliance_id IN (70121,70122)`,
		`UPDATE characters SET description='Fixture character bio' WHERE character_id IN (4242,4343)`,
		`INSERT INTO ek_wallet_transactions(character_id,type,amount,balance_after,description,external_reference,campaign_id,created_at) SELECT 4242,CASE WHEN i=54 THEN 3 WHEN i=55 THEN 4 ELSE 0 END,CASE WHEN i=1 THEN 48000.50 WHEN i=54 THEN -1000 ELSE 1000 END,CASE WHEN i<54 THEN 48000.50+(i-1)*1000 WHEN i=54 THEN 99000.50 ELSE 100000.50 END,'Fixture wallet entry '||i,'fixture-settings-wallet-'||i,CASE WHEN i=54 THEN 'PrizeOpen00001' WHEN i=55 THEN 'PrizeReady0001' ELSE NULL END,now()-(56-i)*interval '1 hour' FROM generate_series(1,55) i ON CONFLICT DO NOTHING`,
		`INSERT INTO ek_wallet_reservations(character_id,external_reference,transaction_type,amount,description,expires_at) VALUES(4242,'fixture-settings-pending-purchase',1,1000,'Fixture pending purchase',now()+interval '2 hours') ON CONFLICT DO NOTHING`,
		`UPDATE users SET created_at=now()-interval '90 days',last_login=now()-interval '2 hours' WHERE character_id IN (4242,4343)`,
		`UPDATE user_sessions SET ip_address='127.0.0.1',country_code='DK',user_agent='Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) AppleWebKit/537.36 Chrome/130.0.0.0 Safari/537.36',created_at=now()-interval '2 hours' WHERE character_id IN (4242,4343)`,
		`INSERT INTO user_esi_tokens(character_id,access_token,refresh_token,token_expiry,scopes,last_fetched) VALUES(4242,'fixture-access','fixture-refresh',now()+interval '1 hour',ARRAY['esi-killmails.read_killmails.v1','esi-killmails.read_corporation_killmails.v1'],now()-interval '15 minutes')`,
		`INSERT INTO esi_request_logs(character_id,endpoint,source,success,status_code,new_items,request_duration_ms,created_at) SELECT 4242,'/characters/4242/killmails/recent/','fixture-settings',i<>3,CASE WHEN i=3 THEN 503 ELSE 200 END,CASE WHEN i=3 THEN 0 ELSE 7 END,150+i,now()-i*interval '1 minute' FROM generate_series(1,8) i`,
		`INSERT INTO esi_request_logs(character_id,endpoint,source,success,status_code,error_message,items_returned,new_items,request_duration_ms,created_at) SELECT 4242,CASE WHEN i%2=0 THEN '/corporations/10/killmails/recent/' ELSE '/characters/4242/killmails/recent/' END,'fixture-settings-history',i%11<>0,CASE WHEN i%11=0 THEN 503 WHEN i%7=0 THEN 304 ELSE 200 END,CASE WHEN i%11=0 THEN 'Fixture upstream error' ELSE NULL END,10,CASE WHEN i%11=0 THEN 0 ELSE 2 END,100+i,now()-(30+i)*interval '1 hour' FROM generate_series(1,65) i`,
	} {
		if _, err := pool.Exec(ctx, query); err != nil {
			return err
		}
	}
	for _, entry := range []struct{ Country, Agent string }{
		{"DE", "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) AppleWebKit/605.1.15 Version/17.0 Safari/605.1.15"},
		{"US", "Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 Chrome/130.0.0.0 Mobile Safari/537.36"},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO user_sessions(session_id,character_id,ip_address,country_code,user_agent,created_at,last_seen_at,expires_at) VALUES($1,4242,'127.0.0.1',$2,$3,now()-interval '2 days',now()-interval '2 hours',now()+interval '24 hours')`, uuid.NewString(), entry.Country, entry.Agent); err != nil {
			return err
		}
	}
	return nil
}
