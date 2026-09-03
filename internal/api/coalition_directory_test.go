package api

import (
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

func TestCoalitionDirectoryRoutesArePublicToReadAndAuthenticatedToWrite(t *testing.T) {
	a := humachi.New(chi.NewRouter(), huma.DefaultConfig("coalitions-test", "test"))
	registerCoalitionDirectoryRoutes(a, Options{})

	list := a.OpenAPI().Paths["/coalitions"]
	if list == nil || list.Get == nil || list.Post == nil {
		t.Fatal("coalition collection routes were not registered")
	}
	if len(list.Get.Security) != 0 {
		t.Fatal("GET /coalitions unexpectedly requires a session")
	}
	if len(list.Post.Security) == 0 {
		t.Fatal("POST /coalitions does not require a session")
	}
	if list.Post.DefaultStatus != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", list.Post.DefaultStatus)
	}
	detail := a.OpenAPI().Paths["/coalitions/{slug}"]
	if detail == nil || detail.Get == nil || detail.Patch == nil {
		t.Fatal("coalition detail routes were not registered")
	}
	if len(detail.Patch.Security) == 0 {
		t.Fatal("PATCH /coalitions/{slug} does not require a session")
	}
	if detail.Get.Parameters[0].Name != "slug" || detail.Get.Parameters[0].In != "path" {
		t.Fatalf("detail path parameter = %#v", detail.Get.Parameters)
	}
}

func TestValidateCoalitionCreateNormalizesAndDeduplicates(t *testing.T) {
	source := " https://example.com/coalition "
	body := &coalitionCreateBody{
		Name:        stringPointer(" The Imperium "),
		Description: "  A coalition description.  ",
		SourceURL:   optional[string]{Set: true, Value: &source},
		AllianceIDs: requestList[jsInt]{jsInt(3), jsInt(1), jsInt(3), jsInt(2)},
	}
	got, err := validateCoalitionCreate(body)
	if err != nil {
		t.Fatal(err)
	}
	if got.Slug != "the-imperium" || got.Name != "The Imperium" {
		t.Fatalf("normalized identity = %q, %q", got.Slug, got.Name)
	}
	if got.Description != "A coalition description." {
		t.Fatalf("description = %q", got.Description)
	}
	if got.SourceURL == nil || *got.SourceURL != "https://example.com/coalition" {
		t.Fatalf("source URL = %#v", got.SourceURL)
	}
	if want := []int32{1, 2, 3}; !reflect.DeepEqual(got.AllianceIDs, want) {
		t.Fatalf("alliances = %v, want %v", got.AllianceIDs, want)
	}
}

func TestValidateCoalitionCreateRejectsEmptyMembershipAndUnsafeSource(t *testing.T) {
	name := "Test Coalition"
	if _, err := validateCoalitionCreate(&coalitionCreateBody{Name: &name}); err == nil {
		t.Fatal("empty membership was accepted")
	}
	unsafe := "javascript:alert(1)"
	_, err := validateCoalitionCreate(&coalitionCreateBody{
		Name:        &name,
		SourceURL:   optional[string]{Set: true, Value: &unsafe},
		AllianceIDs: requestList[jsInt]{jsInt(1)},
	})
	if err == nil || !strings.Contains(err.Error(), "absolute HTTP") {
		t.Fatalf("unsafe source error = %v", err)
	}
}

func TestSummarizeCoalitionEditExplainsEveryPublicChange(t *testing.T) {
	oldSource, newSource := "https://old.example", "https://new.example"
	before := coalitionSnapshot{
		Name: "Old Name", Description: "Old", SourceURL: &oldSource,
		AllianceIDs: []int32{1, 2, 3},
	}
	after := coalitionSnapshot{
		Name: "New Name", Description: "New", SourceURL: &newSource,
		AllianceIDs: []int32{2, 3, 4, 5},
	}
	want := "Renamed the coalition; updated the description; updated the verification source; added 2 alliances; removed 1 alliance"
	if got := summarizeCoalitionEdit(before, after); got != want {
		t.Fatalf("summary = %q, want %q", got, want)
	}
	if got := summarizeCoalitionEdit(after, after); got != "" {
		t.Fatalf("no-op summary = %q", got)
	}
}

func TestCoalitionDirectoryStatsDeduplicateAllianceAttackers(t *testing.T) {
	if !strings.Contains(coalitionDirectorySummarySQL, "SELECT DISTINCT cm.coalition_id, ka.killmail_id") {
		t.Fatal("coalition kills can be multiplied by multiple alliance attackers")
	}
	if !strings.Contains(coalitionDirectorySummarySQL, "count(ch.character_id)::bigint AS member_count") {
		t.Fatal("coalition member totals must use character affiliations rather than stale alliance counters")
	}
}
