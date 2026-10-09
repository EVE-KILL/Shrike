package frontend

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/api"
	"golang.org/x/sync/errgroup"
)

type itemPageView struct {
	ID          int64
	Tab         string
	Data        api.UniverseItemData
	Description string
	Recent      api.KilllistPage
	Kills       api.KilllistPage
	Market      itemMarketView
	Fits        api.UniverseFitFamilies
	FitQuery    url.Values
	FitSort     string
}

type itemMarketView struct {
	RegionID int32
	Days     int
	Security string
	Orders   api.UniverseMarketOrders
	History  api.UniverseMarketHistory
}

type itemCrumb struct {
	Name, Path string
}

func loadItemPage(ctx context.Context, reader UniverseReader, id int64, tab string, query url.Values) (itemPageView, error) {
	if id < 1 || id > 2_147_483_647 {
		return itemPageView{}, ErrNotFound
	}
	if tab == "" {
		tab = "dashboard"
	}
	data, found, err := reader.UniverseItem(ctx, id)
	if err != nil {
		return itemPageView{}, err
	}
	if !found || data.Item.Name == "" {
		return itemPageView{}, ErrNotFound
	}
	switch tab {
	case "dashboard", "kills":
	case "market":
		if len(data.MarketBreadcrumb) == 0 {
			return itemPageView{}, ErrNotFound
		}
	case "fittings":
		if !data.Item.IsShip {
			return itemPageView{}, ErrNotFound
		}
	default:
		return itemPageView{}, ErrNotFound
	}
	view := itemPageView{ID: id, Tab: tab, Data: data, Description: entityBioHTML("", data.Item.Description)}
	switch tab {
	case "dashboard":
		view.Recent, err = reader.UniverseItemKills(ctx, id, data.Item.IsShip, 10, nil)
	case "kills":
		view.Kills, err = reader.UniverseItemKills(ctx, id, data.Item.IsShip, 50, nil)
	case "market":
		view.Market, err = loadItemMarket(ctx, reader, id, query)
	case "fittings":
		fitView, fitErr := loadItemFits(ctx, reader, id, query)
		view.Fits, view.FitQuery, view.FitSort, err = fitView.Fits, fitView.FitQuery, fitView.FitSort, fitErr
	}
	if err != nil {
		return itemPageView{}, err
	}
	return view, nil
}

func loadItemMarket(ctx context.Context, reader UniverseReader, id int64, query url.Values) (itemMarketView, error) {
	view := itemMarketView{Days: 30, Security: "high,low,null"}
	if raw := query.Get("region"); raw != "" {
		region, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || region < 0 {
			return view, ErrNotFound
		}
		view.RegionID = int32(region)
	}
	if raw := query.Get("days"); raw != "" {
		days, err := strconv.Atoi(raw)
		if err != nil || days != 30 && days != 90 {
			return view, ErrNotFound
		}
		view.Days = days
	}
	if raw := query.Get("security"); raw != "" {
		seen := map[string]bool{}
		for _, value := range strings.Split(raw, ",") {
			if value != "high" && value != "low" && value != "null" || seen[value] {
				return view, ErrNotFound
			}
			seen[value] = true
		}
		view.Security = strings.Join(func() []string {
			ordered := make([]string, 0, 3)
			for _, value := range []string{"high", "low", "null"} {
				if seen[value] {
					ordered = append(ordered, value)
				}
			}
			return ordered
		}(), ",")
	}
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		orders, err := reader.UniverseMarketOrders(groupCtx, id, view.RegionID, view.Security)
		view.Orders = orders
		return err
	})
	group.Go(func() error {
		history, err := reader.UniverseMarketHistory(groupCtx, id, view.RegionID, view.Days)
		view.History = history
		return err
	})
	return view, group.Wait()
}

func itemImage(view itemPageView, size int) string {
	variant := "icon"
	if view.Data.Item.IsShip {
		variant = "render"
	}
	return fmt.Sprintf("/images/types/%d/%s?size=%d", view.ID, variant, size)
}

func itemRolePhrase(view itemPageView) string {
	item := view.Data.Item
	if item.GroupName == "" {
		return ""
	}
	meta := ""
	if item.MetaGroupName != "" {
		meta = item.MetaGroupName + " "
	}
	if item.IsShip {
		return meta + item.GroupName + " — EVE Online ship"
	}
	category := ""
	if item.CategoryName != "" && item.CategoryName != item.GroupName {
		category = " (" + item.CategoryName + ")"
	}
	return meta + item.GroupName + category + " — EVE Online item"
}

