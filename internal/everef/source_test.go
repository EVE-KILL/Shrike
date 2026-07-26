package everef

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

// EVE Ref publishes no manifest, so discovering what exists means scraping a
// directory index. That makes the regex load-bearing: if the listing format
// changes, an importer reports "no archives published" and silently stops
// backfilling rather than failing.
func TestListParsesADirectoryIndex(t *testing.T) {
	const index = `<html><head><title>Index of /killmails/2026</title></head><body>
<h1>Index of /killmails/2026</h1><pre>
<a href="../">../</a>
<a href="killmails-2026-07-18.tar.bz2">killmails-2026-07-18.tar.bz2</a>  18-Jul-2026 02:14  11M
<a href="killmails-2026-07-20.tar.bz2">killmails-2026-07-20.tar.bz2</a>  20-Jul-2026 02:11  12M
<a href="killmails-2026-07-19.tar.bz2">killmails-2026-07-19.tar.bz2</a>  19-Jul-2026 02:09  10M
<a href="killmails-2026-07-20.tar.bz2.md5">killmails-2026-07-20.tar.bz2.md5</a>  20-Jul-2026 02:11  60
</pre></body></html>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, index)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, UserAgent: "test", HTTP: srv.Client()}
	got, err := c.List(context.Background(), srv.URL+"/killmails/2026/", killmailArchive)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"2026-07-18", "2026-07-19", "2026-07-20"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	// Sorted, because callers resume from a bookmark by string comparison.
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d = %s, want %s", i, got[i], want[i])
		}
	}
}

// A checksum file sitting beside an archive must not be mistaken for one; the
// importer would then fetch a 60-byte file and report zero killmails.
func TestListIgnoresNeighbouringFiles(t *testing.T) {
	const index = `<a href="wars-2026-07-20_00-13-08.tar.bz2">x</a>
<a href="wars-2026-07-20_00-13-08.tar.bz2.md5">x</a>
<a href="wars-2026-07-21_00-13-08.tar.bz2">x</a>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, index)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, UserAgent: "test", HTTP: srv.Client()}
	got, err := c.List(context.Background(), srv.URL+"/", warDailyArchive)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %v, want the two archives without the checksum", got)
	}
	for _, name := range got {
		if strings.HasSuffix(name, ".md5") {
			t.Errorf("a checksum file was listed as an archive: %s", name)
		}
	}
}

// The sovereignty daily listing is dated directories, not files.
func TestListMatchesDatedDirectories(t *testing.T) {
	const index = `<a href="../">../</a>
<a href="2023-01-15/">2023-01-15/</a>
<a href="2023-01-16/">2023-01-16/</a>
<a href="notes.txt">notes.txt</a>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, index)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, UserAgent: "test", HTTP: srv.Client()}
	got, err := c.List(context.Background(), srv.URL+"/", sovDayDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "2023-01-15" || got[1] != "2023-01-16" {
		t.Errorf("got %v, want the two dated directories", got)
	}
}

// A day EVE Ref has not published yet is routine, not a failure — the most
// recent day is regularly absent for hours. Importers rely on telling the two
// apart to keep going.
func TestNotPublishedIsDistinguishable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/gone":
			w.WriteHeader(http.StatusNotFound)
		case "/broken":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			_, _ = io.WriteString(w, `{"ok":true}`)
		}
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, UserAgent: "test", HTTP: srv.Client()}
	ctx := context.Background()

	var out map[string]bool
	if err := c.JSON(ctx, srv.URL+"/gone", &out); !errors.Is(err, ErrNotPublished) {
		t.Errorf("a 404 returned %v, want ErrNotPublished", err)
	}
	// A 500 is EVE Ref being unwell, which must not be mistaken for absence —
	// treating it as "not published" would advance the bookmark past real data.
	if err := c.JSON(ctx, srv.URL+"/broken", &out); err == nil || errors.Is(err, ErrNotPublished) {
		t.Errorf("a 500 returned %v, want a plain error", err)
	}
	if err := c.JSON(ctx, srv.URL+"/fine", &out); err != nil {
		t.Errorf("a 200 returned %v", err)
	}
	if !out["ok"] {
		t.Error("the body was not decoded")
	}
}

// Everything large is bzip2'd, and the decision to decompress is made from the
// URL suffix.
func TestStreamDecompressesByExtension(t *testing.T) {
	c := fileServer(t, map[string]string{
		"/market-history/2026/market-history-2026-07-20.csv.bz2": "market-history-2026-07-20.csv.bz2",
	})

	var got string
	err := c.Stream(context.Background(),
		c.url("/market-history/2026/market-history-2026-07-20.csv.bz2"),
		func(r io.Reader) error {
			body, err := io.ReadAll(r)
			got = string(body)
			return err
		})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "average,date,highest") {
		t.Errorf("the body was not decompressed: %.40q", got)
	}
}

// Regexes used by the listing tests, asserted directly so a change to them is
// visible here rather than as an empty backfill.
func TestListingPatternsAreAnchored(t *testing.T) {
	cases := []struct {
		name    string
		pattern *regexp.Regexp
		input   string
		match   bool
	}{
		{"killmail archive", killmailArchive, "killmails-2026-07-20.tar.bz2", true},
		{"killmail checksum", killmailArchive, "killmails-2026-07-20.tar.bz2.md5", false},
		{"war archive", warDailyArchive, "wars-2026-07-20_00-13-08.tar.bz2", true},
		{"war checksum", warDailyArchive, "wars-2026-07-20_00-13-08.tar.bz2.md5", false},
		{"sovereignty directory", sovDayDir, "2023-01-15/", true},
		{"sovereignty file", sovDayDir, "2023-01-15.json", false},
		{"sovereignty day file", sovDayFile, "sovereignty-map-2023-01-15T00-00-02.json.bz2", true},
	}
	for _, c := range cases {
		if got := c.pattern.MatchString(c.input); got != c.match {
			t.Errorf("%s: %q matched=%v, want %v", c.name, c.input, got, c.match)
		}
	}
}
