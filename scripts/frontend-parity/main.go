// A disposable local account/API fixture for Go browser interaction tests.
// Production services are used only for public GET data and image assets.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/eve-kill/shrike/internal/api"
	"github.com/eve-kill/shrike/internal/frontend"
	"github.com/eve-kill/shrike/internal/migrate"
	"github.com/eve-kill/shrike/internal/testfixtures"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Keep all API transactions real without dispatching optional River jobs.
type fixtureDB struct{ *pgxpool.Pool }
type account struct {
	id            int32
	name, session string
	admin         bool
}

var cases = []struct {
	id, name      string
	sides         int
	prize, paused bool
}{
	{"PrizeOpen00001", "Fixture funding open", 2, true, false},
	{"PrizeReady0001", "Fixture official results", 2, true, false},
	{"PrizeWait00001", "Fixture settling", 2, true, false},
	{"FourSides00001", "Fixture four sides", 4, false, false},
	{"PausedCamp0001", "Fixture processing paused", 2, false, true},
	{"EmptyCamp00001", "Fixture area without activity", 0, false, false},
}

func main() {
	if err := runCommand(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func runCommand() error {
	backend := flag.String("backend", "", "Render current Go assets against an existing fixture API; no database is created")
	backendPort := flag.Int("backend-port", 4451, "Loopback port for current rendering against an existing backend")
	fixturePort := flag.Int("fixture-port", 4450, "Loopback port for a new disposable account/API fixture")
	referencePort := flag.Int("reference-port", 0, "Optional loopback port serving the retained renderer and its assets through a read-only anonymous proxy")
	public := flag.String("public", "http://127.0.0.1:4200", "Public read-only Go preview")
	publicReview := flag.Bool("public-review", false, "Compare complete public renderers against the read-only Go preview; no database is created")
	reviewGoPort := flag.Int("go-port", 4201, "Go facade port in public-review mode")
	mediaFixtures := flag.Bool("media-fixtures", false, "Use a deterministic local GIF provider and comment relay for browser tests")
	faultFixtures := flag.Bool("faults", false, "Enable local API/fragment delay and failure controls for comparison tests")
	trackerRelay := flag.Bool("tracker-relay", false, "Use a local tracker relay for live notification and reconnect comparisons")
	blogFixtures := flag.Bool("blog-fixtures", false, "Seed paginated, rich, archived and draft blog posts in the disposable local database")
	adminFixtures := flag.Bool("admin-fixtures", false, "Seed user directory pages and admin attention badges in the disposable local database")
	submissionFixtures := flag.Bool("submission-fixtures", false, "Intercept killmail submissions and arrivals in memory; never dispatch jobs")
	referenceCoalitions := flag.Bool("reference-coalitions", false, "Render coalition fixtures with the archived original JavaScript on the reference port")
	referenceEntities := flag.Bool("reference-entities", false, "Render entity comparisons with the archived original scripts")
	referencePublicKills := flag.Bool("reference-public-kills", false, "Render public kill feeds and detail comparisons with archived original scripts")
	referenceFittings := flag.Bool("reference-fittings", false, "Compare the archived fitting editor with the same Go markup and pinned engine")
	referenceMaps := flag.Bool("reference-maps", false, "Compare archived map controllers/helpers with identical Go markup and local read data")
	referenceWars := flag.Bool("reference-wars", false, "Compare original war and faction-war scripts against identical current Go markup")
	referenceRoam := flag.Bool("reference-roam", false, "Compare original roam report and composition scripts against current Go markup")
	referenceTools := flag.Bool("reference-tools", false, "Compare archived scan, API documentation and MCP controllers against current Go markup")
	referenceAnalysis := flag.Bool("reference-analysis", false, "Compare archived statistics, rankings and graph controllers against current Go markup")
	referenceStatus := flag.Bool("reference-status", false, "Compare archived status rendering with captured DTOs and an in-memory reconnect relay")
	referenceLegacy := flag.Bool("reference-legacy", false, "Compare the archived historical archive controller against public read-only data")
	referenceUniverse := flag.Bool("reference-universe", false, "Compare original market, item/group and location scripts against identical public GET data")
	entityDatabase := flag.Bool("entity-database", false, "Read entity pages from the existing disposable local database in read-only sessions")
	flag.Parse()
	if *publicReview {
		if *backend != "" || *mediaFixtures || *trackerRelay || *blogFixtures || *adminFixtures || *submissionFixtures || *referenceCoalitions || *referenceEntities || *referencePublicKills || *referenceFittings || *referenceMaps || *referenceWars || *referenceRoam || *referenceTools || *referenceAnalysis || *referenceStatus || *referenceLegacy || *referenceUniverse || *entityDatabase || *referencePort != 0 {
			return errors.New("public review cannot run private fixture modes")
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if *publicReview {
		return runPublicReview(ctx, *public, *reviewGoPort, *faultFixtures)
	}
	return run(ctx, *backend, *public, *mediaFixtures, *faultFixtures, *trackerRelay, *blogFixtures, *adminFixtures, *submissionFixtures, *referenceCoalitions, *referenceEntities, *referencePublicKills, *referenceFittings, *referenceMaps, *referenceWars, *referenceRoam, *referenceTools, *referenceAnalysis, *referenceStatus, *referenceLegacy, *referenceUniverse, *entityDatabase, *backendPort, *referencePort, *fixturePort)
}

func localURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.User != nil || (u.Hostname() != "127.0.0.1" && u.Hostname() != "localhost") || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return nil, errors.New("fixture upstream must be a plain loopback HTTP origin")
	}
	return u, nil
}

func run(ctx context.Context, backend, public string, mediaFixtures, faultFixtures, trackerRelay, blogFixtures, adminFixtures, submissionFixtures, referenceCoalitions, referenceEntities, referencePublicKills, referenceFittings, referenceMaps, referenceWars, referenceRoam, referenceTools, referenceAnalysis, referenceStatus, referenceLegacy, referenceUniverse, entityDatabase bool, backendPort, referencePort, fixturePort int) error {
	if fixturePort < 1 || fixturePort > 65535 {
		return errors.New("fixture-port needs a valid loopback port")
	}
	faults := &fixtureFaults{enabled: faultFixtures}
	if backend != "" && (backendPort < 1 || backendPort > 65535) {
		return errors.New("backend-port needs a valid loopback port")
	}
	if referenceAnalysis && (backend == "" || referencePort == 0) {
		return errors.New("reference-analysis requires an existing backend and reference-port")
	}
	if referenceStatus && (backend == "" || referencePort == 0) {
		return errors.New("reference-status requires an existing backend and reference-port")
	}
	if referenceUniverse && (backend == "" || referencePort == 0) {
		return errors.New("reference-universe requires an existing backend and reference-port")
	}
	if referenceLegacy && (backend == "" || referencePort == 0) {
		return errors.New("reference-legacy requires an existing backend and reference-port")
	}
	if referenceTools && (backend == "" || referencePort == 0 || backend == public) {
		return errors.New("reference-tools requires a separate private fixture backend and reference-port")
	}
	if referenceRoam && (backend == "" || referencePort == 0) {
		return errors.New("reference-roam requires an existing backend and reference-port")
	}
	if referenceWars && (backend == "" || referencePort == 0) {
		return errors.New("reference-wars requires an existing backend and reference-port")
	}
	if referenceMaps && (backend == "" || referencePort == 0) {
		return errors.New("reference-maps requires an existing local backend and reference-port")
	}
	if referenceFittings && (backend == "" || referencePort == 0) {
		return errors.New("reference-fittings requires an existing local backend and reference-port")
	}
	if referencePublicKills && (!entityDatabase || referencePort == 0) {
		return errors.New("reference-public-kills requires entity-database and reference-port")
	}
	if referenceEntities && (!entityDatabase || referencePort == 0) {
		return errors.New("reference-entities requires entity-database and reference-port")
	}
	if entityDatabase && backend == "" {
		return errors.New("entity-database requires an existing local backend")
	}
	if submissionFixtures && backend == "" {
		return errors.New("submission-fixtures requires an existing local backend")
	}
	if referenceCoalitions && (backend == "" || referencePort == 0) {
		return errors.New("reference-coalitions requires an existing local backend and reference-port")
	}
	if referencePort != 0 && (backend == "" || referencePort < 1 || referencePort > 65535 || referencePort == backendPort) {
		return errors.New("reference-port requires an existing backend and a separate valid loopback port")
	}
	if backend != "" {
		if blogFixtures || adminFixtures {
			return errors.New("seeding requires the owned disposable database")
		}
		target, err := localURL(backend)
		if err != nil {
			return err
		}
		proxy := httputil.NewSingleHostReverseProxy(target)
		publicTarget, err := localURL(public)
		if err != nil {
			return err
		}
		var apiProxy http.Handler = proxy
		var sessions frontend.SessionReader = remoteSession{target.String()}
		// Both URLs are restricted to loopback; aliases must not turn the
		// public renderer into a write-enabled private fixture.
		if target.Port() == publicTarget.Port() {
			if referenceTools {
				return errors.New("reference-tools requires a separate private fixture backend")
			}
			sessions = nil
			proxy = anonymousFixtureProxy(target)
			apiProxy = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet && r.Method != http.MethodHead {
					http.Error(w, "Fixture public proxy is read-only", http.StatusMethodNotAllowed)
					return
				}
				proxy.ServeHTTP(w, r)
			})
		}
		if mediaFixtures {
			media := newCommentMediaFixture()
			apiProxy = media.facade(apiProxy)
		}
		battleProxy := anonymousFixtureProxy(publicTarget)
		battleAPI := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				http.Error(w, "Fixture public proxy is read-only", http.StatusMethodNotAllowed)
				return
			}
			battleProxy.ServeHTTP(w, r)
		})
		var homepage frontend.HomepageReader
		var entities frontend.EntityReader
		var universe frontend.UniverseReader
		var legacy frontend.LegacyReader
		if referenceLegacy {
			legacy = remoteLegacy{publicTarget.String()}
		}
		if entityDatabase {
			pool, err := openEntityDatabase(ctx, os.Getenv("TEST_DATABASE_URL"))
			if err != nil {
				return err
			}
			defer pool.Close()
			reader := api.NewHomepageReader(&fixtureDB{pool})
			entities = fixtureEntityReader{reader}
			if referenceFittings {
				universe = fixtureFittingReader{reader}
			}
			if referencePublicKills {
				homepage = reader
			}
		}
		if referenceUniverse {
			universe = remoteUniverse{publicTarget.String()}
			if entities == nil {
				entities = api.NewHomepageReader(nil)
			}
			entities = remoteLocationEntities{entities, publicTarget.String()}
		}
		ui := fixtureFrameHandler(frontend.New(frontend.Options{Homepage: homepage, Universe: universe, Legacy: legacy, AdvancedSearch: remoteAdvancedSearch{target.String()}, CampaignsAPI: apiProxy, ConflictsAPI: apiProxy, FittingsAPI: faults.wrap(apiProxy), BattleGenerator: remoteGenerator{target.String()}, Wars: remoteWars{publicTarget.String()}, FactionWars: remoteFactionWar{publicTarget.String()}, Statistics: remoteStatistics{target.String()}, Rankings: remoteRankings{target.String()}, Scans: remoteScans{target.String()}, RoamReports: remoteRoamReports{target.String()}, CommentsAPI: apiProxy, Sessions: sessions, AccountAPI: faults.wrap(apiProxy), Killmail: remoteKillmail{target.String()}, Blog: remoteBlog{target.String()}, BlogPreviewAPI: apiProxy, Coalitions: remoteCoalitions{target.String()}, Entities: entities}))
		comparison := func(renderer http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if os.Getenv("SHRIKE_FIXTURE_TRACE") == "1" {
					// #nosec G706 -- %q escapes request strings, including embedded line breaks.
					log.Printf("fixture request started: %q %q", r.Method, r.URL.Path)
					// #nosec G706 -- %q escapes the matching completion record as well.
					defer log.Printf("fixture request finished: %q %q", r.Method, r.URL.Path)
				}
				if fixtureAudioTrace(w, r) {
					return
				} else if fixtureBlogCover(w, r) {
					return
				} else if fixtureViewportPage(w, r) {
					return
				} else if fixtureEntitySearch(w, r) {
					return
				} else if fixtureEntityImage(w, r) {
					return
				} else if r.URL.Path == "/api/conflicts/battles" || r.URL.Path == "/api/conflicts/wars" || r.URL.Path == "/api/wars/eligible" || strings.HasPrefix(r.URL.Path, "/api/map/") {
					battleAPI.ServeHTTP(w, r)
				} else if universe != nil && strings.HasPrefix(r.URL.Path, "/_fragments/fits/variants/") {
					renderer.ServeHTTP(w, r)
				} else if backendPath(r.URL.Path) {
					apiProxy.ServeHTTP(w, r)
				} else {
					renderer.ServeHTTP(w, r)
				}
			})
		}
		wrap := faults.wrap
		if submissionFixtures {
			submissions, relay := newFixtureSubmissions(), newFixtureTrackerRelay()
			wrap = func(handler http.Handler) http.Handler { return faults.wrap(submissions.wrap(relay.wrap(handler))) }
		}
		if referenceStatus {
			relay, previous := newFixtureStatusRelay(), wrap
			wrap = func(handler http.Handler) http.Handler { return previous(relay.wrap(handler)) }
		}
		servers := []*http.Server{{Addr: fmt.Sprintf("127.0.0.1:%d", backendPort), Handler: wrap(comparison(fixtureGlobalSite(ui)))}}
		if referencePort != 0 {
			reference := referenceRenderer(target)
			if referenceCoalitions {
				scripts, err := archivedCoalitionScripts(".data/source-archives/browser-before-ts-pipeline-20261007.tar.gz")
				if err != nil {
					return err
				}
				reference = coalitionReferenceRenderer(reference, ui, scripts)
			}
			if referenceEntities {
				scripts, err := archivedComponentScripts(".data/source-archives/browser-before-ts-pipeline-20261007.tar.gz", []string{"entity.js", "fragment.js", "browser-request.js", "default-tab.js"})
				if err != nil {
					return err
				}
				reference = entityReferenceRenderer(reference, ui, scripts)
			}
			if referencePublicKills {
				scripts, err := archivedComponentScripts(".data/source-archives/browser-before-ts-pipeline-20261007.tar.gz", []string{"home.js", "kill-detail.js", "fragment.js", "kill-row.js"})
				if err != nil {
					return err
				}
				reference = publicKillsReferenceRenderer(reference, ui, scripts)
			}
			if referenceFittings {
				scripts, err := archivedComponentScripts(".data/source-archives/browser-before-ts-pipeline-20261007.tar.gz", []string{"fit-editor.js", "dogma.js", "fit-wheel.js", "fit-stats.js", "fit-resources.js", "fit-variants.js", "fit-document.js", "fit-capacity.js", "fit-catalog-search.js", "fit-eft.js", "fit-table-data.js", "fit-drag-data.js", "fits-catalog.js", "fits-search.js", "fit-search-state.js", "search-picker.js", "browser-request.js", "fit-detail.js", "comment-thread.js", "comment-input.js", "comment-stream.js", "comment-controls.js", "comment-input-data.js", "klipy-picker.js"})
				if err != nil {
					return err
				}
				reference = fittingReferenceRenderer(reference, ui, scripts)
			}
			if referenceMaps {
				scripts, err := archivedComponentScripts(".data/source-archives/browser-before-ts-pipeline-20261007.tar.gz", []string{"map-page.js", "map-live.js", "map-canvas.js", "map-aiid-audio.js"})
				if err != nil {
					return err
				}
				reference = mapReferenceRenderer(reference, ui, scripts)
			}
			if referenceWars {
				scripts, err := archivedComponentScripts(".data/source-archives/browser-before-ts-pipeline-20261007.tar.gz", []string{"wars.js", "war-detail.js", "fragment.js", "faction-war.js", "faction-war-map.js", "faction-war-data.js"})
				if err != nil {
					return err
				}
				reference = warsReferenceRenderer(reference, ui, scripts)
			}
			if referenceRoam {
				scripts, err := archivedComponentScripts(".data/source-archives/browser-before-ts-pipeline-20261007.tar.gz", []string{"roam-report.js", "roam-composition.js", "date-time-picker.js", "map-canvas.js", "map-aiid-audio.js"})
				if err != nil {
					return err
				}
				reference = roamReferenceRenderer(reference, ui, scripts)
			}
			if referenceTools {
				scripts, err := archivedComponentScripts(".data/source-archives/browser-before-ts-pipeline-20261007.tar.gz", []string{"scans.js", "browser-request.js", "docs.js", "mcp.js"})
				if err != nil {
					return err
				}
				reference = toolsReferenceRenderer(reference, ui, apiProxy, scripts)
			}
			if referenceAnalysis {
				scripts, err := archivedComponentScripts(".data/source-archives/browser-before-ts-pipeline-20261007.tar.gz", []string{"statistics.js", "rankings.js", "graph.js", "graph-render.js", "formatters.js"})
				if err != nil {
					return err
				}
				reference = analysisReferenceRenderer(reference, ui, scripts)
			}
			if referenceStatus {
				scripts, err := archivedComponentScripts(".data/source-archives/browser-before-ts-pipeline-20261007.tar.gz", []string{"status.js"})
				if err != nil {
					return err
				}
				reference = statusReferenceRenderer(reference, ui, scripts)
			}
			if referenceLegacy {
				scripts, err := archivedComponentScripts(".data/source-archives/browser-before-ts-pipeline-20261007.tar.gz", []string{"legacy.js", "fragment.js"})
				if err != nil {
					return err
				}
				reference = legacyReferenceRenderer(reference, ui, scripts)
			}
			if referenceUniverse {
				scripts, err := archivedComponentScripts(".data/source-archives/browser-before-ts-pipeline-20261007.tar.gz", []string{"market.js", "universe.js", "system.js", "fragment.js", "default-tab.js", "browser-request.js"})
				if err != nil {
					return err
				}
				reference = universeReferenceRenderer(reference, ui, scripts)
			}
			source := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if fixtureViewportPage(w, r) || fixtureAudioTrace(w, r) || fixtureEntitySearch(w, r) {
					return
				}
				// Current rendering reads public map geography; the archived
				// controller must use that same anonymous, read-only snapshot.
				if strings.HasPrefix(r.URL.Path, "/api/map/") {
					battleAPI.ServeHTTP(w, r)
					return
				}
				reference.ServeHTTP(w, r)
			})
			servers = append(servers, &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", referencePort), Handler: wrap(source), ReadHeaderTimeout: 10 * time.Second})
		}
		return serve(ctx, servers)
	}
	publicURL, err := localURL(public)
	if err != nil {
		return err
	}
	dsn := os.Getenv("TEST_DATABASE_URL")
	cfg, err := pgxpool.ParseConfig(dsn)
	if dsn == "" || err != nil || (cfg.ConnConfig.Host != "127.0.0.1" && cfg.ConnConfig.Host != "localhost") {
		return errors.New("TEST_DATABASE_URL must explicitly identify the local PostgreSQL test server")
	}
	admin, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer admin.Close()
	var random [6]byte
	if _, err = rand.Read(random[:]); err != nil {
		return err
	}
	name := "shrike_frontend_parity_" + hex.EncodeToString(random[:])
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		return err
	}
	defer func() {
		clean, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if _, err := admin.Exec(clean, "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			log.Printf("fixture cleanup failed: %v", err)
		} else {
			log.Print("Disposable frontend database removed")
		}
	}()
	child := cfg.Copy()
	child.ConnConfig.Database = name
	pool, err := pgxpool.NewWithConfig(ctx, child)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err = migrate.Apply(ctx, pool); err != nil {
		return err
	}
	accounts := map[string]account{"owner": {4242, "Fixture Owner", uuid.NewString(), true}, "reader": {4343, "Fixture Reader", uuid.NewString(), false}}
	assetStorage := &fixtureDomainAssets{}
	options := api.Options{DB: &fixtureDB{pool}, KlipyAPIKey: os.Getenv("KLIPY_API_KEY"), DomainAssets: assetStorage}
	if adminFixtures {
		options.PrimaryPool = pool
	}
	var media *commentMediaFixture
	if mediaFixtures {
		media = newCommentMediaFixture()
		options.KlipyAPIKey = "local-provider-fixture"
		options.Auth.HTTPClient = &http.Client{Transport: media, Timeout: 5 * time.Second}
	}
	service := api.New(options)
	site := service.Site()
	if adminFixtures {
		site = newFixtureAnnouncementRelay().wrap(site)
	}
	if trackerRelay {
		site = newFixtureTrackerRelay().wrap(site)
	}
	if media != nil {
		site = media.handler(site)
	}
	if err = seed(ctx, pool, service.Site(), accounts); err != nil {
		return err
	}
	if err = testfixtures.SeedBattleGenerator(ctx, pool); err != nil {
		return err
	}
	if err = testfixtures.SeedScans(ctx, pool); err != nil {
		return err
	}
	if err = testfixtures.SeedFittings(ctx, pool); err != nil {
		return err
	}
	if err = seedSettings(ctx, pool); err != nil {
		return err
	}
	if blogFixtures {
		if err = seedBlogFixtures(ctx, service.Site(), accounts["owner"]); err != nil {
			return err
		}
	}
	if adminFixtures {
		if err = seedAdminFixtures(ctx, pool); err != nil {
			return err
		}
		if err = seedAdminRiverFixtures(ctx, pool); err != nil {
			return err
		}
		if err = seedAdminWalletFixtures(ctx, pool); err != nil {
			return err
		}
		if err = seedAdminDomainFixtures(ctx, pool, assetStorage); err != nil {
			return err
		}
		if err = seedAdminCampaignFixtures(ctx, pool); err != nil {
			return err
		}
		if err = seedAdminReviewFixtures(ctx, pool); err != nil {
			return err
		}
	}
	publicProxy := anonymousFixtureProxy(publicURL)
	baseRewrite := publicProxy.Rewrite
	publicProxy.Rewrite = func(r *httputil.ProxyRequest) {
		baseRewrite(r)
		// The comparison hostname isolates cookies, but is not a published
		// custom board. Bootstrap both renderers with the global public site.
		if r.Out.URL.Path == "/api/site" {
			r.Out.Host = publicURL.Host
			r.Out.Header.Del("X-Forwarded-Host")
		}
	}
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fixtureBlogCover(w, r) {
			return
		}
		if fixtureViewportPage(w, r) {
			return
		}
		if trackerRelay && (r.URL.Path == "/ws/killlist" || r.URL.Path == "/__fixture/tracker-relay") {
			site.ServeHTTP(w, r)
			return
		}
		if adminFixtures && (r.URL.Path == "/ws/announcements" || r.URL.Path == "/__fixture/announcement-relay") {
			site.ServeHTTP(w, r)
			return
		}
		if media != nil && (r.URL.Path == "/ws/comments" || strings.HasPrefix(r.URL.Path, "/__fixture/media/")) {
			site.ServeHTTP(w, r)
			return
		}
		if fixtureEntityImage(w, r) {
			return
		}
		if strings.HasPrefix(r.URL.Path, "/__fixture") {
			w.Header().Set("Cache-Control", "private, no-store")
			if r.URL.Path == "/__fixture/account" {
				selected, on := accounts[r.URL.Query().Get("as")]
				age := -1
				hint := ""
				if on {
					age = 86400
					hint = url.PathEscape(fmt.Sprintf("%d:%s", selected.id, selected.name))
				}
				// #nosec G124 -- disposable session on loopback HTTP; Secure would prevent fixture login.
				http.SetCookie(w, &http.Cookie{Name: "evelogin", Value: selected.session, Path: "/", MaxAge: age, HttpOnly: true, SameSite: http.SameSiteLaxMode})
				// #nosec G124 -- browser-readable identity hint on loopback HTTP, not an authentication credential.
				http.SetCookie(w, &http.Cookie{Name: "ek_auth", Value: hint, Path: "/", MaxAge: age, SameSite: http.SameSiteLaxMode})
				destination := r.URL.Query().Get("returnTo")
				if !strings.HasPrefix(destination, "/") || strings.HasPrefix(destination, "//") || strings.ContainsAny(destination, "\\\r\n") {
					destination = "/__fixture"
				}
				// #nosec G710 -- validated local path above; rejects authorities, backslashes and line breaks.
				http.Redirect(w, r, destination, http.StatusSeeOther)
				return
			}
			_, _ = fmt.Fprint(w, "<h1>Disposable frontend fixture</h1><p><a href='/__fixture/account?as=owner'>Owner account</a> · <a href='/__fixture/account?as=reader'>Reader account</a> · <a href='/__fixture/account?as=anonymous'>Anonymous</a></p><ul>")
			for _, entry := range cases {
				_, _ = fmt.Fprintf(w, "<li><a href='/campaign/%s'>%s</a></li>", entry.id, entry.name)
			}
			_, _ = fmt.Fprint(w, "</ul>")
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/battle/generator/") {
			site.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/battle/") {
			id := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/battle/"), "/")[0]
			var found bool
			if err := pool.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM battles WHERE battle_id::text=$1)", id).Scan(&found); err == nil && found {
				site.ServeHTTP(w, r)
				return
			}
		}
		if fixtureEntitySearch(w, r) {
			return
		}
		publicRead := r.URL.Path == "/api/search" || r.URL.Path == "/api/site" || (r.URL.Path == "/api/announcements" && !adminFixtures) || r.URL.Path == "/api/conflicts/battles" || strings.HasPrefix(r.URL.Path, "/api/battle/") || strings.HasPrefix(r.URL.Path, "/images/") || strings.HasPrefix(r.URL.Path, "/ws/")
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/campaign/") {
			id := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/campaign/"), "/")[0]
			var found bool
			if err := pool.QueryRow(r.Context(), "SELECT EXISTS (SELECT 1 FROM campaigns WHERE campaign_id=$1)", id).Scan(&found); err == nil && !found {
				publicRead = true
			}
		}
		if publicRead {
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				http.Error(w, "Fixture public proxy is read-only", http.StatusMethodNotAllowed)
				return
			}
			publicProxy.ServeHTTP(w, r)
			return
		}
		site.ServeHTTP(w, r)
	})
	ui := fixtureFrameHandler(frontend.New(frontend.Options{CampaignsAPI: apiHandler, ConflictsAPI: apiHandler, FittingsAPI: apiHandler, BattleGenerator: service.Homepage(), Statistics: service.Homepage(), Scans: service.Homepage(), RoamReports: service.Homepage(), Killmail: service.Homepage(), CommentsAPI: service.Site(), Sessions: service.Session(), AccountAPI: faults.wrap(apiHandler), Blog: service.Homepage(), BlogPreviewAPI: site}))
	renderUI := fixtureGlobalSite(ui)
	goHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if backendPath(r.URL.Path) {
			apiHandler.ServeHTTP(w, r)
		} else {
			renderUI.ServeHTTP(w, r)
		}
	})
	log.Printf("Fixture ready: %d Go/API; use frontend-parity.localhost to isolate test cookies", fixturePort)
	return serve(ctx, []*http.Server{{Addr: fmt.Sprintf("127.0.0.1:%d", fixturePort), Handler: faults.wrap(goHandler)}})
}

