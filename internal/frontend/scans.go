package frontend

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/a-h/templ"
)

type dscanType struct {
	TypeName string `json:"typeName"`
	TypeID   *int64 `json:"typeId"`
	Count    int64  `json:"count"`
}

type dscanGroup struct {
	GroupID *int64      `json:"groupId"`
	Types   []dscanType `json:"types"`
}

type dscanCategory struct {
	CategoryID *int64                `json:"categoryId"`
	Groups     map[string]dscanGroup `json:"groups"`
}

type dscanResult struct {
	Grouped     map[string]dscanCategory `json:"grouped"`
	TotalCount  int64                    `json:"totalCount"`
	UniqueTypes int                      `json:"uniqueTypes"`
}

type dscanGroupView struct {
	Name, Key string
	Types     []dscanType
	Count     int64
}

type dscanCategoryView struct {
	Name   string
	Groups []dscanGroupView
	Count  int64
}

type dscanView struct {
	Categories  []dscanCategoryView
	TotalCount  int64
	UniqueTypes int
}

func makeDscanView(result dscanResult) dscanView {
	view := dscanView{TotalCount: result.TotalCount, UniqueTypes: result.UniqueTypes}
	for categoryName, category := range result.Grouped {
		item := dscanCategoryView{Name: categoryName}
		for groupName, group := range category.Groups {
			gv := dscanGroupView{Name: groupName, Key: categoryName + "-" + groupName, Types: group.Types}
			for _, t := range group.Types {
				gv.Count += t.Count
			}
			item.Count += gv.Count
			item.Groups = append(item.Groups, gv)
		}
		slices.SortFunc(item.Groups, func(a, b dscanGroupView) int {
			if a.Count != b.Count {
				if a.Count > b.Count {
					return -1
				}
				return 1
			}
			return strings.Compare(a.Name, b.Name)
		})
		view.Categories = append(view.Categories, item)
	}
	slices.SortFunc(view.Categories, func(a, b dscanCategoryView) int {
		if a.Name == "Ship" && b.Name != "Ship" {
			return -1
		}
		if b.Name == "Ship" && a.Name != "Ship" {
			return 1
		}
		if a.Count != b.Count {
			if a.Count > b.Count {
				return -1
			}
			return 1
		}
		return strings.Compare(a.Name, b.Name)
	})
	return view
}

type localCharacter struct {
	CharacterID int64  `json:"characterId"`
	Name        string `json:"name"`
	Kills       int64  `json:"kills"`
}

type localCorporation struct {
	Name       string           `json:"name"`
	Ticker     string           `json:"ticker"`
	Characters []localCharacter `json:"characters"`
}

type localAlliance struct {
	Name         string                      `json:"name"`
	Ticker       string                      `json:"ticker"`
	Corporations map[string]localCorporation `json:"corporations"`
}

type localScanResult struct {
	Alliances       map[string]localAlliance    `json:"alliances"`
	Corporations    map[string]localCorporation `json:"corporations"`
	Unresolved      []string                    `json:"unresolved"`
	TotalCharacters int                         `json:"totalCharacters"`
	TotalDangerous  int                         `json:"totalDangerous"`
}

type localCorpView struct {
	ID     int64
	Name   string
	Ticker string
	People []localCharacter
}

type localAllianceView struct {
	ID     int64
	Name   string
	Ticker string
	Corps  []localCorpView
	People int
}

type localThreat struct {
	localCharacter
	Corporation string
	Alliance    string
}

type localScanView struct {
	Alliances       []localAllianceView
	Corporations    []localCorpView
	Threats         []localThreat
	Unresolved      []string
	TotalCharacters int
	TotalCorps      int
}

func makeLocalScanView(result localScanResult) localScanView {
	view := localScanView{Unresolved: result.Unresolved, TotalCharacters: result.TotalCharacters}
	for _, key := range scanEntityKeys(result.Alliances) {
		alliance := result.Alliances[key]
		item := localAllianceView{ID: numericMapKey(key), Name: alliance.Name, Ticker: alliance.Ticker}
		for _, corpKey := range scanEntityKeys(alliance.Corporations) {
			corp := alliance.Corporations[corpKey]
			cv := localCorpView{ID: numericMapKey(corpKey), Name: corp.Name, Ticker: corp.Ticker, People: slices.Clone(corp.Characters)}
			slices.SortStableFunc(cv.People, compareLocalCharacters)
			item.Corps = append(item.Corps, cv)
			item.People += len(cv.People)
			view.TotalCorps++
			for _, ch := range cv.People {
				if ch.Kills > 5 {
					view.Threats = append(view.Threats, localThreat{ch, corp.Name, alliance.Name})
				}
			}
		}
		slices.SortFunc(item.Corps, compareLocalCorps)
		view.Alliances = append(view.Alliances, item)
	}
	for _, key := range scanEntityKeys(result.Corporations) {
		corp := result.Corporations[key]
		cv := localCorpView{ID: numericMapKey(key), Name: corp.Name, Ticker: corp.Ticker, People: slices.Clone(corp.Characters)}
		slices.SortStableFunc(cv.People, compareLocalCharacters)
		view.Corporations = append(view.Corporations, cv)
		view.TotalCorps++
		for _, ch := range cv.People {
			if ch.Kills > 5 {
				view.Threats = append(view.Threats, localThreat{ch, corp.Name, ""})
			}
		}
	}
	slices.SortFunc(view.Alliances, func(a, b localAllianceView) int {
		if a.People != b.People {
			if a.People > b.People {
				return -1
			}
			return 1
		}
		return cmp.Compare(a.ID, b.ID)
	})
	slices.SortFunc(view.Corporations, compareLocalCorps)
	slices.SortStableFunc(view.Threats, func(a, b localThreat) int { return compareLocalCharacters(a.localCharacter, b.localCharacter) })
	return view
}

