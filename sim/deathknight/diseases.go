package deathknight

import (
	"strconv"
	"time"

	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/proto"
)

func (dk *Deathknight) drwCountActiveDiseases(target *core.Unit) float64 {
	count := 0
	if dk.RuneWeapon.FrostFeverDisease[target.Index].IsActive() {
		count++
	}
	if dk.RuneWeapon.BloodPlagueDisease[target.Index].IsActive() {
		count++
	}
	return float64(count)
}

func (dk *Deathknight) dkCountActiveDiseases(target *core.Unit) float64 {
	count := 0
	if dk.FrostFeverDisease[target.Index].IsActive() {
		count++
	}
	if dk.BloodPlagueDisease[target.Index].IsActive() {
		count++
	}
	if dk.EbonPlagueOrCryptFeverAura[target.Index].IsActive() {
		count++
	}
	return float64(count)
}

// diseaseMultiplier calculates the bonus based on if you have DarkrunedBattlegear 4p.
//
//	This function is slow so should only be used during initialization.
func (dk *Deathknight) dkDiseaseMultiplier(multiplier float64) float64 {
	if dk.Env.IsFinalized() {
		panic("dont call dk.diseaseMultiplier function during runtime, cache result during initialization")
	}
	if dk.HasSetBonus(ItemSetDarkrunedBattlegear, 4) {
		return multiplier * 1.2
	}
	return multiplier
}

func (dk *Deathknight) registerDiseaseDots() {
	dk.registerFrostFever()
	dk.registerBloodPlague()
}

func (dk *Deathknight) registerFrostFever() {
	actionID := core.ActionID{SpellID: 55095}

	flagTs := make([]bool, dk.Env.GetNumTargets())
	isRefreshing := make([]bool, dk.Env.GetNumTargets())

	dk.FrostFeverSpell = dk.RegisterSpell(nil, core.SpellConfig{
		ActionID:    actionID,
		SpellSchool: core.SpellSchoolFrost,
		ProcMask:    core.ProcMaskSpellDamage,
		Flags:       core.SpellFlagDisease,

		DamageMultiplier: core.TernaryFloat64(dk.HasMajorGlyph(proto.DeathknightMajorGlyph_GlyphOfIcyTouch), 1.2, 1.0),
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, unit *core.Unit, spell *core.Spell) {
			if dk.FrostFeverDisease[unit.Index].IsActive() {
				isRefreshing[unit.Index] = true
			}
			dk.FrostFeverDisease[unit.Index].Apply(sim)
			isRefreshing[unit.Index] = false
			dk.FrostFeverDebuffAura[unit.Index].Activate(sim)
		},
	}, nil, nil)

	dk.FrostFeverDisease = make([]*core.Dot, dk.Env.GetNumTargets())
	dk.FrostFeverExtended = make([]int, dk.Env.GetNumTargets())

	for _, encounterTarget := range dk.Env.Encounter.Targets {
		target := &encounterTarget.Unit
		dk.FrostFeverDisease[target.Index] = core.NewDot(core.Dot{
			Spell: dk.FrostFeverSpell.Spell,
			Aura: target.RegisterAura(core.Aura{
				Label:    "FrostFever-" + strconv.Itoa(int(dk.Index)),
				ActionID: actionID,
				OnGain: func(aura *core.Aura, sim *core.Simulation) {
					if dk.IcyTalonsAura != nil {
						dk.IcyTalonsAura.Activate(sim)
					}
					if dk.EbonPlagueOrCryptFeverAura[aura.Unit.Index] != nil {
						dk.EbonPlagueOrCryptFeverAura[aura.Unit.Index].Activate(sim)
					}
				},
				OnExpire: func(aura *core.Aura, sim *core.Simulation) {
					if !isRefreshing[aura.Unit.Index] {
						flagTs[aura.Unit.Index] = false
					}
				},
			}),
			NumberOfTicks: 5 + dk.Talents.Epidemic,
			TickLength:    time.Second * 3,
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				firstTsApply := !flagTs[target.Index]
				flagTs[target.Index] = true
				// 80.0 * 0.32 * 1.15 base, 0.055 * 1.15
				dot.SnapshotBaseDamage = (29.44 + 0.06325*dk.getImpurityBonus(dot.Spell)) *
					core.TernaryFloat64(firstTsApply, 1.0, dk.RoRTSBonus(target))

				if !isRollover {
					dot.SnapshotAttackerMultiplier = dot.Spell.AttackerDamageMultiplier(dot.Spell.Unit.AttackTables[target.UnitIndex])
				}
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				result := dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.Spell.OutcomeAlwaysHit)
				dk.doWanderingPlague(sim, dot.Spell, result)
			},
		})
	}
}

