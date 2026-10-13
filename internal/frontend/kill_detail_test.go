package frontend

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

func TestFeaturedAttackerImagesRespectTypeCategory(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		category               *int64
		character, corporation *int64
		mode                   string
	}{
		{"ship", ptrKill[int64](6), ptrKill[int64](42), nil, "overlayrender"},
		{"structure", ptrKill[int64](65), nil, ptrKill[int64](77), "overlayrender"},
		{"corporation-owned structure weapon", ptrKill[int64](66), nil, ptrKill[int64](77), "icon"},
		{"ordinary module", ptrKill[int64](7), ptrKill[int64](42), nil, "icon"},
		{"environment NPC", nil, nil, nil, "icon"},
		{"older player payload", nil, ptrKill[int64](42), nil, "overlayrender"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			attacker := api.KillmailAttacker{ShipTypeID: ptrKill[int64](35928), ShipCategoryID: tc.category, CharacterID: tc.character, CorporationID: tc.corporation}
			if image := killAttackerImage(attacker, 512); image != "/images/types/35928/"+tc.mode+"?size=512" {
				t.Fatalf("incorrect attacker image: %s", image)
			}
		})
	}
}

func TestCorporationModuleFeaturedCardUsesIconWithoutPortrait(t *testing.T) {
	var body strings.Builder
	attacker := api.KillmailAttacker{ShipTypeID: ptrKill[int64](35928), ShipCategoryID: ptrKill[int64](66), CorporationID: ptrKill[int64](1639878825), CorporationName: "Initiative Holding", ShipName: "Standup Arcing Vorton Projector I"}
	if err := killFeaturedAttacker(attacker, "DMG", "damage", 100, true).Render(context.Background(), &body); err != nil {
		t.Fatal(err)
	}
	html := body.String()
	if !strings.Contains(html, `src="/images/types/35928/icon?size=512"`) || strings.Contains(html, "overlayrender") || strings.Contains(html, "/characters/") {
		t.Fatalf("module card requested a ship or character image: %s", html)
	}
}

type testKillmailReader struct {
	data  api.KillmailPageData
	found bool
}

func (r testKillmailReader) KillmailPage(context.Context, int64) (api.KillmailPageData, bool, error) {
	return r.data, r.found, nil
}

func ptrKill[T any](value T) *T { return &value }

func TestKillDetailGroupsAndWheel(t *testing.T) {
	data := api.KillmailPageData{Victim: api.KillmailVictim{ShipPrice: 100}, Items: []api.KillmailItem{
		{ItemIndex: 1, TypeID: 10, TypeName: "Launcher", FlagID: 27, Slot: "high", CategoryID: ptrKill[int64](7), QuantityDropped: 1, Price: 20},
		{ItemIndex: 2, TypeID: 10, TypeName: "Launcher", FlagID: 28, Slot: "high", CategoryID: ptrKill[int64](7), QuantityDestroyed: 1, Price: 20},
		{ItemIndex: 3, TypeID: 11, TypeName: "Ammo", FlagID: 27, Slot: "high", CategoryID: ptrKill[int64](8), QuantityDestroyed: 30, Price: 1},
		{ItemIndex: 4, TypeID: 20, TypeName: "Wrap", FlagID: 5, Slot: "cargo", IsContainer: true, QuantityDropped: 1},
		{ItemIndex: 5, TypeID: 21, TypeName: "Contained", Slot: "container_item", ParentIndex: ptrKill(4), QuantityDropped: 2, Price: 30},
		{ItemIndex: 6, TypeID: 21, TypeName: "Contained", Slot: "container_item", ParentIndex: ptrKill(4), QuantityDestroyed: 1, Price: 30},
	}}
	view := killBuildView(data)
	if len(view.Groups) != 2 || len(view.Groups[0].Rows) != 2 || view.Groups[0].Rows[0].QuantityDropped != 1 || view.Groups[0].Rows[0].QuantityDestroyed != 1 {
		t.Fatalf("unexpected merged high-slot rows: %+v", view.Groups)
	}
	if view.FittedValue != 70 || view.OtherValue != 90 || view.Groups[1].Value != 90 || len(view.Groups[1].Rows[0].Children) != 1 {
		t.Fatalf("unexpected totals or container children: fitted=%v other=%v cargo=%+v", view.FittedValue, view.OtherValue, view.Groups[1])
	}
	if view.Wheel[0].Item == nil || view.Wheel[0].Item.TypeID != 10 || view.Wheel[0].Ammo == nil || view.Wheel[0].Ammo.TypeID != 11 {
		t.Fatalf("ammo was not paired with its launcher: %+v", view.Wheel[0])
	}
}

func TestKillDetailRoute(t *testing.T) {
	data := api.KillmailPageData{KillmailID: 138563696, KillmailTime: time.Date(2026, 9, 19, 21, 53, 18, 0, time.UTC), Victim: api.KillmailVictim{CharacterID: ptrKill[int64](95898978), CharacterName: "Sarah Lafisques", ShipTypeID: ptrKill[int64](32878), ShipName: "Talwar", ShipPrice: 100}, SolarSystemID: 30002543, SolarSystemName: "Eystur", TotalValue: 200, AttackerCount: 1, Attackers: []api.KillmailAttacker{{CharacterName: "1connu malou", FinalBlow: true, DamageDone: 10}}, Items: []api.KillmailItem{{TypeID: 10, TypeName: "Launcher", Slot: "high", QuantityDestroyed: 1, Price: 20}}}
	handler := New(Options{Killmail: testKillmailReader{data: data, found: true}})
	for _, tt := range []struct {
		path     string
		status   int
		contains string
	}{
		{"/kill/138563696", 200, "Talwar destroyed in Eystur by 1 attacker"},
		{"/kill/123", 301, "/legacy/kill/123"},
		{"/kill/nope", 404, "Page not found"},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.path, nil))
		if w.Code != tt.status || !strings.Contains(w.Body.String()+w.Header().Get("Location"), tt.contains) {
			t.Errorf("%s: status %d body/location missing %q", tt.path, w.Code, tt.contains)
		}
	}
	for _, tt := range []struct {
		query, card, image string
		noImage            bool
	}{
		{"", `content="summary"`, `/images/types/32878/overlayrender?size=64`, false},
		{"?preview=big", `content="summary_large_image"`, `/images/types/32878/overlayrender?size=512`, false},
		{"?preview=fancy", `content="summary_large_image"`, `/images/killmail/138563696/social.png`, false},
		{"?preview=none", "", "", true},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/kill/138563696"+tt.query, nil))
		body := w.Body.String()
		if w.Code != 200 || tt.card != "" && !strings.Contains(body, tt.card) || tt.image != "" && !strings.Contains(body, tt.image) || tt.noImage && strings.Contains(body, `property="og:image"`) {
			t.Errorf("preview %s: status=%d, unexpected social metadata", tt.query, w.Code)
		}
	}
}