func compareLocalCharacters(a, b localCharacter) int {
	if a.Kills != b.Kills {
		if a.Kills > b.Kills {
			return -1
		}
		return 1
	}
	return 0
}

func compareLocalCorps(a, b localCorpView) int {
	if len(a.People) != len(b.People) {
		if len(a.People) > len(b.People) {
			return -1
		}
		return 1
	}
	return cmp.Compare(a.ID, b.ID)
}

// JavaScript enumerates numeric entity keys in ascending order and preserves
// that order for count ties. Retain the source's stable ordering in Go too.
func scanEntityKeys[T any](entities map[string]T) []string {
	keys := make([]string, 0, len(entities))
	for key := range entities {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b string) int { return cmp.Compare(numericMapKey(a), numericMapKey(b)) })
	return keys
}

func numericMapKey(key string) int64 {
	id, _ := strconv.ParseInt(key, 10, 64)
	return id
}

func scanTypeImage(id int64) string {
	return fmt.Sprintf("/images/types/%d/icon?size=64", id)
}

func unaffiliatedCount(corps []localCorpView) int {
	count := 0
	for _, corp := range corps {
		count += len(corp.People)
	}
	return count
}

func unresolvedNames(names []string) string {
	if len(names) <= 20 {
		return strings.Join(names, ", ")
	}
	return strings.Join(names[:20], ", ") + "..."
}

func scanInputInstruction(kind string) string {
	if kind == "dscan" {
		return "Copy your directional scan from EVE Online and paste it here."
	}
	return "Copy the character names from your EVE local chat and paste them here."
}
func scanSavedTitle(kind string) string {
	if kind == "dscan" {
		return "D-Scan Result"
	}
	return "Local Scan Result"
}

type scanResultView struct {
	Dscan *dscanView
	Local *localScanView
}

func decodeScanResult(kind string, raw []byte) (scanResultView, error) {
	switch kind {
	case "dscan":
		var result dscanResult
		if err := json.Unmarshal(raw, &result); err != nil {
			return scanResultView{}, err
		}
		view := makeDscanView(result)
		return scanResultView{Dscan: &view}, nil
	case "localscan":
		var result localScanResult
		if err := json.Unmarshal(raw, &result); err != nil {
			return scanResultView{}, err
		}
		view := makeLocalScanView(result)
		return scanResultView{Local: &view}, nil
	default:
		return scanResultView{}, fmt.Errorf("invalid scan type")
	}
}

func validScanHash(hash string) bool {
	if len(hash) != 64 {
		return false
	}
	for _, ch := range hash {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return false
		}
	}
	return true
}

func scanMetadata(kind string, saved bool, hash string) Metadata {
	meta := scanPageMetadata(kind, saved, hash)
	meta.Stylesheets = []string{"utilities.css", "utility-page.css", "scans.css"}
	if !saved {
		meta.ModuleScripts = []string{"scans.js"}
	}
	return meta
}

func scanPageMetadata(kind string, saved bool, hash string) Metadata {
	if kind == "dscan" {
		if saved {
			return Metadata{Title: "D-Scan Result", Description: "Shared D-Scan analysis from EVE Online — fleet composition breakdown.", CanonicalPath: "/tools/dscan/" + hash, Indexable: true}
		}
		return Metadata{Title: "D-Scan Analyzer", Description: "Paste your EVE Online directional scan to analyze fleet composition, ship types, and threat assessment.", CanonicalPath: "/tools/dscan", Indexable: true}
	}
	if saved {
		return Metadata{Title: "Local Scan Result", Description: "Shared local scan analysis from EVE Online — pilot affiliations and threat levels.", CanonicalPath: "/tools/localscan/" + hash, Indexable: true}
	}
	return Metadata{Title: "Local Scan", Description: "Paste your EVE Online local chat to analyze pilot affiliations, threat levels, and fleet composition.", CanonicalPath: "/tools/localscan", Indexable: true}
}

func (s *site) savedScanPage(ctx context.Context, kind, hash string) (Page, error) {
	reader := s.options.Scans
	if reader == nil {
		reader = s.options.Homepage
	}
	if !validScanHash(hash) || reader == nil {
		return Page{}, ErrNotFound
	}
	raw, found, err := reader.SavedScan(ctx, kind, hash)
	if err != nil {
		return Page{}, err
	}
	if !found {
		return Page{}, ErrNotFound
	}
	result, err := decodeScanResult(kind, raw)
	if err != nil {
		return Page{}, err
	}
	return Page{Metadata: scanMetadata(kind, true, hash), Content: scanSavedPage(kind, result)}, nil
}

func (s *site) scanResultFragment(w http.ResponseWriter, r *http.Request) {
	kind := r.PathValue("kind")
	if kind != "dscan" && kind != "localscan" {
		http.NotFound(w, r)
		return
	}
	defer func() { _ = r.Body.Close() }()
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 8<<20))
	if err != nil {
		http.Error(w, "Scan result is too large", http.StatusRequestEntityTooLarge)
		return
	}
	result, err := decodeScanResult(kind, raw)
	if err != nil {
		http.Error(w, "Invalid scan result", http.StatusBadRequest)
		return
	}
	s.renderFragment(w, r, scanResultContent(result))
}

func scanResultContent(view scanResultView) templ.Component {
	return scanResults(view)
}
