package frontend

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type marketPageView struct {
	Tree       []api.UniverseMarketTreeNode
	Breadcrumb []api.UniverseMarketTreeNode
	Active     *api.UniverseMarketTreeNode
	Items      []api.UniverseMarketGroupItem
	Path       string
	Redirect   string
}

func loadMarketPage(ctx context.Context, reader UniverseReader, rawPath string) (marketPageView, error) {
	tree, err := reader.UniverseMarketTree(ctx)
	if err != nil {
		return marketPageView{}, err
	}
	view := marketPageView{Tree: tree}
	if rawPath == "" {
		return view, nil
	}
	segments := strings.Split(strings.Trim(rawPath, "/"), "/")
	if len(segments) > 12 {
		return marketPageView{}, ErrNotFound
	}
	current := tree
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return marketPageView{}, ErrNotFound
		}
		found := false
		for index := range current {
			if current[index].Slug != segment {
				continue
			}
			node := current[index]
			view.Breadcrumb = append(view.Breadcrumb, node)
			view.Active = &view.Breadcrumb[len(view.Breadcrumb)-1]
			current = node.Children
			found = true
			break
		}
		if !found {
			return marketPageView{}, ErrNotFound
		}
	}
	view.Path = strings.Join(segments, "/")
	if view.Active != nil && view.Active.HasTypes {
		view.Items, err = reader.UniverseMarketGroupItems(ctx, view.Active.ID)
		if err != nil {
			return marketPageView{}, err
		}
		if len(view.Items) == 1 && view.Items[0].MarketGroupID == view.Active.ID {
			view.Redirect = idPath("market/item", view.Items[0].ID)
		}
	}
	return view, nil
}

func marketMetadata(view marketPageView) Metadata {
	if view.Active == nil {
		return Metadata{Title: "Market Browser", CanonicalPath: "/market", Indexable: true,
			Description: "Browse the regional EVE Online market by category, then compare current orders and traded history."}
	}
	return Metadata{Title: view.Active.Name + " — Market", CanonicalPath: "/market/" + view.Path, Indexable: true,
		Description: "Browse " + view.Active.Name + " in the EVE Online market. Compare current Jita sell orders, universe-wide traded averages, and volume."}
}

func marketGroupPath(base string, node api.UniverseMarketTreeNode) string {
	if base == "" {
		return "/market/" + node.Slug
	}
	return "/market/" + base + "/" + node.Slug
}
func marketTreePath(prefix, slug string) string {
	if prefix == "" {
		return slug
	}
	return prefix + "/" + slug
}
func marketTreeLinkClass(active, path string) string {
	if marketNodeActive(active, path) {
		return "active"
	}
	return ""
}
func marketIntroTitle(view marketPageView) string {
	if view.Active == nil {
		return "Market Explorer"
	}
	return view.Active.Name
}
func marketIntroDescription(view marketPageView) string {
	if view.Active == nil {
		return "Browse the regional market by category, then compare current orders and traded history."
	}
	return fmt.Sprintf("Choose one of %d subcategories to narrow the market.", len(view.Active.Children))
}
func marketCardImage(item api.UniverseMarketGroupItem) string {
	variant := "icon"
	if item.IsShip {
		variant = "render"
	}
	return fmt.Sprintf("/images/types/%d/%s?size=64", item.ID, variant)
}

func marketNodeOpen(activePath, path string) bool {
	return activePath == path || strings.HasPrefix(activePath, path+"/")
}
func marketNodeActive(activePath, path string) bool { return activePath == path }
func marketVisibleGroups(view marketPageView) []api.UniverseMarketTreeNode {
	if view.Active == nil {
		return view.Tree
	}
	return view.Active.Children
}
func marketGroupSubtitle(node api.UniverseMarketTreeNode) string {
	if node.HasTypes {
		return "View market items"
	}
	return fmt.Sprintf("%d subcategories", len(node.Children))
}
func marketGroupIcon(node api.UniverseMarketTreeNode) string {
	if node.HasTypes {
		return "package-search"
	}
	return "folder-tree"
}
func marketBreadcrumbPath(view marketPageView, index int) string {
	segments := make([]string, 0, index+1)
	for _, crumb := range view.Breadcrumb[:index+1] {
		segments = append(segments, crumb.Slug)
	}
	return "/market/" + strings.Join(segments, "/")
}
func marketItemPath(view marketItemPageView) string {
	segments := make([]string, 0, len(view.Item.MarketBreadcrumb))
	for _, crumb := range view.Item.MarketBreadcrumb {
		segments = append(segments, crumb.Slug)
	}
	return strings.Join(segments, "/")
}

