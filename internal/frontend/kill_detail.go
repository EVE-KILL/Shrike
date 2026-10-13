package frontend

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type killMergedItemRow struct {
	api.KillmailItem
	Children []api.KillmailItem
	Value    float64
}

type killItemGroup struct {
	Slot      string
	Label     string
	Rows      []killMergedItemRow
	Dropped   float64
	Destroyed float64
	Value     float64
}

type killWheelSlot struct {
	Group string
	Index int
	Item  *api.KillmailItem
	Ammo  *api.KillmailItem
}

type killDetailView struct {
	Data        api.KillmailPageData
	Groups      []killItemGroup
	Wheel       []killWheelSlot
	FinalBlow   *api.KillmailAttacker
	TopDamage   *api.KillmailAttacker
	FittedValue float64
	OtherValue  float64
	Tools       []killToolGroup
	Accent      string
	BorderStyle string
}

type killToolLink struct{ Label, Description, URL string }
type killToolGroup struct {
	Name, Icon string
	Links      []killToolLink
}

func killToolIcon(icon string) string { return assetURL(strings.TrimPrefix(icon, "/")) }

var killSlotOrder = []struct{ slot, label string }{
	{"high", "High Slots"}, {"mid", "Mid Slots"}, {"low", "Low Slots"},
	{"rig", "Rig Slots"}, {"subsystem", "Subsystems"}, {"service", "Service Slots"},
	{"drone", "Drone Bay"}, {"fighter", "Fighters"}, {"fighter_bay", "Fighter Bay"},
	{"cargo", "Cargo"}, {"fuel", "Fuel Bay"}, {"fleet", "Fleet Hangar"},
	{"ship_maintenance_bay", "Ship Maintenance Bay"}, {"specialized", "Specialized Hold"},
	{"implant", "Implants"}, {"booster", "Boosters"}, {"other", "Other"},
}

func killBuildView(data api.KillmailPageData) killDetailView {
	view := killDetailView{Data: data}
	for i := range data.Attackers {
		if data.Attackers[i].FinalBlow {
			view.FinalBlow = &data.Attackers[i]
			break
		}
	}
	if len(data.Attackers) > 0 {
		top := &data.Attackers[0]
		if view.FinalBlow == nil || top.CharacterID != nil && !killSameID(top.CharacterID, view.FinalBlow.CharacterID) || top.CharacterID == nil && !killSameID(top.CorporationID, view.FinalBlow.CorporationID) {
			view.TopDamage = top
		}
	}
	children := map[int]map[int64]*api.KillmailItem{}
	for _, item := range data.Items {
		if item.ParentIndex == nil || item.Slot != "container_item" {
			continue
		}
		bucket := children[*item.ParentIndex]
		if bucket == nil {
			bucket = map[int64]*api.KillmailItem{}
			children[*item.ParentIndex] = bucket
		}
		if prior := bucket[item.TypeID]; prior != nil {
			prior.QuantityDropped += item.QuantityDropped
			prior.QuantityDestroyed += item.QuantityDestroyed
		} else {
			copy := item
			bucket[item.TypeID] = &copy
		}
	}
	bySlot := map[string][]killMergedItemRow{}
	indices := map[string]map[int64]int{}
	for _, item := range data.Items {
		if item.Slot == "container_item" {
			continue
		}
		slot := item.Slot
		if slot == "" {
			slot = "other"
		}
		if indices[slot] == nil {
			indices[slot] = map[int64]int{}
		}
		if !item.IsContainer {
			if index, ok := indices[slot][item.TypeID]; ok {
				bySlot[slot][index].QuantityDropped += item.QuantityDropped
				bySlot[slot][index].QuantityDestroyed += item.QuantityDestroyed
				continue
			}
			indices[slot][item.TypeID] = len(bySlot[slot])
		}
		bySlot[slot] = append(bySlot[slot], killMergedItemRow{KillmailItem: item})
	}
	for _, def := range killSlotOrder {
		rows := bySlot[def.slot]
		if len(rows) == 0 {
			continue
		}
		group := killItemGroup{Slot: def.slot, Label: def.label, Rows: rows}
		for index := range group.Rows {
			row := &group.Rows[index]
			if row.IsContainer {
				for _, child := range data.Items {
					if child.ParentIndex == nil || *child.ParentIndex != row.ItemIndex {
						continue
					}
					// The map above owns merged quantities; keep source order by
					// visiting only the first occurrence of each type.
					if merged := children[row.ItemIndex][child.TypeID]; merged != nil {
						row.Children = append(row.Children, *merged)
						delete(children[row.ItemIndex], child.TypeID)
					}
				}
				for _, child := range row.Children {
					group.Dropped += child.Price * float64(child.QuantityDropped)
					group.Destroyed += child.Price * float64(child.QuantityDestroyed)
					row.Value += child.Price * float64(child.QuantityDropped+child.QuantityDestroyed)
				}
			} else {
				group.Dropped += row.Price * float64(row.QuantityDropped)
				group.Destroyed += row.Price * float64(row.QuantityDestroyed)
				row.Value = row.Price * float64(row.QuantityDropped+row.QuantityDestroyed)
			}
			group.Value += row.Value
		}
		switch def.slot {
		case "high", "mid", "low", "rig", "subsystem", "service":
			view.FittedValue += group.Value
		default:
			view.OtherValue += group.Value
		}
		view.Groups = append(view.Groups, group)
	}
	view.Wheel = killWheelSlots(data.Items)
	view.Tools = killToolGroups(view)
	var palette api.TopPalette
	if len(data.Victim.CorporationPalette) > 0 && json.Unmarshal(data.Victim.CorporationPalette, &palette) == nil {
		view.Accent = paletteAccent(&palette)
		if rgb, ok := parsePaletteColor(view.Accent); ok {
			view.BorderStyle = fmt.Sprintf("border-color:rgba(%d,%d,%d,0.4)", accentByte(rgb[0]), accentByte(rgb[1]), accentByte(rgb[2]))
		}
	}
	return view
}