func itemMetaGroupClass(name string) string {
	name = strings.ToLower(name)
	switch {
	case strings.Contains(name, "tech ii"):
		return "tech-two"
	case strings.Contains(name, "tech iii"):
		return "tech-three"
	case strings.Contains(name, "faction"):
		return "faction"
	case strings.Contains(name, "deadspace"):
		return "deadspace"
	case strings.Contains(name, "officer"):
		return "officer"
	default:
		return ""
	}
}

var itemTagPattern = regexp.MustCompile(`<[^>]+>`)

func itemDescriptionExcerpt(value string) string {
	value = strings.Join(strings.Fields(itemTagPattern.ReplaceAllString(value, "")), " ")
	runes := []rune(value)
	if len(runes) <= 180 {
		return value
	}
	return strings.TrimSpace(string(runes[:177])) + "…"
}

func itemMetadata(view itemPageView) Metadata {
	path := fmt.Sprintf("/item/%d", view.ID)
	title := view.Data.Item.Name
	if view.Tab != "dashboard" {
		label := strings.ToUpper(view.Tab[:1]) + view.Tab[1:]
		title += " (" + label + ")"
		path += "/" + view.Tab
	}
	description := fmt.Sprintf("%s — %s", view.Data.Item.Name, itemRolePhrase(view))
	if summary := view.Data.Pricing.Summary; summary != nil && summary.Latest > 0 {
		description += " · Jita " + compactNumber(summary.Latest) + " ISK"
	}
	if excerpt := itemDescriptionExcerpt(view.Data.Item.Description); excerpt != "" {
		description += " · " + excerpt
	}
	description += " · Killmails, fittings, and market data on EVE-KILL."
	crumbs := []any{map[string]any{"@type": "ListItem", "position": 1, "name": "Home", "item": canonicalOrigin + "/"},
		map[string]any{"@type": "ListItem", "position": 2, "name": view.Data.Item.GroupName, "item": canonicalOrigin + fmt.Sprintf("/group/%d", view.Data.Item.GroupID)},
		map[string]any{"@type": "ListItem", "position": 3, "name": view.Data.Item.Name, "item": canonicalOrigin + fmt.Sprintf("/item/%d", view.ID)}}
	product := map[string]any{"@context": "https://schema.org", "@type": "Product", "name": view.Data.Item.Name,
		"url": canonicalOrigin + fmt.Sprintf("/item/%d", view.ID), "image": canonicalOrigin + itemImage(view, 512),
		"category": view.Data.Item.GroupName, "description": itemDescriptionExcerpt(view.Data.Item.Description)}
	if summary := view.Data.Pricing.Summary; summary != nil && summary.Latest > 0 {
		product["offers"] = map[string]any{"@type": "Offer", "price": summary.Latest, "priceCurrency": "ISK",
			"priceValidUntil": time.Now().UTC().Add(24 * time.Hour).Format("2006-01-02"), "availability": "https://schema.org/InStock"}
	}
	return Metadata{Title: title, Description: description, CanonicalPath: path, Image: itemImage(view, 256),
		ImageAlt: view.Data.Item.Name, Indexable: true, StructuredData: []any{
			map[string]any{"@context": "https://schema.org", "@type": "BreadcrumbList", "itemListElement": crumbs}, product,
		}}
}

func itemAttribute(view itemPageView, id int64) (float64, bool) {
	for _, group := range view.Data.ShipAttributes {
		for _, attribute := range group {
			if attribute.ID == id {
				return attribute.Value, true
			}
		}
	}
	return 0, false
}

func itemRawHP(view itemPageView) float64 {
	var total float64
	for _, id := range []int64{263, 265, 9} {
		value, _ := itemAttribute(view, id)
		total += value
	}
	return total
}

func itemRomanLevel(level int) string {
	roman := []string{"", "I", "II", "III", "IV", "V"}
	if level >= 1 && level <= 5 {
		return roman[level]
	}
	return strconv.Itoa(level)
}

