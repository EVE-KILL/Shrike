package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func seedAdminReviewFixtures(ctx context.Context, pool *pgxpool.Pool) error {
	for i := 1; i <= 12; i++ {
		kind := i
		if kind > 10 {
			kind = 2
		}
		id := int64(0)
		slug := any(nil)
		domain := any(nil)
		switch kind {
		case 1:
			id = 2000000001
		case 2:
			id = 4343
		case 3:
			id = 11
		case 4:
			id = 70122
		case 5:
			id = 30000142
			domain = 9101
		case 6:
			slug = "about"
		case 7:
			slug = "FixtureBattle1"
		case 8:
			slug = "PublicFit00001"
		case 9:
			id = 91001
		case 10:
			slug = "PrizeOpen00001"
		}
		body := fmt.Sprintf("<p><strong>Review comment %02d</strong> with <em>formatting</em> and a <a href='/faq'>link</a>.</p>", i)
		if i == 8 {
			body += `<img src="/api/admin/domains/9101/assets/9201/preview" alt="Fixture review image"><script>alert('stored script')</script>`
		}
		if i == 10 {
			body += `<div class="embed embed-youtube"><iframe src="https://www.youtube.com/embed/dQw4w9WgXcQ" loading="lazy" allowfullscreen></iframe></div>`
		}
		flagged := i%2 == 1
		reports := i % 4
		status := i % 4
		visibility := status
		if i == 1 || i == 2 || i == 7 || i == 9 || i == 10 {
			flagged = true
			status = 1
			visibility = 1
		}
		if i == 12 {
			status = 0
			visibility = 0
			reports = 0
			flagged = false
		}
		if i == 5 || i == 6 {
			status = 0
			visibility = 0
		}
		if i == 8 {
			status = 2
			visibility = 2
			flagged = true
		}
		if _, err := pool.Exec(ctx, `INSERT INTO comments(id,target_type,target_id,target_slug,domain_id,body_md,body_html,character_id,character_name,corporation_id,corporation_name,flagged,reports_count,moderation_status,visibility,created_at,deleted_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,4343,'Fixture Reader',11,'Fixture Blue',$8,$9,$10,$11,now()-$12*interval '1 hour',CASE WHEN $12=11 THEN now() ELSE NULL END)`, 20000+i, kind, id, slug, domain, fmt.Sprintf("Review comment %02d", i), body, flagged, reports, status, visibility, i); err != nil {
			return err
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO comments(id,target_type,target_id,body_md,body_html,character_id,character_name,corporation_id,corporation_name,created_at)
	SELECT 21000+i,2,4343,'Review archive row '||i,'<p>Review archive row '||i||'</p>',4343,'Fixture Reader',11,'Fixture Blue',now()-i*interval '1 day' FROM generate_series(1,95)i`); err != nil {
		return err
	}
	for _, spec := range []struct {
		ID, Kind, Target, Status                    int
		Body, Format, HTML, Action, Category, Notes string
		Score                                       float64
	}{
		{10001, 0, 20001, 0, "Review pending kill", "markdown", "<p><strong>Review pending kill</strong> awaiting approval.</p>", "flag", "harassment", "", 0.91},
		{10002, 0, 20002, 0, "Review pending profile", "markdown", "<p>Review pending profile.</p>", "flag", "violence", "", 0},
		{10003, 1, 4343, 0, "Review character bio", "eve_html", `<p><span style="color:#ff00ff;font-size:18px">Review character bio</span> with <a href='/faq'>a link</a>.</p><script>alert(1)</script>`, "flag", "harassment", "", 0.505},
		{10004, 2, 11, 0, "Raw corporation bio\n<script>plain preview</script>", "markdown", "", "flag", "spam", "", 0.7},
		{10005, 3, 70122, 0, "Review alliance bio", "markdown", "<p><strong>Review alliance bio</strong> awaiting approval.</p>", "flag", "harassment", "", 0.3},
		{10006, 0, 20007, 0, "Review battle", "markdown", "<p>Review battle comment.</p>", "flag", "harassment", "", 0.9},
		{10007, 0, 20009, 0, "Review blog", "markdown", "<p>Review legacy blog comment.</p>", "flag", "spam", "", 0.3},
		{10008, 0, 20010, 0, "Review campaign", "markdown", "<p>Review campaign comment.</p>", "flag", "spam", "", 0.4},
		{10009, 1, 4242, 1, "Reviewed character", "markdown", "<p>Reviewed character bio.</p>", "pass", "", "<script>plain review note</script>", 0.02},
		{10010, 2, 10, 2, "Rejected corporation", "markdown", "<p>Rejected corporation bio.</p>", "flag", "spam", "Not appropriate for a public bio", 0.8},
		{10011, 0, 20006, 3, "Auto approved page", "markdown", "<p>Auto approved page comment.</p>", "pass", "", "", 0.01},
		{10012, 0, 20008, 4, "Auto rejected fit", "markdown", "<p>Auto rejected fit comment.</p>", "delete", "harassment", "", 0.99},
	} {
		if spec.Kind == 0 {
			if err := pool.QueryRow(ctx, `SELECT body_md,body_html FROM comments WHERE id=$1`, spec.Target).Scan(&spec.Body, &spec.HTML); err != nil {
				return err
			}
		}
		var html any
		if spec.HTML != "" {
			html = spec.HTML
		}
		if _, err := pool.Exec(ctx, `INSERT INTO moderation_queue(id,target_kind,target_id,body,body_format,rendered_html,character_id,character_name,corporation_id,corporation_name,status,submitted_at,reviewed_at,reviewed_by,review_notes,ai_action,ai_category,ai_max_score,ai_source)
		VALUES($1::bigint,$2,$3,$4,$5,$6,4343,'Fixture Reader',11,'Fixture Blue',$7::smallint,now()-$1::bigint*interval '1 minute',CASE WHEN $7::smallint>0 THEN now()-interval '2 hours' ELSE NULL END,CASE WHEN $7::smallint>0 THEN 4242 ELSE NULL END,NULLIF($8,''),$9,NULLIF($10,''),$11,'disposable-fixture')`, spec.ID, spec.Kind, spec.Target, spec.Body, spec.Format, html, spec.Status, spec.Notes, spec.Action, spec.Category, spec.Score); err != nil {
			return err
		}
	}
	_, err := pool.Exec(ctx, `SELECT setval('comments_id_seq',(SELECT max(id) FROM comments)); SELECT setval('moderation_queue_id_seq',(SELECT max(id) FROM moderation_queue))`)
	return err
}
