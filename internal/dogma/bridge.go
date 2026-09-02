// Package dogma evaluates EVE ship fits through Shrike's long-lived Bun bridge.
package dogma

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
	"strconv"
	"sync"
)

type Slot struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}
type Charge struct {
	TypeID int64 `json:"type_id"`
}
type Module struct {
	TypeID int64   `json:"type_id"`
	Slot   Slot    `json:"slot"`
	State  string  `json:"state"`
	Charge *Charge `json:"charge,omitempty"`
}
type Drone struct {
	TypeID int64  `json:"type_id"`
	State  string `json:"state"`
}
type Fit struct {
	ShipTypeID int64    `json:"ship_type_id"`
	Modules    []Module `json:"modules"`
	Drones     []Drone  `json:"drones"`
}

type Stats struct {
	AlignTime              *float64 `json:"align_time"`
	MaxVelocity            *float64 `json:"max_velocity"`
	SignatureRadius        *float64 `json:"signature_radius"`
	Mass                   *float64 `json:"mass"`
	EHP                    *float64 `json:"ehp"`
	ShieldEHP              *float64 `json:"shield_ehp"`
	ArmorEHP               *float64 `json:"armor_ehp"`
	HullEHP                *float64 `json:"hull_ehp"`
	CapCapacity            *float64 `json:"cap_capacity"`
	CapDepletesIn          *float64 `json:"cap_depletes_in"`
	CapPeakDelta           *float64 `json:"cap_peak_delta"`
	DPSWithReload          *float64 `json:"dps_with_reload"`
	DPSWithoutReload       *float64 `json:"dps_without_reload"`
	Alpha                  *float64 `json:"alpha"`
	MaxTargetRange         *float64 `json:"max_target_range"`
	ScanResolution         *float64 `json:"scan_resolution"`
	MaxLockedTargets       *float64 `json:"max_locked_targets"`
	PGOutput               *float64 `json:"pg_output"`
	CPUOutput              *float64 `json:"cpu_output"`
	Calibration            *float64 `json:"calibration"`
	ShieldBoost            *float64 `json:"shield_boost"`
	ShieldEffectiveBoost   *float64 `json:"shield_effective_boost"`
	ArmorRepair            *float64 `json:"armor_repair"`
	ArmorEffectiveRepair   *float64 `json:"armor_effective_repair"`
	HullRepair             *float64 `json:"hull_repair"`
	HullEffectiveRepair    *float64 `json:"hull_effective_repair"`
	PassiveShield          *float64 `json:"passive_shield"`
	PassiveShieldEffective *float64 `json:"passive_shield_effective"`
	RemoteShield           *float64 `json:"remote_shield"`
	RemoteArmor            *float64 `json:"remote_armor"`
	RemoteHull             *float64 `json:"remote_hull"`
	RemoteCap              *float64 `json:"remote_cap"`
	Neut                   *float64 `json:"neut"`
	Nos                    *float64 `json:"nos"`
	ShieldHP               *float64 `json:"shield_hp"`
	ArmorHP                *float64 `json:"armor_hp"`
	HullHP                 *float64 `json:"hull_hp"`
	ShieldEMResist         *float64 `json:"shield_em_resist"`
	ShieldThermalResist    *float64 `json:"shield_thermal_resist"`
	ShieldKineticResist    *float64 `json:"shield_kinetic_resist"`
	ShieldExplosiveResist  *float64 `json:"shield_explosive_resist"`
	ArmorEMResist          *float64 `json:"armor_em_resist"`
	ArmorThermalResist     *float64 `json:"armor_thermal_resist"`
	ArmorKineticResist     *float64 `json:"armor_kinetic_resist"`
	ArmorExplosiveResist   *float64 `json:"armor_explosive_resist"`
	HullEMResist           *float64 `json:"hull_em_resist"`
	HullThermalResist      *float64 `json:"hull_thermal_resist"`
	HullKineticResist      *float64 `json:"hull_kinetic_resist"`
	HullExplosiveResist    *float64 `json:"hull_explosive_resist"`
}

