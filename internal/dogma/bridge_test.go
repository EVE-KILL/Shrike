package dogma

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if os.Getenv("SHRIKE_DOGMA_HELPER") == "1" {
		for scanner := bufio.NewScanner(os.Stdin); scanner.Scan(); {
			var req request
			if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
				os.Exit(2)
			}
			switch req.Fit.ShipTypeID {
			case -1:
				req.ID++
			case -2:
				fmt.Printf("{\"id\":%d,\"error\":\"invalid ship\"}\n", req.ID)
				continue
			case -3:
				time.Sleep(time.Minute)
			case -4:
				os.Exit(3)
			}
			fmt.Printf("{\"id\":%d,\"hull\":{\"ehp\":123.5}}\n", req.ID)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func helperProcess(t *testing.T) *process {
	t.Helper()
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("SHRIKE_DOGMA_HELPER", "1")
	t.Setenv("DOGMA_WORKER_PATH", binary)
	dir := t.TempDir()
	for _, name := range []string{"sde"} {
		if err := os.WriteFile(filepath.Join(dir, name+".dat"), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("DOGMA_SDE_DIR", dir)
	p := &process{}
	t.Cleanup(p.stop)
	return p
}

func TestWorkerProtocolRecovery(t *testing.T) {
	p := helperProcess(t)
	for _, id := range []int64{1, 2} {
		stats, err := p.evaluate(t.Context(), Fit{ShipTypeID: id}, nil)
		if err != nil || stats.EHP == nil || *stats.EHP != 123.5 {
			t.Fatalf("stats=%+v err=%v", stats, err)
		}
	}
	for _, id := range []int64{-1, -2, -4} {
		if _, err := p.evaluate(t.Context(), Fit{ShipTypeID: id}, nil); err == nil {
			t.Fatalf("ship %d should fail", id)
		}
		stats, err := p.evaluate(t.Context(), Fit{ShipTypeID: 1}, nil)
		if err != nil || stats.EHP == nil {
			t.Fatalf("worker did not recover: %v", err)
		}
	}
}

func TestWorkerCancellation(t *testing.T) {
	p := helperProcess(t)
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	started := time.Now()
	if _, err := p.evaluate(ctx, Fit{ShipTypeID: -3}, nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
	if time.Since(started) > 3*time.Second {
		t.Fatal("cancelled worker blocked its pool slot")
	}
	if _, err := p.evaluate(t.Context(), Fit{ShipTypeID: 1}, nil); err != nil {
		t.Fatalf("restart after cancellation: %v", err)
	}
}

func TestWorkerPaths(t *testing.T) {
	t.Setenv("DOGMA_WORKER_PATH", t.TempDir())
	if _, err := workerPath(); err == nil {
		t.Fatal("directory accepted as executable")
	}
	t.Setenv("DOGMA_SDE_DIR", t.TempDir())
	if _, err := dataPath(); err == nil {
		t.Fatal("missing data accepted")
	}
}

// These snapshots were captured from the previous engine before removing its
// runtime. They cover weapons, resistances, skills, propulsion, Bastion, drones
// and Shrike's remote repair/capacitor additions on the same pinned SDE.
func TestNativeEngineParity(t *testing.T) {
	if _, err := workerPath(); err != nil {
		if os.Getenv("DOGMA_WORKER_PATH") != "" {
			t.Fatal(err)
		}
		t.Skip("run make build-dogma to enable native engine regression tests")
	}
	_, err := dataPath()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join("..", "..")
	requests, err := os.ReadFile(filepath.Join(root, "engine/dogma/testdata/requests.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile(filepath.Join(root, "engine/dogma/testdata/expected.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	rows := strings.Split(strings.TrimSpace(string(requests)), "\n")
	snapshots := strings.Split(strings.TrimSpace(string(expected)), "\n")
	if len(rows) != len(snapshots) {
		t.Fatal("fixture counts differ")
	}
	p := &process{}
	t.Cleanup(p.stop)
	for i, line := range rows {
		t.Run(fmt.Sprintf("fit-%02d", i+1), func(t *testing.T) {
			var req request
			if err := json.Unmarshal([]byte(line), &req); err != nil {
				t.Fatal(err)
			}
			if len(req.Fit.Modules) == 0 && len(req.Fit.Drones) == 0 {
				// A bare hull supplied by a Go caller may use nil slices.
				req.Fit.Modules, req.Fit.Drones = nil, nil
			}
			var want struct {
				Hull map[string]*float64 `json:"hull"`
			}
			if err := json.Unmarshal([]byte(snapshots[i]), &want); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			stats, err := p.evaluate(ctx, req.Fit, req.Skills)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := json.Marshal(stats)
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]*float64
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatal(err)
			}
			if len(got) != len(want.Hull) {
				t.Fatalf("field count %d != %d", len(got), len(want.Hull))
			}
			for key, w := range want.Hull {
				g, ok := got[key]
				if !ok || (g == nil) != (w == nil) {
					t.Errorf("%s presence mismatch", key)
					continue
				}
				if g != nil && math.Abs(*g-*w) > math.Max(1e-7, math.Abs(*w)*1e-9) {
					t.Errorf("%s got %.12g want %.12g", key, *g, *w)
				}
			}
		})
	}
}