func killToolGroups(view killDetailView) []killToolGroup {
	k := view.Data
	id := func(v *int64) string {
		if v == nil {
			return ""
		}
		return strconv.FormatInt(*v, 10)
	}
	ship := id(k.Victim.ShipTypeID)
	group := id(k.Victim.ShipGroupID)
	character := id(k.Victim.CharacterID)
	corporation := id(k.Victim.CorporationID)
	alliance := id(k.Victim.AllianceID)
	constellation := id(k.ConstellationID)
	region := id(k.RegionID)
	finalShip := ""
	if view.FinalBlow != nil {
		finalShip = id(view.FinalBlow.ShipTypeID)
	}
	topShip := ""
	if view.TopDamage != nil {
		topShip = id(view.TopDamage.ShipTypeID)
	}
	system := strconv.FormatInt(k.SolarSystemID, 10)
	killID := strconv.FormatInt(k.KillmailID, 10)
	mapName := func(s string) string { return url.QueryEscape(strings.ReplaceAll(s, " ", "_")) }
	groups := make([]killToolGroup, 0, 12)
	add := func(name, icon string, links ...killToolLink) {
		if len(links) > 0 {
			groups = append(groups, killToolGroup{Name: name, Icon: icon, Links: links})
		}
	}
	link := func(label, description, address string) killToolLink {
		return killToolLink{label, description, address}
	}
	dotlan := []killToolLink{}
	if k.SolarSystemName != "" {
		dotlan = append(dotlan, link("System", k.SolarSystemName, "https://evemaps.dotlan.net/system/"+url.PathEscape(k.SolarSystemName)))
	}
	if k.ConstellationName != "" {
		dotlan = append(dotlan, link("Constellation", k.ConstellationName, "https://evemaps.dotlan.net/map/"+url.PathEscape(k.RegionName)+"/"+url.PathEscape(k.ConstellationName)))
	}
	if k.RegionName != "" {
		dotlan = append(dotlan, link("Region", k.RegionName, "https://evemaps.dotlan.net/region/"+url.PathEscape(k.RegionName)))
	}
	add("DOTLAN", "/remotes/dotlan.png", dotlan...)
	if k.RegionName != "" {
		add("EVEEye", "/remotes/eveeye.svg", link("Region", k.RegionName, "https://eveeye.com/?m="+url.QueryEscape(k.RegionName)))
	}
	mission := []killToolLink{link("System", k.SolarSystemName, "https://evemissioneer.com/system/"+system)}
	if region != "" {
		mission = append(mission, link("Region", k.RegionName, "https://evemissioneer.com/r/"+region))
	}
	add("Missioneer", "/remotes/evemissioneer.png", mission...)
	if ship != "" {
		add("EveShip.fit", "/remotes/eveship-fit.png", link("Fitting", k.Victim.ShipName, "https://eveship.fit/fit/"+killID))
	}
	everef := []killToolLink{}
	if group != "" {
		everef = append(everef, link("Ship Group", k.Victim.ShipGroupName, "https://everef.net/group/"+group))
	}
	if ship != "" {
		everef = append(everef, link("Ship", k.Victim.ShipName, "https://everef.net/type/"+ship))
	}
	add("EVERef", "/remotes/everef.png", everef...)
	jita := []killToolLink{link("System", k.SolarSystemName, "https://www.jita.space/system/"+system)}
	if constellation != "" {
		jita = append(jita, link("Constellation", k.ConstellationName, "https://www.jita.space/constellation/"+constellation))
	}
	if region != "" {
		jita = append(jita, link("Region", k.RegionName, "https://www.jita.space/region/"+region))
	}
	add("Jita.Space", "/remotes/jita-space.png", jita...)
	identities := func(base string) []killToolLink {
		result := []killToolLink{}
		if character != "" {
			result = append(result, link("Character", k.Victim.CharacterName, base+"/character/"+character))
		}
		if corporation != "" {
			result = append(result, link("Corporation", k.Victim.CorporationName, base+"/corporation/"+corporation))
		}
		if alliance != "" {
			result = append(result, link("Alliance", k.Victim.AllianceName, base+"/alliance/"+alliance))
		}
		return result
	}
	add("EVEWho", "/remotes/evewho.png", identities("https://evewho.com")...)
	capsuleers := identities("https://capsuleers.app")
	if ship != "" {
		capsuleers = append(capsuleers, link("Ship", k.Victim.ShipName, "https://capsuleers.app/items/"+ship))
	}
	if k.RegionName != "" {
		capsuleers = append(capsuleers, link("Region", k.RegionName, "https://capsuleers.app/tools/map/"+mapName(k.RegionName)))
	}
	if k.RegionName != "" && k.SolarSystemName != "" {
		capsuleers = append(capsuleers, link("System", k.SolarSystemName, "https://capsuleers.app/tools/map/"+mapName(k.RegionName)+"?system="+mapName(k.SolarSystemName)))
	}
	add("Capsuleers.app", "", capsuleers...)
	zkb := []killToolLink{link("Killmail", "#"+killID, "https://zkillboard.com/kill/"+killID+"/"), link("System", k.SolarSystemName, "https://zkillboard.com/system/"+system+"/")}
	if constellation != "" {
		zkb = append(zkb, link("Constellation", k.ConstellationName, "https://zkillboard.com/constellation/"+constellation+"/"))
	}
	if region != "" {
		zkb = append(zkb, link("Region", k.RegionName, "https://zkillboard.com/region/"+region+"/"))
	}
	if group != "" {
		zkb = append(zkb, link("Ship Group", k.Victim.ShipGroupName, "https://zkillboard.com/group/"+group+"/"))
	}
	if ship != "" {
		zkb = append(zkb, link("Ship", k.Victim.ShipName, "https://zkillboard.com/ship/"+ship+"/"))
	}
	for _, identity := range identities("https://zkillboard.com") {
		zkb = append(zkb, killToolLink{identity.Label, identity.Description, identity.URL + "/"})
	}
	add("zKillboard", "/remotes/zkillboard.png", zkb...)
	kb := []killToolLink{link("Killmail", "#"+killID, "https://kb.evetools.org/kill/"+killID)}
	if ship != "" {
		kb = append(kb, link("Victim Ship", k.Victim.ShipName, "https://kb.evetools.org/ship/"+ship))
	}
	if finalShip != "" {
		kb = append(kb, link("Final Blow Ship", view.FinalBlow.ShipName, "https://kb.evetools.org/ship/"+finalShip))
	}
	if topShip != "" {
		kb = append(kb, link("Top Damage Ship", view.TopDamage.ShipName, "https://kb.evetools.org/ship/"+topShip))
	}
	kb = append(kb, identities("https://kb.evetools.org")...)
	kb = append(kb, link("System", k.SolarSystemName, "https://kb.evetools.org/system/"+system))
	if region != "" {
		kb = append(kb, link("Region", k.RegionName, "https://kb.evetools.org/region/"+region))
	}
	add("kb.evetools", "/remotes/evetools.png", kb...)
	add("SocketKill", "/remotes/socketkill.png", link("Killmail", "#"+killID, "https://socketkill.com/kill/"+killID))
	add("ESI", "", link("Raw Killmail", "JSON from CCP", "https://esi.evetech.net/latest/killmails/"+killID+"/"+url.PathEscape(k.KillmailHash)+"/"))
	return groups
}

