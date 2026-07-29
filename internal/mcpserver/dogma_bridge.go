package mcpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

type EsfSlot struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}

type EsfCharge struct {
	TypeID int64 `json:"type_id"`
}

type EsfModule struct {
	TypeID int64      `json:"type_id"`
	Slot   EsfSlot    `json:"slot"`
	State  string     `json:"state"`
	Charge *EsfCharge `json:"charge,omitempty"`
}

type EsfDrone struct {
	TypeID int64  `json:"type_id"`
	State  string `json:"state"`
}

type EsfFit struct {
	ShipTypeID int64       `json:"ship_type_id"`
	Modules    []EsfModule `json:"modules"`
	Drones     []EsfDrone  `json:"drones"`
}

type HullStats struct {
	AlignTime        *float64 `json:"align_time"`
	MaxVelocity      *float64 `json:"max_velocity"`
	SignatureRadius  *float64 `json:"signature_radius"`
	Mass             *float64 `json:"mass"`
	EHP              *float64 `json:"ehp"`
	ShieldEHP        *float64 `json:"shield_ehp"`
	ArmorEHP         *float64 `json:"armor_ehp"`
	HullEHP          *float64 `json:"hull_ehp"`
	CapCapacity      *float64 `json:"cap_capacity"`
	CapDepletesIn    *float64 `json:"cap_depletes_in"`
	CapPeakDelta     *float64 `json:"cap_peak_delta"`
	DPSWithReload    *float64 `json:"dps_with_reload"`
	DPSWithoutReload *float64 `json:"dps_without_reload"`
	Alpha            *float64 `json:"alpha"`
	MaxTargetRange   *float64 `json:"max_target_range"`
	ScanResolution   *float64 `json:"scan_resolution"`
	MaxLockedTargets *float64 `json:"max_locked_targets"`
	PGOutput         *float64 `json:"pg_output"`
	CPUOutput        *float64 `json:"cpu_output"`
	Calibration      *float64 `json:"calibration"`
}

type dogmaBridgeRequest struct {
	ID     int64             `json:"id"`
	Fit    EsfFit            `json:"fit"`
	Skills *map[string]int64 `json:"skills,omitempty"`
}

type dogmaBridgeResponse struct {
	ID    int64     `json:"id"`
	Hull  HullStats `json:"hull"`
	Error string    `json:"error,omitempty"`
}

type dogmaBridgeProcess struct {
	mu      sync.Mutex
	nextID  int64
	command *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Scanner
}

var sharedDogmaBridge dogmaBridgeProcess

func evaluateDogma(ctx context.Context, fit EsfFit, noSkills bool) (HullStats, error) {
	return sharedDogmaBridge.evaluate(ctx, fit, noSkills)
}

func (bridge *dogmaBridgeProcess) evaluate(ctx context.Context, fit EsfFit, noSkills bool) (HullStats, error) {
	bridge.mu.Lock()
	defer bridge.mu.Unlock()
	for attempt := 0; attempt < 2; attempt++ {
		if err := bridge.ensureStarted(ctx); err != nil {
			return HullStats{}, err
		}
		bridge.nextID++
		request := dogmaBridgeRequest{ID: bridge.nextID, Fit: fit}
		if noSkills {
			empty := map[string]int64{}
			request.Skills = &empty
		}
		data, err := json.Marshal(request)
		if err == nil {
			_, err = bridge.stdin.Write(append(data, '\n'))
		}
		if err != nil {
			bridge.stop()
			continue
		}
		if !bridge.stdout.Scan() {
			err = bridge.stdout.Err()
			bridge.stop()
			if err == nil {
				err = fmt.Errorf("dogma bridge exited")
			}
			if attempt == 1 {
				return HullStats{}, err
			}
			continue
		}
		var response dogmaBridgeResponse
		if err := json.Unmarshal(bridge.stdout.Bytes(), &response); err != nil {
			bridge.stop()
			if attempt == 1 {
				return HullStats{}, fmt.Errorf("decode dogma bridge response: %w", err)
			}
			continue
		}
		if response.ID != request.ID {
			bridge.stop()
			return HullStats{}, fmt.Errorf("dogma bridge response id mismatch")
		}
		if response.Error != "" {
			return HullStats{}, fmt.Errorf("dogma engine failed: %s", response.Error)
		}
		return response.Hull, nil
	}
	return HullStats{}, fmt.Errorf("dogma bridge unavailable")
}

func (bridge *dogmaBridgeProcess) ensureStarted(ctx context.Context) error {
	if bridge.command != nil {
		return nil
	}
	path, err := dogmaBridgePath()
	if err != nil {
		return err
	}
	command := exec.CommandContext(context.WithoutCancel(ctx), "bun", path)
	stdin, err := command.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		return err
	}
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		return fmt.Errorf("start dogma bridge: %w", err)
	}
	bridge.command, bridge.stdin, bridge.stdout = command, stdin, bufio.NewScanner(stdout)
	bridge.stdout.Buffer(make([]byte, 64*1024), 2*1024*1024)
	return nil
}

func (bridge *dogmaBridgeProcess) stop() {
	if bridge.stdin != nil {
		_ = bridge.stdin.Close()
	}
	if bridge.command != nil && bridge.command.Process != nil {
		_ = bridge.command.Process.Kill()
		_, _ = bridge.command.Process.Wait()
	}
	bridge.command, bridge.stdin, bridge.stdout = nil, nil, nil
}

func dogmaBridgePath() (string, error) {
	if configured := os.Getenv("DOGMA_BRIDGE_PATH"); configured != "" {
		return configured, nil
	}
	candidates := []string{
		"web/packages/dogma/src/bridge.ts",
		filepath.Join("..", "web", "packages", "dogma", "src", "bridge.ts"),
	}
	if _, sourceFile, _, ok := runtime.Caller(0); ok {
		candidates = append(candidates, filepath.Join(
			filepath.Dir(sourceFile), "..", "..", "web", "packages", "dogma",
			"src", "bridge.ts",
		))
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			path, absoluteErr := filepath.Abs(candidate)
			return path, absoluteErr
		}
	}
	return "", fmt.Errorf("dogma bridge not found; set DOGMA_BRIDGE_PATH")
}
