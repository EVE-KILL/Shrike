package killmail

import (
	"math/big"
	"sort"
)

// DefaultParticipationBasisPoints reserves ten percent of a killmail's point
// pool for showing up. The remaining ninety percent follows damage dealt.
const DefaultParticipationBasisPoints int64 = 1_000

// PointParticipant is the player-visible information used to split a
// killmail's point pool. CharacterID zero identifies an NPC and is ignored.
type PointParticipant struct {
	CharacterID int32
	DamageDone  int64
	FinalBlow   bool
}

type pointCandidate struct {
	characterID int32
	damage      int64
	finalBlow   bool
	points      int64
	remainder   *big.Int
}

// AllocatePoints splits pool between distinct player characters. A fixed
// participation reserve is shared equally and the rest follows damage dealt.
// Integer leftovers use largest-remainder allocation, so the returned shares
// always sum exactly to pool when at least one player participated.
func AllocatePoints(pool, participationBasisPoints int64, participants []PointParticipant) map[int32]int64 {
	result := make(map[int32]int64)
	if pool <= 0 {
		return result
	}
	if participationBasisPoints < 0 {
		participationBasisPoints = 0
	}
	if participationBasisPoints > 10_000 {
		participationBasisPoints = 10_000
	}

	merged := make(map[int32]*pointCandidate)
	for _, participant := range participants {
		if participant.CharacterID == 0 {
			continue
		}
		candidate := merged[participant.CharacterID]
		if candidate == nil {
			candidate = &pointCandidate{characterID: participant.CharacterID}
			merged[participant.CharacterID] = candidate
		}
		if participant.DamageDone > 0 {
			candidate.damage += participant.DamageDone
		}
		candidate.finalBlow = candidate.finalBlow || participant.FinalBlow
	}
	if len(merged) == 0 {
		return result
	}

	candidates := make([]*pointCandidate, 0, len(merged))
	var totalDamage int64
	for _, candidate := range merged {
		candidates = append(candidates, candidate)
		totalDamage += candidate.damage
	}
	if totalDamage <= 0 {
		participationBasisPoints = 10_000
		totalDamage = 1
		for _, candidate := range candidates {
			candidate.damage = 0
		}
	}

	// share = pool * (reserve/n + damagePart*damage/totalDamage).
	// A common denominator lets QuoRem produce exact integer floors and exact
	// remainders without floating-point drift, even on enormous fleet fights.
	n := int64(len(candidates))
	denominator := new(big.Int).Mul(big.NewInt(10_000*n), big.NewInt(totalDamage))
	allocated := int64(0)
	for _, candidate := range candidates {
		reserveNumerator := new(big.Int).Mul(big.NewInt(participationBasisPoints), big.NewInt(totalDamage))
		damageNumerator := new(big.Int).Mul(big.NewInt((10_000-participationBasisPoints)*n), big.NewInt(candidate.damage))
		numerator := new(big.Int).Add(reserveNumerator, damageNumerator)
		numerator.Mul(numerator, big.NewInt(pool))

		quotient, remainder := new(big.Int), new(big.Int)
		quotient.QuoRem(numerator, denominator, remainder)
		candidate.points = quotient.Int64()
		candidate.remainder = remainder
		allocated += candidate.points
	}

	sort.Slice(candidates, func(i, j int) bool {
		if cmp := candidates[i].remainder.Cmp(candidates[j].remainder); cmp != 0 {
			return cmp > 0
		}
		if candidates[i].damage != candidates[j].damage {
			return candidates[i].damage > candidates[j].damage
		}
		if candidates[i].finalBlow != candidates[j].finalBlow {
			return candidates[i].finalBlow
		}
		return candidates[i].characterID < candidates[j].characterID
	})
	for i := int64(0); i < pool-allocated; i++ {
		candidates[i%n].points++
	}
	for _, candidate := range candidates {
		result[candidate.characterID] = candidate.points
	}
	return result
}