func killSameID(a, b *int64) bool { return a != nil && b != nil && *a == *b }

func killWheelSlots(items []api.KillmailItem) []killWheelSlot {
	groups := []struct {
		name         string
		first, count int
	}{{"top", 27, 8}, {"right", 19, 8}, {"bottom", 11, 8}, {"left", 92, 3}, {"sub", 125, 4}}
	result := make([]killWheelSlot, 0, 31)
	for _, group := range groups {
		for index := 0; index < group.count; index++ {
			slot := killWheelSlot{Group: group.name, Index: index}
			for itemIndex := range items {
				item := &items[itemIndex]
				if item.ParentIndex != nil || item.FlagID != group.first+index {
					continue
				}
				if item.CategoryID != nil && *item.CategoryID == 8 {
					slot.Ammo = item
				} else {
					slot.Item = item
				}
			}
			result = append(result, slot)
		}
	}
	return result
}

func killWheelPosition(group string, index int) string {
	angle := float64(index) * 10.5
	switch group {
	case "top":
		angle -= 125
	case "right":
		angle -= 37
	case "bottom":
		angle += 54
	case "left":
		angle += 196
	case "sub":
		angle += 140
	}
	radians := angle * math.Pi / 180
	return fmt.Sprintf("left:%.3f%%;top:%.3f%%", math.Round((50+42*math.Cos(radians))*1000)/1000, math.Round((50+42*math.Sin(radians))*1000)/1000)
}