func itemQuantity(value float64) string {
	if value == math.Trunc(value) {
		return fullNumber(value)
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", value), "0"), ".")
}

func itemMarketLeaf(view itemPageView) string {
	crumbs := view.Data.MarketBreadcrumb
	if len(crumbs) >= 3 {
		return crumbs[len(crumbs)-1].Name
	}
	return ""
}

func itemMarketTrail(view itemPageView) []api.UniverseMarketCrumb {
	crumbs := view.Data.MarketBreadcrumb
	if len(crumbs) > 2 {
		return crumbs[:2]
	}
	return crumbs
}

func itemBreadcrumb(view itemPageView) []itemCrumb {
	result := []itemCrumb{{Name: "Market", Path: "/market"}}
	path := ""
	for _, crumb := range itemMarketTrail(view) {
		if crumb.Slug == "" {
			continue
		}
		if path != "" {
			path += "/"
		}
		path += crumb.Slug
		result = append(result, itemCrumb{Name: crumb.Name, Path: "/market/" + path})
	}
	if view.Data.Item.GroupID > 0 {
		result = append(result, itemCrumb{Name: view.Data.Item.GroupName, Path: idPath("group", view.Data.Item.GroupID)})
	}
	return append(result, itemCrumb{Name: view.Data.Item.Name})
}

func itemAttributeGroupTitle(key string) string {
	switch key {
	case "defense":
		return "Defense"
	case "fitting":
		return "Fitting"
	case "navigation":
		return "Navigation"
	case "capacitor":
		return "Capacitor"
	case "targeting":
		return "Targeting"
	case "drones":
		return "Drones"
	default:
		return key
	}
}

func itemAttributeValue(attribute api.UniverseAttribute) string {
	id, value := attribute.ID, attribute.Value
	switch id {
	case 271, 272, 273, 274, 267, 268, 269, 270, 113, 110, 109, 111:
		return fmt.Sprintf("%.1f%%", (1-value)*100)
	case 73, 55, 479:
		return fmt.Sprintf("%.1fs", value/1000)
	case 564:
		if value >= 1000 {
			return fmt.Sprintf("%.1f km", value/1000)
		}
		return fmt.Sprintf("%.0f m", value)
	case 6, 482, 600:
		return fmt.Sprintf("%.1f GJ", value)
	case 30, 11:
		return fmt.Sprintf("%.0f MW", value)
	case 50, 48:
		return fmt.Sprintf("%.0f tf", value)
	case 37:
		return fmt.Sprintf("%.0f m/s", value)
	case 552:
		return fmt.Sprintf("%.0f m", value)
	case 70:
		return fmt.Sprintf("%.3f", value)
	case 161:
		return fmt.Sprintf("%.1f AU/s", value)
	case 4:
		return fullNumber(value/1000) + " kg"
	case 38, 284:
		return fullNumber(value) + " m³"
	case 283:
		return fmt.Sprintf("%.0f Mbit/s", value)
	case 212:
		return fmt.Sprintf("%.2fx", value)
	case 20, 554, 68:
		if value >= -2 && value <= 2 {
			return fmt.Sprintf("%.1f%%", (value-1)*100)
		}
		return fmt.Sprintf("%.1f%%", value)
	default:
		if value == math.Trunc(value) {
			return fullNumber(value)
		}
		return fmt.Sprintf("%.2f", value)
	}
}

func itemAttributeColor(attribute api.UniverseAttribute) string {
	switch attribute.ID {
	case 271, 272, 273, 274, 267, 268, 269, 270, 113, 110, 109, 111:
		pct := (1 - attribute.Value) * 100
		if pct >= 60 {
			return "high-resist"
		}
		if pct >= 30 {
			return "medium-resist"
		}
		return "low-resist"
	default:
		return ""
	}
}

func itemNamedAttributes(rows []api.UniverseAttribute) []api.UniverseAttribute {
	allowed := map[int64]string{9: "Structure HP", 263: "Shield HP", 265: "Armor HP", 37: "Max Velocity", 552: "Signature Radius", 4: "Mass",
		38: "Capacity", 283: "Drone Bandwidth", 284: "Drone Capacity", 482: "Capacitor Capacity", 55: "Capacitor Recharge",
		6: "Activation Cost", 73: "Cycle Time", 30: "Powergrid Usage", 50: "CPU Usage", 1153: "Calibration Cost",
		114: "EM Damage", 118: "Thermal Damage", 117: "Kinetic Damage", 116: "Explosive Damage", 212: "Damage Multiplier",
		20: "Velocity Modifier", 554: "Signature Radius Modifier", 68: "Armor HP Bonus", 128: "Charge Size", 604: "Charge Group",
		271: "Shield EM Resist", 272: "Shield Thermal Resist", 273: "Shield Kinetic Resist", 274: "Shield Explosive Resist",
		267: "Armor EM Resist", 268: "Armor Thermal Resist", 269: "Armor Kinetic Resist", 270: "Armor Explosive Resist",
		11: "Powergrid Output", 48: "CPU Output", 12: "High Slots", 13: "Med Slots", 14: "Low Slots", 1137: "Rig Slots",
		1132: "Calibration Output", 1154: "Launcher Hardpoints", 102: "Turret Hardpoints", 161: "Warp Speed", 70: "Inertia Modifier",
		564: "Max Target Range", 192: "Max Locked Targets", 76: "Radar Strength", 77: "Ladar Strength", 78: "Magnetometric Strength",
		79: "Gravimetric Strength", 479: "Shield Recharge Time"}
	result := make([]api.UniverseAttribute, 0, len(rows))
	for _, row := range rows {
		if name, ok := allowed[row.ID]; ok {
			row.Name = name
			result = append(result, row)
		}
	}
	return result
}

func (s *site) registerItemPages(mux *http.ServeMux) {
	load := func(r *http.Request) (Page, error) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			return Page{}, ErrNotFound
		}
		view, err := loadItemPage(r.Context(), s.options.Universe, id, r.PathValue("tab"), r.URL.Query())
		if err != nil {
			return Page{}, err
		}
		return Page{Metadata: itemMetadata(view), Content: itemPage(view), SharedCacheTTL: 2 * time.Minute}, nil
	}
	mux.Handle("GET /item/{id}", s.pageHandler(load))
	mux.Handle("GET /item/{id}/{tab}", s.pageHandler(load))
	mux.HandleFunc("GET /_fragments/item/tab/{id}/{tab}", s.itemTabFragment)
	mux.HandleFunc("GET /_fragments/item/kills/{id}", s.itemKillsFragment)
	mux.HandleFunc("GET /_fragments/item/market/{id}", s.itemMarketFragment)
	mux.HandleFunc("GET /_fragments/item/fittings/{id}", s.itemFitsFragment)
	mux.HandleFunc("GET /_fragments/item/fit-meta/{id}", s.itemFitMetaFragment)
	mux.HandleFunc("GET /_fragments/item/fit-distributions/{id}", s.itemFitDistributionsFragment)
}