// Keep the documented cookie-isolation host on the global frontend rather
// than treating it as a custom killboard. API origins and sessions are intact.
func fixtureGlobalSite(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.EqualFold(r.URL.Hostname(), "frontend-parity.localhost") || strings.HasPrefix(strings.ToLower(r.Host), "frontend-parity.localhost:") || strings.EqualFold(r.Host, "frontend-parity.localhost") {
			r = r.Clone(r.Context())
			r.Host = "localhost"
		}
		next.ServeHTTP(w, r)
	})
}

var fixtureEntityImagePattern = regexp.MustCompile(`^/images/(characters|corporations|alliances)/(\d+)/(portrait|logo)$`)

func fixtureEntityImage(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	if match := fixtureEntityImagePattern.FindStringSubmatch(r.URL.Path); match != nil {
		query := r.URL.Query()
		// The old local image service resized tiny badges. The public CDN
		// supplies a supported source image; the original CSS retains its size.
		if size, _ := strconv.Atoi(query.Get("size")); size < 64 {
			query.Set("size", "64")
		}
		// #nosec G710 -- fixed CDN origin, regex-validated path and URL-encoded query.
		http.Redirect(w, r, "https://images.evetech.net/"+match[1]+"/"+match[2]+"/"+match[3]+"?"+query.Encode(), http.StatusTemporaryRedirect)
		return true
	}
	return false
}