type request struct {
	ID     int64             `json:"id"`
	Fit    Fit               `json:"fit"`
	Skills *map[string]int64 `json:"skills,omitempty"`
}
type response struct {
	ID    int64  `json:"id"`
	Hull  Stats  `json:"hull"`
	Error string `json:"error,omitempty"`
}
type process struct {
	mu      sync.Mutex
	nextID  int64
	command *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Scanner
}
type processPool struct {
	once      sync.Once
	available chan *process
}

var shared processPool

func Evaluate(ctx context.Context, fit Fit, skills *map[string]int64) (Stats, error) {
	shared.once.Do(func() {
		size := 4
		if raw := os.Getenv("DOGMA_BRIDGE_PROCESSES"); raw != "" {
			if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 32 {
				size = n
			}
		}
		shared.available = make(chan *process, size)
		for range size {
			shared.available <- &process{}
		}
	})
	select {
	case p := <-shared.available:
		defer func() { shared.available <- p }()
		return p.evaluate(ctx, fit, skills)
	case <-ctx.Done():
		return Stats{}, ctx.Err()
	}
}

func (p *process) evaluate(ctx context.Context, fit Fit, skills *map[string]int64) (Stats, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for attempt := range 2 {
		if err := p.ensureStarted(ctx); err != nil {
			return Stats{}, err
		}
		p.nextID++
		req := request{ID: p.nextID, Fit: fit, Skills: skills}
		data, err := json.Marshal(req)
		if err == nil {
			_, err = p.stdin.Write(append(data, '\n'))
		}
		if err != nil {
			p.stop()
			continue
		}
		if !p.stdout.Scan() {
			err = p.stdout.Err()
			p.stop()
			if err == nil {
				err = fmt.Errorf("dogma bridge exited")
			}
			if attempt == 1 {
				return Stats{}, err
			}
			continue
		}
		var res response
		if err := json.Unmarshal(p.stdout.Bytes(), &res); err != nil {
			p.stop()
			if attempt == 1 {
				return Stats{}, fmt.Errorf("decode dogma bridge response: %w", err)
			}
			continue
		}
		if res.ID != req.ID {
			p.stop()
			return Stats{}, fmt.Errorf("dogma bridge response id mismatch")
		}
		if res.Error != "" {
			return Stats{}, fmt.Errorf("dogma engine failed: %s", res.Error)
		}
		return res.Hull, nil
	}
	return Stats{}, fmt.Errorf("dogma bridge unavailable")
}

func (p *process) ensureStarted(ctx context.Context) error {
	if p.command != nil {
		return nil
	}
	path, err := bridgePath()
	if err != nil {
		return err
	}
	// #nosec G204 -- path is resolved from Shrike's fixed, validated bridge locations.
	cmd := exec.CommandContext(context.WithoutCancel(ctx), "bun", path)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start dogma bridge: %w", err)
	}
	p.command, p.stdin, p.stdout = cmd, stdin, bufio.NewScanner(stdout)
	p.stdout.Buffer(make([]byte, 64*1024), 2*1024*1024)
	return nil
}
func (p *process) stop() {
	if p.stdin != nil {
		_ = p.stdin.Close()
	}
	if p.command != nil && p.command.Process != nil {
		_ = p.command.Process.Kill()
		_, _ = p.command.Process.Wait()
	}
	p.command, p.stdin, p.stdout = nil, nil, nil
}
func bridgePath() (string, error) {
	if configured := os.Getenv("DOGMA_BRIDGE_PATH"); configured != "" {
		return configured, nil
	}
	candidates := []string{"web/packages/dogma/src/bridge.ts", filepath.Join("..", "web", "packages", "dogma", "src", "bridge.ts")}
	if _, source, _, ok := runtime.Caller(0); ok {
		candidates = append(candidates, filepath.Join(filepath.Dir(source), "..", "..", "web", "packages", "dogma", "src", "bridge.ts"))
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Abs(candidate)
		}
	}
	return "", fmt.Errorf("dogma bridge not found; set DOGMA_BRIDGE_PATH")
}