func killID(id *int64) int64 {
	if id == nil {
		return 0
	}
	return *id
}
func killNumber(n int64) string { return fullNumber(float64(n)) }
func killRelative(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}
func killTimestamp(t time.Time) string { return t.UTC().Format("2 Jan 2006, 15:04:05 EVE") }
func killSecurity(s *float64) string {
	if s == nil {
		return ""
	}
	return fmt.Sprintf("%.1f", *s)
}
func killSecClass(s *float64) string {
	if s == nil {
		return ""
	}
	if *s >= .5 {
		return "high"
	}
	if *s > 0 {
		return "low"
	}
	return "null"
}
func killWho(data api.KillmailPageData) string {
	if data.Victim.CharacterName != "" {
		return data.Victim.CharacterName
	}
	if data.Victim.CorporationName != "" {
		return data.Victim.CorporationName
	}
	return "Unknown"
}
func killShip(data api.KillmailPageData) string {
	if data.Victim.ShipName != "" {
		return data.Victim.ShipName
	}
	return "Unknown"
}
func killHeadline(view killDetailView) string {
	where := view.Data.SolarSystemName
	if where == "" {
		where = "unknown space"
	}
	if view.Data.IsSolo {
		name := "one attacker"
		if view.FinalBlow != nil && view.FinalBlow.CharacterName != "" {
			name = view.FinalBlow.CharacterName
		}
		return fmt.Sprintf("%s destroyed in %s solo by %s", killShip(view.Data), where, name)
	}
	n := view.Data.AttackerCount
	suffix := "s"
	if n == 1 {
		suffix = ""
	}
	return fmt.Sprintf("%s destroyed in %s by %d attacker%s", killShip(view.Data), where, n, suffix)
}

