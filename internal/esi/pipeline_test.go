package esi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"
)

// End-to-end tests through the whole pipeline, with a fake ESI standing in for
// CCP. What each asserts is not "did I get data" — that would pass with the
// coordination ripped out — but what reached the server, and what state was left
// in Redis afterwards.

type character struct {
	Name          string `json:"name"`
	CorporationID int32  `json:"corporation_id"`
}

func TestCachedResponseSkipsESIEntirely(t *testing.T) {
	rdb := testRedis(t)
	fake := newFakeESI(t, func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"name":"Rifter Pilot","corporation_id":98187159}`, time.Now().Add(time.Hour))
	})
	c := testClient(t, rdb, fake.URL)
	ctx := context.Background()

	first, err := Get[character](ctx, c, "/latest/characters/1000/")
	if err != nil {
		t.Fatal(err)
	}
	if !first.OK() || first.Data.Name != "Rifter Pilot" {
		t.Fatalf("first response: %+v", first)
	}
	if first.Cached {
		t.Error("the first call cannot have been cached")
	}

	second, err := Get[character](ctx, c, "/latest/characters/1000/")
	if err != nil {
		t.Fatal(err)
	}
	if !second.OK() || second.Data.Name != "Rifter Pilot" {
		t.Fatalf("second response: %+v", second)
	}
	if !second.Cached {
		t.Error("the second call was not served from cache")
	}

	if fake.Hits() != 1 {
		t.Errorf("ESI was contacted %d times for a cacheable resource", fake.Hits())
	}
}

// Ten workers asking for the same character at the same moment must produce one
// ESI request. This is the property that keeps a fleet fight from turning into
// hundreds of identical calls.
func TestSingleflightCollapsesConcurrentRequests(t *testing.T) {
	rdb := testRedis(t)
	fake := newFakeESI(t, func(w http.ResponseWriter, _ *http.Request) {
		// Long enough that every caller is inside the window together.
		time.Sleep(250 * time.Millisecond)
		jsonOK(w, `{"name":"Contended","corporation_id":1}`, time.Now().Add(time.Hour))
	})
	c := testClient(t, rdb, fake.URL)

	const callers = 10
	var wg sync.WaitGroup
	var ok counter
	results := make([]Response[character], callers)

	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			res, err := Get[character](context.Background(), c, "/latest/characters/2000/")
			if err == nil && res.OK() {
				ok.inc()
			}
			results[i] = res
		}(i)
	}
	wg.Wait()

	if ok.get() != callers {
		t.Errorf("%d of %d callers got data", ok.get(), callers)
	}
	if fake.Hits() != 1 {
		t.Errorf("ESI was contacted %d times; singleflight should have collapsed these to 1", fake.Hits())
	}
	for i, res := range results {
		if res.Data == nil || res.Data.Name != "Contended" {
			t.Errorf("caller %d got %+v", i, res.Data)
		}
	}
}

// An expired entry with an ETag must produce a conditional request, and a 304
// must return the body we already held rather than nothing.
func TestConditionalRequestOn304(t *testing.T) {
	rdb := testRedis(t)

	var sawETag string
	fake := newFakeESI(t, func(w http.ResponseWriter, r *http.Request) {
		sawETag = r.Header.Get("If-None-Match")
		w.Header().Set("Expires", time.Now().Add(time.Hour).UTC().Format(http.TimeFormat))
		w.Header().Set("ETag", `W/"v2"`)
		w.WriteHeader(http.StatusNotModified)
	})
	c := testClient(t, rdb, fake.URL)
	ctx := context.Background()

	url := c.fullURL("/latest/characters/3000/")
	c.cache.Set(ctx, url, &Entry{
		Data:    json.RawMessage(`{"name":"Unchanged","corporation_id":7}`),
		Status:  200,
		Expires: time.Now().Add(-time.Minute).UnixMilli(),
		ETag:    `W/"v1"`,
	})

	res, err := Get[character](ctx, c, "/latest/characters/3000/")
	if err != nil {
		t.Fatal(err)
	}

	if sawETag != `W/"v1"` {
		t.Errorf("If-None-Match was %q — the stale entry's etag was not used", sawETag)
	}
	if res.Status != http.StatusNotModified {
		t.Errorf("status = %d, want 304", res.Status)
	}
	if !res.OK() {
		t.Error("a 304 response backed by cached data must be successful")
	}
	if res.Data == nil || res.Data.Name != "Unchanged" {
		t.Fatalf("a 304 must serve the cached body, got %+v", res.Data)
	}

	// And the refreshed entry must now be fresh, so the next call skips ESI.
	if entry := c.cache.Get(ctx, url); !entry.Fresh(time.Now()) {
		t.Error("the 304 did not extend the cached entry")
	}
}

// A 420 is a statement about the whole application, not one endpoint, so it must
// pause everything.
func TestErrorLimitPausesTheClient(t *testing.T) {
	rdb := testRedis(t)
	fake := newFakeESI(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("x-esi-error-limit-reset", "42")
		w.WriteHeader(statusErrorLimited)
	})
	c := testClient(t, rdb, fake.URL)
	ctx := context.Background()

	res, err := Get[character](ctx, c, "/latest/characters/4000/")
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != statusErrorLimited {
		t.Errorf("status = %d, want 420", res.Status)
	}
	if res.RetryAfter != 42 {
		t.Errorf("retry after = %d, want 42", res.RetryAfter)
	}

	remaining := c.pauseRemaining(ctx)
	if remaining <= 0 {
		t.Fatal("a 420 did not pause the client")
	}
	if remaining > 43*time.Second {
		t.Errorf("pause of %v exceeds what ESI asked for", remaining)
	}
}

func TestTooManyRequestsReportsRetryAfter(t *testing.T) {
	rdb := testRedis(t)
	fake := newFakeESI(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("retry-after", "17")
		w.WriteHeader(http.StatusTooManyRequests)
	})
	c := testClient(t, rdb, fake.URL)

	res, err := Get[character](context.Background(), c, "/latest/characters/4100/")
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != http.StatusTooManyRequests {
		t.Errorf("status = %d", res.Status)
	}
	if res.RetryAfter != 17 {
		t.Errorf("retry after = %d, want 17", res.RetryAfter)
	}
}

// A 5xx is ESI being unwell; retrying is right. A 4xx is an answer; retrying
// spends global error budget to be told the same thing again.
func TestServerErrorsRetryAndClientErrorsDoNot(t *testing.T) {
	t.Run("5xx is retried", func(t *testing.T) {
		rdb := testRedis(t)
		var attempts counter
		fake := newFakeESI(t, func(w http.ResponseWriter, _ *http.Request) {
			if attempts.get() < 2 {
				attempts.inc()
				w.WriteHeader(http.StatusBadGateway)
				return
			}
			attempts.inc()
			jsonOK(w, `{"name":"Recovered","corporation_id":1}`, time.Now().Add(time.Hour))
		})
		c := testClient(t, rdb, fake.URL)

		res, err := Get[character](context.Background(), c, "/latest/characters/4200/")
		if err != nil {
			t.Fatal(err)
		}
		if !res.OK() || res.Data.Name != "Recovered" {
			t.Fatalf("the retry did not recover: %+v", res)
		}
		if fake.Hits() != 3 {
			t.Errorf("server saw %d requests, want 3 (two failures then a success)", fake.Hits())
		}
	})

	t.Run("404 is not retried", func(t *testing.T) {
		rdb := testRedis(t)
		fake := newFakeESI(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		c := testClient(t, rdb, fake.URL)

		res, err := Get[character](context.Background(), c, "/latest/characters/4300/")
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != http.StatusNotFound {
			t.Errorf("status = %d", res.Status)
		}
		if !res.Permanent() {
			t.Error("a 404 should be classified permanent")
		}
		if fake.Hits() != 1 {
			t.Errorf("a 404 was retried %d times", fake.Hits()-1)
		}
	})
}

// A non-200 must never be cached, or one 404 poisons an id for as long as the
// entry lives.
func TestErrorsAreNotCached(t *testing.T) {
	rdb := testRedis(t)
	fake := newFakeESI(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Expires", time.Now().Add(time.Hour).UTC().Format(http.TimeFormat))
		w.WriteHeader(http.StatusNotFound)
	})
	c := testClient(t, rdb, fake.URL)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		if _, err := Get[character](ctx, c, "/latest/characters/4400/"); err != nil {
			t.Fatal(err)
		}
	}
	if fake.Hits() != 2 {
		t.Errorf("a 404 was cached: server saw %d of 2 requests", fake.Hits())
	}
}

// The response's own accounting has to reach the bucket, or the cluster paces
// itself against a preset while ESI is counting something else.
func TestRateLimitHeadersReachTheBucket(t *testing.T) {
	rdb := testRedis(t)
	fake := newFakeESI(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("x-ratelimit-remaining", "17")
		w.Header().Set("x-ratelimit-reset", "30")
		jsonOK(w, `{"name":"Metered","corporation_id":1}`, time.Now().Add(time.Hour))
	})
	c := testClient(t, rdb, fake.URL)
	ctx := context.Background()

	if _, err := Get[character](ctx, c, "/latest/characters/4500/"); err != nil {
		t.Fatal(err)
	}

	state, err := c.limiter.Peek(ctx, Groups["characters"])
	if err != nil {
		t.Fatal(err)
	}
	if state.Remaining != 17 {
		t.Errorf("bucket remaining = %d, want the 17 ESI reported", state.Remaining)
	}
}

// Killmails report remaining with no reset. The counter must still land.
func TestRemainingWithoutResetReachesTheBucket(t *testing.T) {
	rdb := testRedis(t)
	fake := newFakeESI(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("x-ratelimit-remaining", "2500")
		jsonOK(w, `{"killmail_id":1}`, time.Now().Add(time.Hour))
	})
	c := testClient(t, rdb, fake.URL)
	ctx := context.Background()

	path := KillmailPath(137258027, "1d9365aaed385213867e40390d29cd4c7596e0e3")
	if _, err := Get[json.RawMessage](ctx, c, path); err != nil {
		t.Fatal(err)
	}

	state, err := c.limiter.Peek(ctx, Groups["killmail"])
	if err != nil {
		t.Fatal(err)
	}
	if state.Remaining != 2500 {
		t.Errorf("bucket remaining = %d, want 2500", state.Remaining)
	}
}

func TestErrorBudgetHeadersAreRecorded(t *testing.T) {
	rdb := testRedis(t)
	fake := newFakeESI(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("x-esi-error-limit-remain", "55")
		w.Header().Set("x-esi-error-limit-reset", "30")
		jsonOK(w, `{"name":"Erring","corporation_id":1}`, time.Now().Add(time.Hour))
	})
	c := testClient(t, rdb, fake.URL)
	ctx := context.Background()

	if _, err := Get[character](ctx, c, "/latest/characters/4600/"); err != nil {
		t.Fatal(err)
	}

	remain, err := rdb.Get(ctx, keyErrorRemaining).Result()
	if err != nil {
		t.Fatalf("error budget was not recorded: %v", err)
	}
	if remain != "55" {
		t.Errorf("error remaining = %s, want 55", remain)
	}

	// A healthy budget must not add latency; a spent one must.
	if delay := c.errorBudgetDelay(ctx); delay <= 0 {
		t.Error("a budget of 55 should be pacing requests")
	}
	if err := rdb.Set(ctx, keyErrorRemaining, "100", 0).Err(); err != nil {
		t.Fatal(err)
	}
	if delay := c.errorBudgetDelay(ctx); delay != 0 {
		t.Errorf("a healthy budget added %v of delay", delay)
	}
}

// CCP requires an identifying user agent, and the compatibility date is what
// pins the response schema — a missing one means CCP's newest shape arrives
// unannounced.
func TestRequestHeaders(t *testing.T) {
	rdb := testRedis(t)
	var agent, compat, accept string
	fake := newFakeESI(t, func(w http.ResponseWriter, r *http.Request) {
		agent = r.Header.Get("User-Agent")
		compat = r.Header.Get("X-Compatibility-Date")
		accept = r.Header.Get("Accept")
		jsonOK(w, `{"name":"Headers","corporation_id":1}`, time.Now().Add(time.Hour))
	})
	c := testClient(t, rdb, fake.URL)

	if _, err := Get[character](context.Background(), c, "/latest/characters/4700/"); err != nil {
		t.Fatal(err)
	}

	if agent != "shrike-test/1.0" {
		t.Errorf("user agent = %q", agent)
	}
	if accept != "application/json" {
		t.Errorf("accept = %q", accept)
	}
	if _, err := time.Parse("2006-01-02", compat); err != nil {
		t.Errorf("compatibility date %q is not a date", compat)
	}
	// A week back, so a schema change lands in our lap a week after it ships.
	if compat >= time.Now().UTC().Format("2006-01-02") {
		t.Errorf("compatibility date %q is not in the past", compat)
	}
}

func TestPostSendsBodyAndDecodesResponse(t *testing.T) {
	rdb := testRedis(t)
	var method string
	var body []byte
	fake := newFakeESI(t, func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		body, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"character_id":1,"corporation_id":2,"alliance_id":3}]`))
	})
	c := testClient(t, rdb, fake.URL)

	res, err := Post[[]Affiliation](context.Background(), c, "/latest/characters/affiliation/", []int32{1})
	if err != nil {
		t.Fatal(err)
	}
	if method != http.MethodPost {
		t.Errorf("method = %s", method)
	}
	if string(body) != `[1]` {
		t.Errorf("body = %s", body)
	}
	if res.Data == nil || len(*res.Data) != 1 {
		t.Fatalf("response = %+v", res.Data)
	}
	if (*res.Data)[0].AllianceID != 3 {
		t.Errorf("decoded %+v", (*res.Data)[0])
	}
}

