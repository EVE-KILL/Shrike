package frontend

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/api"
	"golang.org/x/sync/errgroup"
)

type groupMetaCount struct {
	Name  string
	Count int
}

type groupPageView struct {
	ID          int64
	Tab         string
	Group       api.UniverseGroup
	Published   []api.UniverseGroupType
	Unpublished []api.UniverseGroupType
	MetaGroups  []groupMetaCount
	Kills       api.KilllistPage
}

func loadGroupPage(ctx context.Context, reader UniverseReader, id int64, tab string) (groupPageView, error) {
	if id < 1 || id > 2_147_483_647 || (tab != "" && tab != "dashboard" && tab != "kills") {
		return groupPageView{}, ErrNotFound
	}
	if tab == "" {
		tab = "dashboard"
	}
	var detail api.UniverseGroupData
	var found bool
	var kills api.KilllistPage
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		var err error
		detail, found, err = reader.UniverseGroup(ctx, id)
		return err
	})
	if tab == "kills" {
		group.Go(func() error {
			var err error
			kills, err = reader.UniverseGroupKills(ctx, id, 50, nil)
			return err
		})
	}
	if err := group.Wait(); err != nil {
		return groupPageView{}, err
	}
	if !found || detail.Group.Name == "" {
		return groupPageView{}, ErrNotFound
	}
	view := groupPageView{ID: id, Tab: tab, Group: detail.Group, Kills: kills}
	metaCounts := map[string]int{}
	for _, item := range detail.Types {
		if item.Published {
			view.Published = append(view.Published, item)
			name := item.MetaGroupName
			if name == "" {
				name = "Tech I"
			}
			metaCounts[name]++
		} else {
			view.Unpublished = append(view.Unpublished, item)
		}
	}
	for name, count := range metaCounts {
		view.MetaGroups = append(view.MetaGroups, groupMetaCount{Name: name, Count: count})
	}
	sort.Slice(view.MetaGroups, func(i, j int) bool {
		if view.MetaGroups[i].Count != view.MetaGroups[j].Count {
			return view.MetaGroups[i].Count > view.MetaGroups[j].Count
		}
		return view.MetaGroups[i].Name < view.MetaGroups[j].Name
	})
	return view, nil
}

func groupDescription(view groupPageView) string {
	noun := "inventory types"
	if view.Group.CategoryID == 6 {
		noun = "ship hulls"
	}
	category := strings.ToLower(view.Group.CategoryName)
	if category == "" {
		category = "inventory"
	}
	return fmt.Sprintf("%s is an EVE Online %s group containing %s published %s. Explore every type in the group and its recent combat losses.",
		view.Group.Name, category, fullNumber(float64(view.Group.PublishedTypeCount)), noun)
}

func groupTypeName(item api.UniverseGroupType) string {
	if item.Name != "" {
		return item.Name
	}
	return fmt.Sprintf("Type #%d", item.ID)
}

func groupImage(view groupPageView, item api.UniverseGroupType, size int) string {
	variant := "icon"
	if view.Group.CategoryID == 6 {
		variant = "render"
	}
	return fmt.Sprintf("/images/types/%d/%s?size=%d", item.ID, variant, size)
}

func groupMetadata(view groupPageView) Metadata {
	path := fmt.Sprintf("/group/%d", view.ID)
	title := view.Group.Name
	if view.Tab == "kills" {
		title += " (Kills)"
		path += "/kills"
	}
	var image string
	if len(view.Published) > 0 {
		image = groupImage(view, view.Published[0], 256)
	}
	return Metadata{Title: title, Description: groupDescription(view), CanonicalPath: path, Image: image,
		Indexable: true, StructuredData: map[string]any{"@context": "https://schema.org", "@type": "BreadcrumbList", "itemListElement": []any{
			map[string]any{"@type": "ListItem", "position": 1, "name": "Home", "item": canonicalOrigin + "/"},
			map[string]any{"@type": "ListItem", "position": 2, "name": view.Group.CategoryName, "item": canonicalOrigin + "/market"},
			map[string]any{"@type": "ListItem", "position": 3, "name": view.Group.Name, "item": canonicalOrigin + fmt.Sprintf("/group/%d", view.ID)},
		}},
	}
}

func (s *site) registerUniversePages(mux *http.ServeMux) {
	load := func(r *http.Request) (Page, error) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			return Page{}, ErrNotFound
		}
		view, err := loadGroupPage(r.Context(), s.options.Universe, id, r.PathValue("tab"))
		if err != nil {
			return Page{}, err
		}
		return Page{Metadata: groupMetadata(view), Content: groupPage(view), SharedCacheTTL: 2 * time.Minute}, nil
	}
	mux.Handle("GET /group/{id}", s.pageHandler(load))
	mux.Handle("GET /group/{id}/{tab}", s.pageHandler(load))
	mux.HandleFunc("GET /_fragments/group/tab/{id}/{tab}", s.groupTabFragment)
	mux.HandleFunc("GET /_fragments/group/kills/{id}", s.groupKillsFragment)
	s.registerItemPages(mux)
	s.registerMarketPages(mux)
	s.registerSystemPages(mux)
	s.registerLocationPages(mux)
}

func (s *site) groupTabFragment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	view, err := loadGroupPage(r.Context(), s.options.Universe, id, r.PathValue("tab"))
	if err != nil {
		if err == ErrNotFound {
			http.NotFound(w, r)
		} else {
			http.Error(w, "Could not load group", http.StatusServiceUnavailable)
		}
		return
	}
	s.renderFragment(w, r, groupMain(view))
}

func (s *site) groupKillsFragment(w http.ResponseWriter, r *http.Request) {
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
	data, err := s.options.Universe.UniverseGroupKills(r.Context(), id, limit, after)
	if err != nil {
		http.Error(w, "Could not load kills", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, killList(data, killListOptions{Kind: "group", Page: page, Limit: limit, Filtered: true}))
}