func killDescription(view killDetailView) string {
	k := view.Data
	v := k.Victim
	who := killWho(k)
	ship := v.ShipName
	if ship == "" {
		ship = "ship"
	}
	affiliation := []string{}
	if v.CharacterName != "" && v.CorporationName != "" {
		affiliation = append(affiliation, v.CorporationName)
	}
	if v.AllianceName != "" {
		affiliation = append(affiliation, v.AllianceName)
	}
	if len(affiliation) > 0 {
		who += " (" + strings.Join(affiliation, " / ") + ")"
	}
	article := "a"
	if strings.ContainsRune("aeiouAEIOU", rune(ship[0])) {
		article = "an"
	}
	where := k.SolarSystemName
	if where == "" {
		where = "Unknown"
	}
	loc := []string{}
	if k.SolarSystemSecurity != nil {
		loc = append(loc, killSecurity(k.SolarSystemSecurity))
	}
	if k.RegionName != "" {
		loc = append(loc, k.RegionName)
	}
	if len(loc) > 0 {
		where += " (" + strings.Join(loc, ", ") + ")"
	}
	count := len(k.Attackers)
	suffix := "s"
	if count == 1 {
		suffix = ""
	}
	desc := fmt.Sprintf("%s lost %s %s in %s worth %s ISK. %d attacker%s", who, article, ship, where, compactNumber(k.TotalValue), count, suffix)
	if view.FinalBlow != nil {
		fb := view.FinalBlow.CharacterName
		if fb == "" {
			fb = view.FinalBlow.CorporationName
		}
		if fb != "" {
			desc += ", final blow by " + fb
		}
	}
	flags := []string{}
	if k.IsSolo {
		flags = append(flags, "Solo")
	}
	if k.IsNPC {
		flags = append(flags, "NPC")
	}
	if len(flags) > 0 {
		desc += " (" + strings.Join(flags, ", ") + ")"
	}
	return desc + "."
}
func killTypeIcon(id int64, size int) string {
	return fmt.Sprintf("/images/types/%d/icon?size=%d", id, size)
}
func killTypeRender(id int64, size int) string {
	return fmt.Sprintf("/images/types/%d/overlayrender?size=%d", id, size)
}
func killAttackerImage(attacker api.KillmailAttacker, size int) string {
	if attacker.ShipTypeID == nil || *attacker.ShipTypeID <= 0 {
		return ""
	}
	// Environment attackers and non-hull types (including corporation-owned
	// structure weapons) have item icons rather than ship overlay renders.
	if killID(attacker.CharacterID) == 0 && killID(attacker.CorporationID) == 0 ||
		attacker.ShipCategoryID != nil && *attacker.ShipCategoryID != 6 && *attacker.ShipCategoryID != 65 {
		return killTypeIcon(*attacker.ShipTypeID, size)
	}
	return killTypeRender(*attacker.ShipTypeID, size)
}
func killBoolClass(v bool, t, f string) string {
	if v {
		return t
	}
	return f
}
func killPercent(value, total int64) string {
	if total == 0 {
		return "0.0"
	}
	return fmt.Sprintf("%.1f", float64(value)*100/float64(total))
}
func killRemaining(count int) string { return strconv.Itoa(max(0, count-10)) }
func killDistance(value float64) string {
	const au = 149597870700.
	if value >= au/10 {
		return fmt.Sprintf("%.2f AU", value/au)
	}
	return killNumber(int64(math.Round(value/1000))) + " km"
}
func killDestroyedPercent(data api.KillmailPageData) float64 {
	total := data.DestroyedValue + data.DroppedValue
	if total <= 0 {
		return 100
	}
	return data.DestroyedValue * 100 / total
}
func killHasSubslots(slots []killWheelSlot) bool {
	for _, slot := range slots {
		if slot.Group == "sub" && slot.Item != nil {
			return true
		}
	}
	return false
}
func killWheelAmmoName(slot killWheelSlot) string {
	if slot.Ammo == nil {
		return ""
	}
	return slot.Ammo.TypeName
}
func killItemStatus(item api.KillmailItem) string {
	if item.IsContainer {
		return "container"
	}
	if item.QuantityDropped > 0 && item.QuantityDestroyed > 0 {
		return "both"
	}
	if item.QuantityDropped > 0 {
		return "dropped"
	}
	return "destroyed"
}
func killAttackerName(attacker api.KillmailAttacker) string {
	if attacker.CharacterName != "" {
		return attacker.CharacterName
	}
	if attacker.CorporationName != "" {
		return attacker.CorporationName
	}
	if attacker.ShipName != "" {
		return attacker.ShipName
	}
	return "NPC"
}
func killDamageWidth(value, top int64) float64 {
	if top <= 0 {
		return 2
	}
	return math.Max(2, float64(value)*100/float64(top))
}