// Authenticated responses are per-character. Caching them by URL would hand one
// character another's data.
func TestAuthenticatedRequestsCarryTokenAndAreNotCached(t *testing.T) {
	rdb := testRedis(t)
	var auth string
	fake := newFakeESI(t, func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		w.Header().Set("x-pages", "3")
		jsonOK(w, `{"name":"Private","corporation_id":1}`, time.Now().Add(time.Hour))
	})
	c := testClient(t, rdb, fake.URL)
	ctx := context.Background()

	res, err := GetAuthenticated[character](ctx, c, "/latest/characters/4800/", "token-abc")
	if err != nil {
		t.Fatal(err)
	}
	if auth != "Bearer token-abc" {
		t.Errorf("authorization = %q", auth)
	}
	if res.Pages != 3 {
		t.Errorf("pages = %d, want 3", res.Pages)
	}

	if _, err := GetAuthenticated[character](ctx, c, "/latest/characters/4800/", "token-abc"); err != nil {
		t.Fatal(err)
	}
	if fake.Hits() != 2 {
		t.Errorf("an authenticated response was cached: server saw %d of 2 requests", fake.Hits())
	}
}

func TestGetAllPagesWalksToTheEnd(t *testing.T) {
	rdb := testRedis(t)
	fake := newFakeESI(t, func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Expires", time.Now().Add(time.Hour).UTC().Format(http.TimeFormat))

		switch page {
		case 1:
			// A full page means "there may be more".
			full := make([]int32, pageSize)
			for i := range full {
				full[i] = int32(i)
			}
			_ = json.NewEncoder(w).Encode(full)
		default:
			_ = json.NewEncoder(w).Encode([]int32{9001, 9002})
		}
	})
	c := testClient(t, rdb, fake.URL)

	all, res, err := GetAllPages[int32](context.Background(), c, "/latest/alliances/", "")
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK() {
		t.Fatalf("status = %d", res.Status)
	}
	if len(all) != pageSize+2 {
		t.Errorf("collected %d items, want %d", len(all), pageSize+2)
	}
	if fake.Hits() != 2 {
		t.Errorf("server saw %d requests, want 2", fake.Hits())
	}
	if all[len(all)-1] != 9002 {
		t.Errorf("last item = %d", all[len(all)-1])
	}
}