func (s *site) itemTabFragment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	view, err := loadItemPage(r.Context(), s.options.Universe, id, r.PathValue("tab"), r.URL.Query())
	if err != nil {
		if err == ErrNotFound {
			http.NotFound(w, r)
		} else {
			http.Error(w, "Could not load item", http.StatusServiceUnavailable)
		}
		return
	}
	s.renderFragment(w, r, itemMain(view))
}

func (s *site) itemMarketFragment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 || id > 2_147_483_647 {
		http.NotFound(w, r)
		return
	}
	market, err := loadItemMarket(r.Context(), s.options.Universe, id, r.URL.Query())
	if err != nil {
		if err == ErrNotFound {
			http.Error(w, "Invalid market filters", http.StatusBadRequest)
		} else {
			http.Error(w, "Could not load market", http.StatusServiceUnavailable)
		}
		return
	}
	s.renderFragment(w, r, itemMarketPanel(itemPageView{ID: id, Tab: "market", Market: market}))
}

func (s *site) itemKillsFragment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 || id > 2_147_483_647 {
		http.NotFound(w, r)
		return
	}
	limit, err := fragmentInt(r.URL.Query().Get("limit"), 50, 10, 100)
	if err != nil {
		http.Error(w, "Invalid limit", http.StatusBadRequest)
		return
	}
	var after *int64
	if raw := r.URL.Query().Get("after"); raw != "" {
		value, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil || value < 1 {
			http.Error(w, "Invalid cursor", http.StatusBadRequest)
			return
		}
		after = &value
	}
	page, err := fragmentInt(r.URL.Query().Get("page"), 1, 1, 1_000_000)
	if err != nil || page > 1 && after == nil {
		http.Error(w, "Invalid page", http.StatusBadRequest)
		return
	}
	data, found, err := s.options.Universe.UniverseItem(r.Context(), id)
	if err != nil {
		http.Error(w, "Could not load item", http.StatusServiceUnavailable)
		return
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	kills, err := s.options.Universe.UniverseItemKills(r.Context(), id, data.Item.IsShip, limit, after)
	if err != nil {
		http.Error(w, "Could not load kills", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, killList(kills, killListOptions{Kind: "item", Page: page, Limit: limit}))
}