func backendPath(path string) bool {
	for _, prefix := range []string{"/api/", "/auth/", "/images/", "/ws/", "/_fragments/fits/variants/", "/__fixture"} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func serve(ctx context.Context, servers []*http.Server) error {
	errors := make(chan error, len(servers))
	for _, server := range servers {
		if server.ReadHeaderTimeout == 0 {
			server.ReadHeaderTimeout = 10 * time.Second
		}
		go func() { errors <- server.ListenAndServe() }()
	}
	var result error
	select {
	case <-ctx.Done():
	case result = <-errors:
		if errors2 := http.ErrServerClosed; result == errors2 {
			result = nil
		}
	}
	clean, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Close both listeners immediately; a pending render on one comparison
	// port must not leave the other port occupied during a restart.
	var shutdown sync.WaitGroup
	for _, server := range servers {
		shutdown.Add(1)
		go func() {
			defer shutdown.Done()
			_ = server.Shutdown(clean)
		}()
	}
	shutdown.Wait()
	return result
}

type remoteSession struct{ origin string }

func (s remoteSession) BrowserSession(ctx context.Context, cookie string) (*api.Principal, error) {
	if cookie == "" {
		return nil, nil
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, s.origin+"/auth/me", nil)
	if err != nil {
		return nil, err
	}
	// #nosec G124 -- outgoing request cookie, not a browser Set-Cookie policy.
	r.AddCookie(&http.Cookie{Name: "evelogin", Value: cookie})
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(r)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("fixture session returned %d", response.StatusCode)
	}
	var data struct {
		User *struct {
			ID    int32  `json:"characterId"`
			Name  string `json:"characterName"`
			Admin bool   `json:"isAdmin"`
		} `json:"user"`
	}
	if err = json.NewDecoder(response.Body).Decode(&data); err != nil {
		return nil, err
	}
	if data.User == nil {
		return nil, nil
	}
	return &api.Principal{CharacterID: data.User.ID, CharacterName: data.User.Name, IsAdmin: data.User.Admin}, nil
}