// A short first page ends the walk immediately; asking for page 2 would be a
// wasted request on every list in the game.
func TestGetAllPagesStopsOnShortFirstPage(t *testing.T) {
	rdb := testRedis(t)
	fake := newFakeESI(t, func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `[1,2,3]`, time.Now().Add(time.Hour))
	})
	c := testClient(t, rdb, fake.URL)

	all, _, err := GetAllPages[int32](context.Background(), c, "/latest/alliances/", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Errorf("collected %d items", len(all))
	}
	if fake.Hits() != 1 {
		t.Errorf("server saw %d requests, want 1", fake.Hits())
	}
}

// A cancelled context must abandon the request rather than run the pipeline's
// retries to completion.
func TestCancellationStopsThePipeline(t *testing.T) {
	rdb := testRedis(t)
	fake := newFakeESI(t, func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(600 * time.Millisecond)
		jsonOK(w, `{"name":"Slow","corporation_id":1}`, time.Now().Add(time.Hour))
	})
	c := testClient(t, rdb, fake.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, _ = Get[character](ctx, c, "/latest/characters/4900/")
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("kept going for %v after the context was cancelled", elapsed)
	}
}

// A drained bucket must not be circumvented: with no tokens and no time to wait,
// the pipeline reports a retryable outcome rather than calling ESI anyway.
func TestExhaustedBucketBlocksTheRequest(t *testing.T) {
	rdb := testRedis(t)
	fake := newFakeESI(t, func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"name":"ShouldNotHappen","corporation_id":1}`, time.Now().Add(time.Hour))
	})
	c := testClient(t, rdb, fake.URL)
	// Every attempt sleeps the cap before retrying, which is a real minute at
	// production settings. The property under test is unaffected by how long
	// each sleep is, so it is shortened.
	c.maxWait = 20 * time.Millisecond
	ctx := context.Background()

	// Drain the characters bucket and leave the window wide open, so waiting
	// cannot help within the pipeline's retry budget.
	g := Groups["characters"]
	if err := rdb.Set(ctx, keyRemaining(g.Name), "0", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if err := rdb.Set(ctx, keyResetAt(g.Name),
		strconv.FormatInt(time.Now().Add(30*time.Minute).UnixMilli(), 10), time.Hour).Err(); err != nil {
		t.Fatal(err)
	}

	res, err := Get[character](ctx, c, "/latest/characters/5000/")
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != 0 {
		t.Errorf("status = %d, want 0 (never reached ESI)", res.Status)
	}
	if fake.Hits() != 0 {
		t.Errorf("the request went out despite an empty bucket (%d hits)", fake.Hits())
	}
}

// The server status decides whether a worker fleet runs at all, so both answers
// have to be legible: up with a player count, and down without one.
func TestFetchStatus(t *testing.T) {
	t.Run("tranquility is up", func(t *testing.T) {
		rdb := testRedis(t)
		fake := newFakeESI(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/latest/status/" {
				t.Errorf("requested %q", r.URL.Path)
			}
			if got := r.URL.Query().Get("datasource"); got != "tranquility" {
				t.Errorf("datasource = %q, want tranquility", got)
			}
			jsonOK(w, `{"players":28451,"server_version":"2847362","start_time":"2026-07-26T11:00:11Z"}`,
				time.Now().Add(30*time.Second))
		})
		c := testClient(t, rdb, fake.URL)

		res, err := FetchStatus(context.Background(), c)
		if err != nil {
			t.Fatal(err)
		}
		if !res.OK() {
			t.Fatalf("status = %d", res.Status)
		}
		if res.Data.Players != 28451 {
			t.Errorf("players = %d", res.Data.Players)
		}
		if res.Data.VIP {
			t.Error("a normal window was read as VIP")
		}
	})

	t.Run("tranquility is down", func(t *testing.T) {
		rdb := testRedis(t)
		fake := newFakeESI(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		})
		c := testClient(t, rdb, fake.URL)

		res, err := FetchStatus(context.Background(), c)
		if err != nil {
			t.Fatal(err)
		}
		if res.OK() {
			t.Error("a 503 was read as the server being up")
		}
		// Downtime is not permanent — the fleet must come back, not give up.
		if res.Permanent() {
			t.Error("a 503 was classified as permanent")
		}
	})

	// VIP is the trap: the server answers 200 and looks healthy, but only staff
	// can log in, so no killmails will arrive.
	t.Run("vip window", func(t *testing.T) {
		rdb := testRedis(t)
		fake := newFakeESI(t, func(w http.ResponseWriter, _ *http.Request) {
			jsonOK(w, `{"players":12,"server_version":"2847362","vip":true}`, time.Now().Add(30*time.Second))
		})
		c := testClient(t, rdb, fake.URL)

		res, err := FetchStatus(context.Background(), c)
		if err != nil {
			t.Fatal(err)
		}
		if !res.OK() {
			t.Fatalf("status = %d", res.Status)
		}
		if !res.Data.VIP {
			t.Error("a VIP window was not reported")
		}
	})

	t.Run("probe bypasses its own global pause", func(t *testing.T) {
		rdb := testRedis(t)
		fake := newFakeESI(t, func(w http.ResponseWriter, _ *http.Request) {
			jsonOK(w, `{"players":28451,"server_version":"2847362"}`, time.Now().Add(30*time.Second))
		})
		c := testClient(t, rdb, fake.URL)
		ctx := context.Background()

		if err := rdb.Set(ctx, keyPaused, "tq_offline", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}

		callCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
		defer cancel()
		res, err := FetchStatus(callCtx, c)
		if err != nil {
			t.Fatalf("status probe was blocked by the pause it must clear: %v", err)
		}
		if !res.OK() {
			t.Fatalf("status = %d, want 200", res.Status)
		}
		if fake.Hits() != 1 {
			t.Fatalf("ESI received %d probes, want one", fake.Hits())
		}
		if ttl := rdb.PTTL(ctx, keyPaused).Val(); ttl <= 0 {
			t.Error("the probe changed pause state before its result was interpreted")
		}
	})

	t.Run("probe bypasses local rate admission", func(t *testing.T) {
		rdb := testRedis(t)
		fake := newFakeESI(t, func(w http.ResponseWriter, _ *http.Request) {
			jsonOK(w, `{"players":28451,"server_version":"2847362"}`, time.Now().Add(30*time.Second))
		})
		c := testClient(t, rdb, fake.URL)
		ctx := context.Background()
		group := Groups["status"]

		if err := rdb.Set(ctx, keyRemaining(group.Name), "0", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
		if err := rdb.Set(
			ctx,
			keyResetAt(group.Name),
			strconv.FormatInt(time.Now().Add(time.Minute).UnixMilli(), 10),
			time.Minute,
		).Err(); err != nil {
			t.Fatal(err)
		}

		callCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
		defer cancel()
		res, err := FetchStatus(callCtx, c)
		if err != nil {
			t.Fatalf("status probe was blocked by local rate admission: %v", err)
		}
		if !res.OK() || fake.Hits() != 1 {
			t.Fatalf("response = %+v, ESI hits = %d", res, fake.Hits())
		}
	})
}