func (s *site) loadKillmailPage(r *http.Request) (Page, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return Page{}, ErrNotFound
	}
	data, found, err := s.options.Killmail.KillmailPage(r.Context(), id)
	if err != nil {
		return Page{}, err
	}
	if !found {
		return Page{}, ErrNotFound
	}
	view := killBuildView(data)
	title := fmt.Sprintf("%s | %s | %s ISK", killShip(data), killWho(data), compactNumber(data.TotalValue))
	desc := killDescription(view)
	image := ""
	imageWidth, imageHeight := 0, 0
	if data.Victim.ShipTypeID != nil {
		image = killTypeRender(*data.Victim.ShipTypeID, 64)
		imageWidth, imageHeight = 64, 64
	}
	preview := r.URL.Query().Get("preview")
	card := "summary"
	kind := "website"
	switch preview {
	case "big":
		if data.Victim.ShipTypeID != nil {
			image = killTypeRender(*data.Victim.ShipTypeID, 512)
			imageWidth, imageHeight = 512, 512
		}
		card, kind = "summary_large_image", "article"
	case "fancy":
		image = fmt.Sprintf("/images/killmail/%d/social.png", id)
		imageWidth, imageHeight = 550, 200
		card, kind = "summary_large_image", "article"
	case "none":
		image = ""
		kind = "article"
		card = ""
	}
	structured := []any{map[string]any{"@context": "https://schema.org", "@type": "BreadcrumbList", "itemListElement": []any{map[string]any{"@type": "ListItem", "position": 1, "name": "Home", "item": canonicalOrigin + "/"}, map[string]any{"@type": "ListItem", "position": 2, "name": "Kills", "item": canonicalOrigin + "/kills/latest"}, map[string]any{"@type": "ListItem", "position": 3, "name": title, "item": canonicalOrigin + r.URL.Path}}}, map[string]any{"@context": "https://schema.org", "@type": "Article", "headline": title, "description": desc, "datePublished": data.KillmailTime.UTC().Format(time.RFC3339), "author": map[string]string{"@type": "Organization", "name": "EVE-KILL"}}}
	return Page{Metadata: Metadata{Title: title, OGTitle: title + " | " + data.SolarSystemName, TwitterTitle: title + " — EVE-KILL", TwitterCard: card, ThemeColor: view.Accent, Description: desc, CanonicalPath: r.URL.Path, Image: image, ImageAlt: killShip(data), ImageWidth: imageWidth, ImageHeight: imageHeight, NoSocialImage: preview == "none", Kind: kind, Indexable: true, CommentMedia: true, CommentEditor: true, StructuredData: structured, Stylesheets: []string{"kill-detail.css", "comment-thread.css"}, ModuleScripts: []string{"kill-detail.js", "comment-thread.js"}}, Content: killDetailPage(view), SharedCacheTTL: 5 * time.Minute}, nil
}
