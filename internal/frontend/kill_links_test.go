package frontend

import (
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/eve-kill/shrike/internal/api"
	"golang.org/x/net/html"
)

func TestKillIdentityNamesLinkToEntities(t *testing.T) {
	victim := api.KillmailVictim{
		CharacterID: ptrKill[int64](42), CharacterName: "Pilot",
		CorporationID: ptrKill[int64](77), CorporationName: "Corporation",
		AllianceID: ptrKill[int64](99), AllianceName: "Alliance",
	}
	view := killBuildView(api.KillmailPageData{Victim: victim})
	for name, component := range map[string]templ.Component{"desktop": killVictimCard(view), "mobile": killVictimCompact(view)} {
		t.Run(name, func(t *testing.T) {
			links := killNameLinks(t, component)
			for path, name := range map[string]string{"/character/42": "Pilot", "/corporation/77": "Corporation", "/alliance/99": "Alliance"} {
				if !strings.Contains(strings.Join(links[path], "|"), name) {
					t.Fatalf("%s name is not linked to %s: %v", name, path, links)
				}
			}
		})
	}
	for _, badge := range []string{"FB", "TD"} {
		attacker := api.KillmailAttacker{CharacterID: victim.CharacterID, CharacterName: victim.CharacterName, ShipTypeID: ptrKill[int64](587), ShipName: "Rifter", WeaponTypeID: ptrKill[int64](484), WeaponName: "Laser"}
		links := killNameLinks(t, killFeaturedAttacker(attacker, badge, "final", 1, true))
		for path, name := range map[string]string{"/character/42": "Pilot", "/item/587": "Rifter", "/item/484": "Laser"} {
			if !strings.Contains(strings.Join(links[path], "|"), name) {
				t.Fatalf("%s: %s name is not linked: %v", badge, name, links)
			}
		}
	}
}

func TestKillNamesWithoutCharactersUseCorporationOrItem(t *testing.T) {
	for _, tc := range []struct {
		attacker api.KillmailAttacker
		path     string
		name     string
	}{
		{api.KillmailAttacker{CorporationID: ptrKill[int64](77), CorporationName: "Owner", ShipTypeID: ptrKill[int64](35928)}, "/corporation/77", "Owner"},
		{api.KillmailAttacker{ShipTypeID: ptrKill[int64](35928), ShipName: "Environment"}, "/item/35928", "Environment"},
	} {
		links := killNameLinks(t, killFeaturedAttacker(tc.attacker, "FB", "final", 1, false))
		if !strings.Contains(strings.Join(links[tc.path], "|"), tc.name) {
			t.Fatalf("fallback name has no correct entity link: %v", links)
		}
		for path := range links {
			if strings.HasPrefix(path, "/character/") || strings.HasSuffix(path, "/0") {
				t.Fatalf("invented identity link: %s", path)
			}
		}
	}
}

func killNameLinks(t *testing.T, component templ.Component) map[string][]string {
	t.Helper()
	var body strings.Builder
	if err := component.Render(context.Background(), &body); err != nil {
		t.Fatal(err)
	}
	doc, err := html.Parse(strings.NewReader(body.String()))
	if err != nil {
		t.Fatal(err)
	}
	var text func(*html.Node) string
	text = func(node *html.Node) string {
		if node.Type == html.TextNode {
			return node.Data
		}
		var value strings.Builder
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			value.WriteString(text(child))
		}
		return value.String()
	}
	links := map[string][]string{}
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "a" {
			for _, attr := range node.Attr {
				if attr.Key == "href" {
					links[attr.Val] = append(links[attr.Val], strings.TrimSpace(text(node)))
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return links
}
