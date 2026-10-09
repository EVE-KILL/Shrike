package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/png"

	"github.com/jackc/pgx/v5/pgxpool"
)

func seedAdminDomainFixtures(ctx context.Context, pool *pgxpool.Pool, storage *fixtureDomainAssets) error {
	for _, sql := range []string{
		`INSERT INTO custom_domains(id,subdomain,custom_hostname,user_id,site_name,site_description,entities,navbar_links,active,created_at) VALUES
		 (9101,'fixture-review','board-fixture.example.invalid',4343,'Fixture review board','A disposable review queue','[{"type":"character","id":4343,"name":"Fixture Reader"},{"type":"corporation","id":11,"name":"Fixture Blue"},{"type":"alliance","id":70122,"name":"Fixture Blue Alliance"}]','[{"label":"Fleet","url":"/battles"},{"label":"FAQ","url":"/faq"}]',true,now()-interval '4 days'),
		 (9102,'fixture-inactive',NULL,4242,'Fixture inactive board','', '[]','[]',false,now()-interval '3 days'),
		 (9103,'fixture-empty',NULL,44001,NULL,'', '[]','[]',true,now()-interval '2 days'),
		 (9104,'fixture-pending',NULL,4242,'Fixture pending board','', '[{"type":"character","id":4242,"name":"Fixture Owner"}]','[]',true,now()-interval '1 day')`,
		`SELECT setval('custom_domains_id_seq',(SELECT max(id) FROM custom_domains))`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			return err
		}
	}
	for i, entry := range []struct {
		Domain               int
		Type, Status, Reason string
	}{{9101, "logo", "pending", ""}, {9101, "banner", "pending", ""}, {9101, "background", "approved", ""}, {9101, "background", "rejected", "<script>plain rejection text</script>"}, {9101, "background", "pending", ""}, {9104, "logo", "pending", ""}} {
		id := 9201 + i
		w, h := 320, 96
		if entry.Type == "logo" {
			w, h = 96, 96
		}
		if entry.Type == "background" {
			w, h = 256, 144
		}
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				img.SetRGBA(x, y, color.RGBA{uint8(30 + x*130/w), uint8(40 + y*110/h), uint8(90 + i*20), 255})
			}
		}
		var body bytes.Buffer
		if err := png.Encode(&body, img); err != nil {
			return err
		}
		hash := sha256.Sum256(body.Bytes())
		key := fmt.Sprintf("domains/%d/%s_%d", entry.Domain, entry.Type, id)
		if err := storage.Put(ctx, key, body.Bytes(), "image/png"); err != nil {
			return err
		}
		_, err := pool.Exec(ctx, `INSERT INTO domain_assets(id,domain_id,type,status,storage_key,content_type,file_size,uploaded_by,reviewed_by,reviewed_at,reject_reason,file_hash,created_at) VALUES($1,$2,$3,$4,$5,'image/png',$6,4343,CASE WHEN $4<>'pending' THEN 4242 END,CASE WHEN $4<>'pending' THEN now()-interval '1 hour' END,NULLIF($7,''),$8,now()-$9*interval '1 hour')`, id, entry.Domain, entry.Type, entry.Status, key, body.Len(), entry.Reason, hex.EncodeToString(hash[:]), 6-i)
		if err != nil {
			return err
		}
	}
	_, err := pool.Exec(ctx, `SELECT setval('domain_assets_id_seq',(SELECT max(id) FROM domain_assets))`)
	return err
}
