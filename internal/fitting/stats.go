package fitting

import (
	"context"
	"errors"
	"fmt"

	"github.com/eve-kill/shrike/internal/dogma"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	DogmaEngineVersion = "7.1.0+ek.3"
)

// CalculateAndStoreStats evaluates the representative payload stored for a fit
// at all-V and persists the fields used by catalogue sorting and filtering.
func CalculateAndStoreStats(ctx context.Context, pool *pgxpool.Pool, fit *Fitting, shipTypeID int32) error {
	var currentEngine, currentSDE string
	err := pool.QueryRow(ctx, `SELECT engine_version, sde_version FROM fitting_stats WHERE fit_hash = $1`, fit.FitHash).Scan(&currentEngine, &currentSDE)
	if err == nil && currentEngine == DogmaEngineVersion && currentSDE == DogmaSDEVersion {
		return nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("load fitting stats version: %w", err)
	}

	evaluation := dogma.Fit{ShipTypeID: int64(shipTypeID), Modules: []dogma.Module{}, Drones: []dogma.Drone{}}
	for _, item := range fit.Items {
		if item.SlotGroup == SlotDrone {
			for range max(1, int(item.Quantity)) {
				evaluation.Drones = append(evaluation.Drones, dogma.Drone{TypeID: int64(item.TypeID), State: "Active"})
			}
			continue
		}
		slot, state := dogmaSlot(item.SlotGroup)
		if slot == "" {
			continue
		}
		module := dogma.Module{TypeID: int64(item.TypeID), Slot: dogma.Slot{Type: slot, Index: int(item.Ordinal)}, State: state}
		if item.ChargeTypeID != 0 {
			module.Charge = &dogma.Charge{TypeID: int64(item.ChargeTypeID)}
		}
		evaluation.Modules = append(evaluation.Modules, module)
	}
	stats, err := dogma.Evaluate(ctx, evaluation, nil)
	if err != nil {
		return err
	}
	capStable := stats.CapDepletesIn != nil && *stats.CapDepletesIn < 0
	_, err = pool.Exec(ctx, `
		INSERT INTO fitting_stats (
			fit_hash, ship_type_id, skill_level, dps_with_reload, dps_without_reload, alpha,
			ehp, shield_ehp, armor_ehp, hull_ehp, shield_boost, shield_effective_boost,
			armor_repair, armor_effective_repair, hull_repair, hull_effective_repair,
			passive_shield, passive_shield_effective, remote_shield, remote_armor, remote_hull,
			remote_cap, neut, nos, cap_stable, cap_depletes_in, cap_capacity, cap_peak_delta,
			max_velocity, align_time, signature_radius, max_target_range, scan_resolution,
			shield_hp,armor_hp,hull_hp,
			shield_em_resist,shield_thermal_resist,shield_kinetic_resist,shield_explosive_resist,
			armor_em_resist,armor_thermal_resist,armor_kinetic_resist,armor_explosive_resist,
			hull_em_resist,hull_thermal_resist,hull_kinetic_resist,hull_explosive_resist,
			engine_version, sde_version, calculated_at
		) VALUES ($1,$2,5,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,
			$33,$34,$35,$36,$37,$38,$39,$40,$41,$42,$43,$44,$45,$46,$47,$48,$49,now())
		ON CONFLICT (fit_hash) DO UPDATE SET
			ship_type_id=excluded.ship_type_id, skill_level=excluded.skill_level,
			dps_with_reload=excluded.dps_with_reload, dps_without_reload=excluded.dps_without_reload, alpha=excluded.alpha,
			ehp=excluded.ehp, shield_ehp=excluded.shield_ehp, armor_ehp=excluded.armor_ehp, hull_ehp=excluded.hull_ehp,
			shield_boost=excluded.shield_boost, shield_effective_boost=excluded.shield_effective_boost,
			armor_repair=excluded.armor_repair, armor_effective_repair=excluded.armor_effective_repair,
			hull_repair=excluded.hull_repair, hull_effective_repair=excluded.hull_effective_repair,
			passive_shield=excluded.passive_shield, passive_shield_effective=excluded.passive_shield_effective,
			remote_shield=excluded.remote_shield, remote_armor=excluded.remote_armor, remote_hull=excluded.remote_hull,
			remote_cap=excluded.remote_cap, neut=excluded.neut, nos=excluded.nos,
			cap_stable=excluded.cap_stable, cap_depletes_in=excluded.cap_depletes_in,
			cap_capacity=excluded.cap_capacity, cap_peak_delta=excluded.cap_peak_delta,
			max_velocity=excluded.max_velocity, align_time=excluded.align_time,
			signature_radius=excluded.signature_radius, max_target_range=excluded.max_target_range,
			scan_resolution=excluded.scan_resolution,
			shield_hp=excluded.shield_hp,armor_hp=excluded.armor_hp,hull_hp=excluded.hull_hp,
			shield_em_resist=excluded.shield_em_resist,shield_thermal_resist=excluded.shield_thermal_resist,
			shield_kinetic_resist=excluded.shield_kinetic_resist,shield_explosive_resist=excluded.shield_explosive_resist,
			armor_em_resist=excluded.armor_em_resist,armor_thermal_resist=excluded.armor_thermal_resist,
			armor_kinetic_resist=excluded.armor_kinetic_resist,armor_explosive_resist=excluded.armor_explosive_resist,
			hull_em_resist=excluded.hull_em_resist,hull_thermal_resist=excluded.hull_thermal_resist,
			hull_kinetic_resist=excluded.hull_kinetic_resist,hull_explosive_resist=excluded.hull_explosive_resist,
			engine_version=excluded.engine_version,
			sde_version=excluded.sde_version, calculated_at=excluded.calculated_at`,
		fit.FitHash, shipTypeID, stats.DPSWithReload, stats.DPSWithoutReload, stats.Alpha,
		stats.EHP, stats.ShieldEHP, stats.ArmorEHP, stats.HullEHP, stats.ShieldBoost,
		stats.ShieldEffectiveBoost, stats.ArmorRepair, stats.ArmorEffectiveRepair,
		stats.HullRepair, stats.HullEffectiveRepair, stats.PassiveShield,
		stats.PassiveShieldEffective, stats.RemoteShield, stats.RemoteArmor, stats.RemoteHull,
		stats.RemoteCap, stats.Neut, stats.Nos, capStable, stats.CapDepletesIn,
		stats.CapCapacity, stats.CapPeakDelta, stats.MaxVelocity, stats.AlignTime,
		stats.SignatureRadius, stats.MaxTargetRange, stats.ScanResolution,
		stats.ShieldHP, stats.ArmorHP, stats.HullHP,
		stats.ShieldEMResist, stats.ShieldThermalResist, stats.ShieldKineticResist, stats.ShieldExplosiveResist,
		stats.ArmorEMResist, stats.ArmorThermalResist, stats.ArmorKineticResist, stats.ArmorExplosiveResist,
		stats.HullEMResist, stats.HullThermalResist, stats.HullKineticResist, stats.HullExplosiveResist,
		DogmaEngineVersion, DogmaSDEVersion)
	if err != nil {
		return fmt.Errorf("store fitting stats: %w", err)
	}
	return nil
}