func (dk *Deathknight) registerBloodPlague() {
	actionID := core.ActionID{SpellID: 55078}

	flagRor := make([]bool, dk.Env.GetNumTargets())
	isRefreshing := make([]bool, dk.Env.GetNumTargets())

	dk.BloodPlagueSpell = dk.RegisterSpell(nil, core.SpellConfig{
		ActionID:    actionID,
		SpellSchool: core.SpellSchoolShadow,
		ProcMask:    core.ProcMaskSpellDamage,
		Flags:       core.SpellFlagDisease,

		DamageMultiplier: 1,
		CritMultiplier:   dk.DefaultMeleeCritMultiplier(),
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, unit *core.Unit, spell *core.Spell) {
			if dk.BloodPlagueDisease[unit.Index].IsActive() {
				isRefreshing[unit.Index] = true
			}
			dk.BloodPlagueDisease[unit.Index].Apply(sim)
			isRefreshing[unit.Index] = false
		},
	}, nil, nil)

	dk.BloodPlagueDisease = make([]*core.Dot, dk.Env.GetNumTargets())
	dk.BloodPlagueExtended = make([]int, dk.Env.GetNumTargets())

	// Tier9 4Piece
	canCrit := dk.HasSetBonus(ItemSetThassariansBattlegear, 4)

	for _, encounterTarget := range dk.Env.Encounter.Targets {
		target := &encounterTarget.Unit
		dk.BloodPlagueDisease[target.Index] = core.NewDot(core.Dot{
			Spell: dk.BloodPlagueSpell.Spell,
			Aura: target.RegisterAura(core.Aura{
				Label:    "BloodPlague-" + strconv.Itoa(int(dk.Index)),
				ActionID: actionID,
				OnGain: func(aura *core.Aura, sim *core.Simulation) {
					if dk.EbonPlagueOrCryptFeverAura[aura.Unit.Index] != nil {
						dk.EbonPlagueOrCryptFeverAura[aura.Unit.Index].Activate(sim)
					}
				},
				OnExpire: func(aura *core.Aura, sim *core.Simulation) {
					if !isRefreshing[aura.Unit.Index] {
						flagRor[aura.Unit.Index] = false
					}
				},
			}),
			NumberOfTicks: 5 + dk.Talents.Epidemic,
			TickLength:    time.Second * 3,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				firstRorApply := !flagRor[target.Index]
				flagRor[target.Index] = true
				// 80.0 * 0.394 * 1.15 for base, 0.055 * 1.15 for ap coeff
				dot.SnapshotBaseDamage = (36.248 + 0.06325*dk.getImpurityBonus(dot.Spell)) *
					core.TernaryFloat64(firstRorApply, 1.0, dk.RoRTSBonus(target))

				if !isRollover {
					dot.SnapshotCritChance = dot.Spell.SpellCritChance(target)
					dot.SnapshotAttackerMultiplier = dot.Spell.AttackerDamageMultiplier(dot.Spell.Unit.AttackTables[target.UnitIndex])
				}
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				var result *core.SpellResult
				if canCrit {
					result = dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeSnapshotCrit)
				} else {
					result = dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.Spell.OutcomeAlwaysHit)
				}
				dk.doWanderingPlague(sim, dot.Spell, result)
			},
		})
	}
}
func (dk *Deathknight) registerDrwDiseaseDots() {
	dk.registerDrwFrostFever()
	dk.registerDrwBloodPlague()
}