func marketNodeByID(nodes []api.UniverseMarketTreeNode, id int64, prefix string) (*api.UniverseMarketTreeNode, string) {
	for index := range nodes {
		path := marketTreePath(prefix, nodes[index].Slug)
		if nodes[index].ID == id {
			return &nodes[index], path
		}
		if found, foundPath := marketNodeByID(nodes[index].Children, id, path); found != nil {
			return found, foundPath
		}
	}
	return nil, ""
}
func marketMetaLabel(id *int64) string {
	if id == nil {
		return ""
	}
	switch *id {
	case 1:
		return "Tech I"
	case 2:
		return "Tech II"
	case 3:
		return "Storyline"
	case 4:
		return "Faction"
	case 5:
		return "Officer"
	case 6:
		return "Deadspace"
	case 14:
		return "Tech III"
	}
	return ""
}
func marketMetaClass(id *int64) string {
	if id == nil {
		return ""
	}
	switch *id {
	case 2:
		return "tech2"
	case 3, 14:
		return "storyline"
	case 4:
		return "faction"
	case 5:
		return "officer"
	case 6:
		return "deadspace"
	}
	return ""
}
func marketCardPrice(value *float64) string {
	if value == nil {
		return "—"
	}
	v := *value
	switch {
	case v >= 1e12:
		return fmt.Sprintf("%.2ft", v/1e12)
	case v >= 1e9:
		return fmt.Sprintf("%.2fb", v/1e9)
	case v >= 1e6:
		return fmt.Sprintf("%.2fm", v/1e6)
	case v >= 1e3:
		return fmt.Sprintf("%.1fk", v/1e3)
	default:
		return marketPrice(v)
	}
}
func marketCardVolume(value *int64) string {
	if value == nil {
		return "No recent volume"
	}
	return fullNumber(float64(*value)) + " traded latest day"
}

type marketItemPageView struct {
	ID     int64
	Item   api.UniverseItemData
	Tree   []api.UniverseMarketTreeNode
	Market itemMarketView
}

func loadMarketItemPage(ctx context.Context, reader UniverseReader, id int64, query map[string][]string) (marketItemPageView, error) {
	if id < 1 || id > 2_147_483_647 {
		return marketItemPageView{}, ErrNotFound
	}
	item, found, err := reader.UniverseItem(ctx, id)
	if err != nil {
		return marketItemPageView{}, err
	}
	if !found || !item.Item.Published || item.Item.Name == "" {
		return marketItemPageView{}, ErrNotFound
	}
	tree, err := reader.UniverseMarketTree(ctx)
	if err != nil {
		return marketItemPageView{}, err
	}
	market, err := loadItemMarket(ctx, reader, id, query)
	if err != nil {
		return marketItemPageView{}, err
	}
	return marketItemPageView{ID: id, Item: item, Tree: tree, Market: market}, nil
}

func marketItemMetadata(view marketItemPageView) Metadata {
	return Metadata{Title: view.Item.Item.Name + " Market", CanonicalPath: fmt.Sprintf("/market/item/%d", view.ID), Indexable: true,
		Description: fmt.Sprintf("Current regional buy and sell orders and corrected market price history for %s in EVE Online.", view.Item.Item.Name),
		Image:       fmt.Sprintf("/images/types/%d/icon?size=256", view.ID), ImageAlt: view.Item.Item.Name}
}

func (s *site) registerMarketPages(mux *http.ServeMux) {
	load := func(r *http.Request) (Page, error) {
		view, err := loadMarketPage(r.Context(), s.options.Universe, r.PathValue("path"))
		if err != nil {
			return Page{}, err
		}
		return Page{Metadata: marketMetadata(view), Content: marketPage(view), SharedCacheTTL: 5 * time.Minute}, nil
	}
	root := s.pageHandler(load)
	mux.Handle("GET /market", root)
	mux.Handle("GET /market/{$}", root)
	mux.HandleFunc("GET /market/{path...}", func(w http.ResponseWriter, r *http.Request) {
		view, err := loadMarketPage(r.Context(), s.options.Universe, r.PathValue("path"))
		if err == nil && view.Redirect != "" {
			http.Redirect(w, r, view.Redirect, http.StatusFound)
			return
		}
		s.pageHandler(func(*http.Request) (Page, error) {
			if err != nil {
				return Page{}, err
			}
			return Page{Metadata: marketMetadata(view), Content: marketPage(view), SharedCacheTTL: 5 * time.Minute}, nil
		}).ServeHTTP(w, r)
	})
	mux.Handle("GET /market/item/{id}", s.pageHandler(func(r *http.Request) (Page, error) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			return Page{}, ErrNotFound
		}
		view, err := loadMarketItemPage(r.Context(), s.options.Universe, id, r.URL.Query())
		if err != nil {
			return Page{}, err
		}
		return Page{Metadata: marketItemMetadata(view), Content: marketItemPage(view), SharedCacheTTL: time.Minute}, nil
	}))
	mux.HandleFunc("GET /_fragments/market/item/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id < 1 || id > 2_147_483_647 {
			http.NotFound(w, r)
			return
		}
		item, found, err := s.options.Universe.UniverseItem(r.Context(), id)
		if err != nil {
			http.Error(w, "Could not load market item", http.StatusServiceUnavailable)
			return
		}
		if !found || !item.Item.Published {
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
		s.renderFragment(w, r, marketStandalonePanel(marketItemPageView{ID: id, Item: item, Market: market}))
	})
	mux.HandleFunc("GET /_fragments/market/tree/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id < 1 || id > 2_147_483_647 {
			http.NotFound(w, r)
			return
		}
		tree, err := s.options.Universe.UniverseMarketTree(r.Context())
		if err != nil {
			http.Error(w, "Could not load market tree", http.StatusServiceUnavailable)
			return
		}
		node, path := marketNodeByID(tree, id, "")
		if node == nil {
			http.NotFound(w, r)
			return
		}
		s.renderFragment(w, r, marketTreeWithParent(node.Children, path, "", id))
	})
}