func seed(ctx context.Context, pool *pgxpool.Pool, handler http.Handler, accounts map[string]account) error {
	for _, sql := range []string{
		`INSERT INTO corporations (corporation_id,name,ticker,palette) VALUES (10,'Fixture Red','RED','{"main_color":"#e74c3c"}'),(11,'Fixture Blue','BLUE','{"main_color":"#3498db"}'),(12,'Fixture Green','GREEN',NULL),(13,'Fixture Orange','ORANGE',NULL)`,
		`INSERT INTO characters (character_id,name,corporation_id) VALUES (4242,'Fixture Owner',10),(4343,'Fixture Reader',11),(4444,'Fixture Third',12)`,
		`INSERT INTO regions (region_id,name) VALUES (10000002,'The Forge')`,
		`INSERT INTO solar_systems (solar_system_id,system_name,region_id,security) VALUES (30000142,'Jita',10000002,0.9)`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			return err
		}
	}
	for _, a := range accounts {
		if _, err := pool.Exec(ctx, `INSERT INTO users (character_id,character_name,character_owner_hash,session_id,is_admin) VALUES ($1,$2,'fixture',$3,$4)`, a.id, a.name, a.session, a.admin); err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, `INSERT INTO user_sessions (session_id,character_id,expires_at) VALUES ($1,$2,now()+interval '24 hours')`, a.session, a.id); err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, `INSERT INTO ek_wallet_accounts (character_id,balance,reserved_balance) VALUES ($1,100000.50,1000)`, a.id); err != nil {
			return err
		}
	}
	pilot := map[string]any{"characterId": 4242, "name": "Fixture Owner", "shipName": "Monitor", "kills": 100, "losses": 3, "iskDestroyed": 25000000000, "corporationTicker": "RED", "corporationName": "Fixture Red", "damage": 1230000}
	intel := map[string]any{"fcs": []any{pilot}, "logistics": []any{pilot}, "capitals": []any{pilot}, "logisticsCount": 20, "capitalsCount": 18}
	stats := map[string]any{"totals": map[string]any{"killCount": 120, "iskDestroyed": 30000000000, "charactersInvolved": 42, "corporationsInvolved": 4, "alliancesInvolved": 2}, "topKillersOverall": []any{pilot}, "topVictimsOverall": []any{pilot}, "topKillersBySide": map[string]any{"0": []any{pilot}, "1": []any{pilot}, "2": []any{pilot}, "3": []any{pilot}}, "shipClassesBySide": map[string]any{}, "shipClassesOverall": []any{}, "topSystems": []any{}, "mostValuable": []any{}, "intelBySide": map[string]any{"0": intel, "1": intel, "2": intel, "3": intel}}
	encoded, _ := json.Marshal(stats)
	now := time.Now().UTC().Truncate(time.Second)
	for _, entry := range cases {
		body := string(encoded)
		if entry.sides == 0 {
			body = `{"totals":{},"topKillersOverall":[],"topVictimsOverall":[],"shipClassesOverall":[],"topSystems":[],"mostValuable":[]}`
		}
		if _, err := pool.Exec(ctx, `INSERT INTO campaigns (campaign_id,name,description,created_by_character_id,status,start_time,end_time,location,stats,processing_paused,processing_note) VALUES ($1,$2,'Disposable browser comparison data',4242,1,$3,$4,'{"systemIds":[30000142]}',$5::jsonb,$6,$7)`, entry.id, entry.name, now.Add(-30*24*time.Hour), now.Add(48*time.Hour), body, entry.paused, "Fixture workload limit"); err != nil {
			return err
		}
		for index := range entry.sides {
			if _, err := pool.Exec(ctx, `INSERT INTO campaign_sides (campaign_id,side_index,name,kills,losses,isk_destroyed,isk_lost) VALUES ($1,$2,$3,100,20,25000000000,5000000000)`, entry.id, index, fmt.Sprintf("Side %d", index+1)); err != nil {
				return err
			}
			if _, err := pool.Exec(ctx, `INSERT INTO campaign_side_entities (campaign_id,side_index,entity_type,entity_id,kills,losses) VALUES ($1,$2,1,$3,100,20)`, entry.id, index, index+10); err != nil {
				return err
			}
			for day := range 6 {
				if _, err := pool.Exec(ctx, `INSERT INTO campaign_stats_daily (campaign_id,side_index,day,kills,losses,isk_destroyed,isk_lost) VALUES ($1,$2,$3,20,5,1000000000,250000000)`, entry.id, index, now.AddDate(0, 0, -6+day)); err != nil {
					return err
				}
			}
		}
		if !entry.prize {
			continue
		}
		if _, err := pool.Exec(ctx, `INSERT INTO campaign_prize_pools (campaign_id,metric,winner_count,payout_percentages) VALUES ($1,0,3,'{70,20,10}')`, entry.id); err != nil {
			return err
		}
		contribution, _ := json.Marshal(map[string]any{"amount": "1000.00", "requestId": uuid.NewString()})
		request := httptest.NewRequest(http.MethodPost, "http://fixture.localhost/api/campaign/"+entry.id+"/prize/contribute", bytes.NewReader(contribution))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", "http://fixture.localhost")
		// #nosec G124 -- outgoing request cookie, not a browser Set-Cookie policy.
		request.AddCookie(&http.Cookie{Name: "evelogin", Value: accounts["owner"].session})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != 200 {
			return fmt.Errorf("seed contribution: %d %s", response.Code, response.Body.String())
		}
		if entry.id == "PrizeWait00001" || entry.id == "PrizeReady0001" {
			hours := -12
			if entry.id == "PrizeReady0001" {
				hours = -48
			}
			if _, err := pool.Exec(ctx, `UPDATE campaigns SET end_time=$2 WHERE campaign_id=$1`, entry.id, now.Add(time.Duration(hours)*time.Hour)); err != nil {
				return err
			}
		}
		if entry.id == "PrizeReady0001" {
			if _, err := pool.Exec(ctx, `UPDATE campaign_prize_pools SET status=1,finalized_at=now() WHERE campaign_id=$1`, entry.id); err != nil {
				return err
			}
		}
		for rank, id := range []int{4242, 4343, 4444} {
			if _, err := pool.Exec(ctx, `INSERT INTO campaign_prize_results (campaign_id,rank,character_id,character_name,metric_value,secondary_value,payout_percentage,payout_amount) VALUES ($1,$2,$3,$4,$5,1000,$6,$7)`, entry.id, rank+1, id, fmt.Sprintf("Fixture winner %d", rank+1), 100-rank*30, []int{70, 20, 10}[rank], []int{700, 200, 100}[rank]); err != nil {
				return err
			}
		}
	}
	return nil
}