func (dk *Deathknight) registerDrwFrostFever() {
	actionID := core.ActionID{SpellID: 55095}

	dk.RuneWeapon.FrostFeverSpell = dk.RuneWeapon.RegisterSpell(core.SpellConfig{
		ActionID:    actionID,
		SpellSchool: core.SpellSchoolFrost,
		ProcMask:    core.ProcMaskSpellDamage,
		Flags:       core.SpellFlagDisease,

		DamageMultiplier: core.TernaryFloat64(dk.HasMajorGlyph(proto.DeathknightMajorGlyph_GlyphOfIcyTouch), 1.2, 1.0),
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, unit *core.Unit, spell *core.Spell) {
			dk.RuneWeapon.FrostFeverDisease[unit.Index].Apply(sim)
		},
	})

	dk.RuneWeapon.FrostFeverDisease = make([]*core.Dot, dk.Env.GetNumTargets())

	for _, encounterTarget := range dk.Env.Encounter.Targets {
		target := &encounterTarget.Unit

		dk.RuneWeapon.FrostFeverDisease[target.Index] = core.NewDot(core.Dot{
			Aura: target.RegisterAura(core.Aura{
				Label:    "DrwFrostFever-" + strconv.Itoa(int(dk.RuneWeapon.Index)),
				ActionID: actionID,
			}),
			NumberOfTicks: 5 + dk.Talents.Epidemic,
			TickLength:    time.Second * 3,
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				// 80.0 * 0.32 * 1.15 base, 0.055 * 1.15
				dot.SnapshotBaseDamage = 29.44 + 0.06325*dk.getImpurityBonus(dot.Spell)

				if !isRollover {
					dot.SnapshotAttackerMultiplier = dot.Spell.AttackerDamageMultiplier(dot.Spell.Unit.AttackTables[target.UnitIndex])
				}
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.Spell.OutcomeAlwaysHit)
			},
		})

		dk.RuneWeapon.FrostFeverDisease[target.Index].Spell = dk.RuneWeapon.FrostFeverSpell
	}
}

func (dk *Deathknight) registerDrwBloodPlague() {
	actionID := core.ActionID{SpellID: 55078}

	dk.RuneWeapon.BloodPlagueSpell = dk.RuneWeapon.RegisterSpell(core.SpellConfig{
		ActionID:    actionID,
		SpellSchool: core.SpellSchoolShadow,
		ProcMask:    core.ProcMaskSpellDamage,
		Flags:       core.SpellFlagDisease,

		DamageMultiplier: 1,
		// TODO: Shouldn't this be dk.RuneWeapon?
		CritMultiplier:   dk.DefaultMeleeCritMultiplier(),
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, unit *core.Unit, spell *core.Spell) {
			dk.RuneWeapon.BloodPlagueDisease[unit.Index].Apply(sim)
		},
	})

	dk.RuneWeapon.BloodPlagueDisease = make([]*core.Dot, dk.Env.GetNumTargets())

	// Tier9 4Piece
	canCrit := dk.HasSetBonus(ItemSetThassariansBattlegear, 4)

	for _, encounterTarget := range dk.Env.Encounter.Targets {
		target := &encounterTarget.Unit

		dk.RuneWeapon.BloodPlagueDisease[target.Index] = core.NewDot(core.Dot{
			Aura: target.RegisterAura(core.Aura{
				Label:    "DrwBloodPlague-" + strconv.Itoa(int(dk.RuneWeapon.Index)),
				ActionID: actionID,
			}),
			NumberOfTicks: 5 + dk.Talents.Epidemic,
			TickLength:    time.Second * 3,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				// 80.0 * 0.394 * 1.15 for base, 0.055 * 1.15 for ap coeff
				dot.SnapshotBaseDamage = 36.248 + 0.06325*dk.getImpurityBonus(dot.Spell)

				if !isRollover {
					dot.SnapshotCritChance = dot.Spell.SpellCritChance(target)
					dot.SnapshotAttackerMultiplier = dot.Spell.AttackerDamageMultiplier(dot.Spell.Unit.AttackTables[target.UnitIndex])
				}
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				var result *core.SpellResult
				if canCrit {
					result = dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeSnapshotCrit)
				} else {
					result = dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.Spell.OutcomeAlwaysHit)
				}
				dk.doWanderingPlague(sim, dot.Spell, result)
			},
		})

		dk.RuneWeapon.BloodPlagueDisease[target.Index].Spell = dk.RuneWeapon.BloodPlagueSpell
	}
}

func (dk *Deathknight) doWanderingPlague(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
	if dk.Talents.WanderingPlague == 0 {
		return
	}

	if dk.LastTickTime == sim.CurrentTime {
		return
	}

	attackTable := dk.AttackTables[result.Target.UnitIndex]
	physCritChance := spell.PhysicalCritChance(result.Target, attackTable)
	if sim.RandomFloat("Wandering Plague Roll") < physCritChance {
		dk.LastTickTime = sim.CurrentTime
		dk.LastDiseaseDamage = result.Damage / dk.WanderingPlague.TargetDamageMultiplier(attackTable, false)
		dk.WanderingPlague.Cast(sim, result.Target)
	}
}