// CalculateStoredStats loads the immutable representative payload for a fit.
func CalculateStoredStats(ctx context.Context, pool *pgxpool.Pool, fitHash string) error {
	var shipTypeID int32
	if err := pool.QueryRow(ctx, `SELECT ship_type_id FROM fittings WHERE fit_hash = $1`, fitHash).Scan(&shipTypeID); err != nil {
		return err
	}
	rows, err := pool.Query(ctx, `SELECT slot_group, ordinal, type_id, coalesce(charge_type_id, 0), quantity FROM fitting_items WHERE fit_hash = $1 ORDER BY slot_group, ordinal`, fitHash)
	if err != nil {
		return err
	}
	defer rows.Close()
	fit := &Fitting{FitHash: fitHash}
	for rows.Next() {
		var item ExtractedItem
		if err := rows.Scan(&item.SlotGroup, &item.Ordinal, &item.TypeID, &item.ChargeTypeID, &item.Quantity); err != nil {
			return err
		}
		fit.Items = append(fit.Items, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return CalculateAndStoreStats(ctx, pool, fit, shipTypeID)
}

func dogmaSlot(slot int32) (string, string) {
	switch slot {
	case SlotHigh:
		return "High", "Active"
	case SlotMed:
		return "Medium", "Active"
	case SlotLow:
		return "Low", "Online"
	case SlotRig:
		return "Rig", "Passive"
	case SlotSubsystem:
		return "SubSystem", "Online"
	}
	return "", ""
}
