package rogue

import (
	"strconv"
	"time"

	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/proto"
)

func (rogue *Rogue) applyPoisons() {
	rogue.applyDeadlyPoison()
	rogue.applyInstantPoison()
	rogue.applyWoundPoison()
}

var DeadlyPoisonActionID = core.ActionID{SpellID: 57973}

func (rogue *Rogue) registerPoisonAuras() {
	numTargets := rogue.Env.GetNumTargets()
	for i := int32(0); i < numTargets; i++ {
		target := rogue.Env.GetTargetUnit(i)
		if rogue.Talents.SavageCombat > 0 {
			rogue.savageCombatDebuffAuras = append(rogue.savageCombatDebuffAuras, core.SavageCombatAura(target, rogue.Talents.SavageCombat))
		}
		if rogue.Talents.MasterPoisoner > 0 {
			masterPoisonerAura := core.MasterPoisonerDebuff(target, float64(rogue.Talents.MasterPoisoner))
			masterPoisonerAura.Duration = core.NeverExpires
			rogue.masterPoisonerDebuffAuras = append(rogue.masterPoisonerDebuffAuras, masterPoisonerAura)
		}
	}
}

func (rogue *Rogue) registerDeadlyPoisonSpell() {
	rogue.DeadlyPoison = rogue.RegisterSpell(core.SpellConfig{
		ActionID:    DeadlyPoisonActionID,
		SpellSchool: core.SpellSchoolNature,
		ProcMask:    core.ProcMaskSpellDamage,

		DamageMultiplier: []float64{1, 1.07, 1.14, 1.20}[rogue.Talents.VilePoisons],
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealOutcome(sim, target, spell.OutcomeMagicHit)

			if result.Landed() {
				dot := rogue.deadlyPoisonDots[target.Index]
				if dot.IsActive() {
					if dot.GetStacks() == 5 {
						if rogue.lastDeadlyPoisonProcMask.Matches(core.ProcMaskMeleeMH) {
							switch rogue.Options.OhImbue {
							case proto.Rogue_Options_DeadlyPoison:
								dot.Refresh(sim)
							case proto.Rogue_Options_InstantPoison:
								rogue.InstantPoison[DeadlyProc].Cast(sim, target)
							case proto.Rogue_Options_WoundPoison:
								rogue.WoundPoison[DeadlyProc].Cast(sim, target)
							}
						}
						if rogue.lastDeadlyPoisonProcMask.Matches(core.ProcMaskMeleeOH) {
							switch rogue.Options.MhImbue {
							case proto.Rogue_Options_DeadlyPoison:
								dot.Refresh(sim)
							case proto.Rogue_Options_InstantPoison:
								rogue.InstantPoison[DeadlyProc].Cast(sim, target)
							case proto.Rogue_Options_WoundPoison:
								rogue.WoundPoison[DeadlyProc].Cast(sim, target)
							}
						}
					}
					dot.Refresh(sim)
					dot.AddStack(sim)
				} else {
					dot.Apply(sim)
					dot.SetStacks(sim, 1)
				}
			}
			rogue.lastDeadlyPoisonProcMask = core.ProcMaskEmpty
		},
	})
	deadlyPoisonDebuffAura := core.Aura{
		Label:     "DeadlyPoison-" + strconv.Itoa(int(rogue.Index)),
		ActionID:  DeadlyPoisonActionID,
		MaxStacks: 5,
		Duration:  time.Second * 12,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			if rogue.Talents.SavageCombat > 0 {
				rogue.savageCombatDebuffAuras[aura.Unit.Index].Activate(sim)
			}
			if rogue.Talents.MasterPoisoner > 0 {
				rogue.masterPoisonerDebuffAuras[aura.Unit.Index].Activate(sim)
			}
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			if rogue.Talents.SavageCombat > 0 {
				rogue.savageCombatDebuffAuras[aura.Unit.Index].Deactivate(sim)
			}
			if rogue.Talents.MasterPoisoner > 0 {
				rogue.masterPoisonerDebuffAuras[aura.Unit.Index].Deactivate(sim)
			}
		},
	}

	onDeadlyTick := func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
		baseDmg := (74 + 0.03*dot.Spell.MeleeAttackPower()) * float64(dot.GetStacks())
		dot.Spell.CalcAndDealPeriodicDamage(sim, target, baseDmg, dot.OutcomeTick)
	}

	if rogue.HasSetBonus(ItemSetTerrorblade, 2) {
		metrics := rogue.NewEnergyMetrics(core.ActionID{SpellID: 64914})
		onDeadlyTick = func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
			baseDmg := (74 + 0.03*dot.Spell.MeleeAttackPower()) * float64(dot.GetStacks())
			result := dot.Spell.CalcAndDealPeriodicDamage(sim, target, baseDmg, dot.OutcomeTick)
			if result.Landed() {
				rogue.AddEnergy(sim, 1, metrics)
			}
		}
	}
	numTargets := rogue.Env.GetNumTargets()
	for i := int32(0); i < numTargets; i++ {
		target := rogue.Env.GetTargetUnit(i)
		dot := core.NewDot(core.Dot{
			Spell:         rogue.DeadlyPoison,
			Aura:          target.RegisterAura(deadlyPoisonDebuffAura),
			NumberOfTicks: 4,
			TickLength:    time.Second * 3,
			// TODO: MAP part snapshots
			OnTick: onDeadlyTick,
		})
		rogue.deadlyPoisonDots = append(rogue.deadlyPoisonDots, dot)
	}
}

func (rogue *Rogue) procDeadlyPoison(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
	rogue.lastDeadlyPoisonProcMask = spell.ProcMask
	rogue.DeadlyPoison.Cast(sim, result.Target)
}

func (rogue *Rogue) applyDeadlyPoison() {
	procMask := core.GetMeleeProcMaskForHands(
		rogue.Options.MhImbue == proto.Rogue_Options_DeadlyPoison,
		rogue.Options.OhImbue == proto.Rogue_Options_DeadlyPoison)

	if procMask == core.ProcMaskUnknown {
		return
	}
	rogue.RegisterAura(core.Aura{
		Label:    "Deadly Poison",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !result.Landed() || !spell.ProcMask.Matches(procMask) {
				return
			}
			if sim.RandomFloat("Deadly Poison") > rogue.GetDeadlyPoisonProcChance(procMask) {
				return
			}
			rogue.procDeadlyPoison(sim, spell, result)
		},
	})
}

func (rogue *Rogue) applyWoundPoison() {
	procMask := core.GetMeleeProcMaskForHands(
		rogue.Options.MhImbue == proto.Rogue_Options_WoundPoison,
		rogue.Options.OhImbue == proto.Rogue_Options_WoundPoison,
	)
	if procMask == core.ProcMaskUnknown {
		return
	}

	const basePPM = 0.5 / (1.4 / 60) // ~21.43, the former 50% normalized to a 1.4 speed weapon
	rogue.woundPoisonPPMM = rogue.AutoAttacks.NewPPMManager(basePPM, procMask)

	rogue.RegisterAura(core.Aura{
		Label:    "Wound Poison",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !result.Landed() || !spell.ProcMask.Matches(procMask) {
				return
			}
			if rogue.woundPoisonPPMM.Proc(sim, spell.ProcMask, "Wound Poison") {
				rogue.WoundPoison[NormalProc].Cast(sim, result.Target)
			}
		},
	})
}

type PoisonProcSource int

const (
	NormalProc PoisonProcSource = iota
	DeadlyProc
	ShivProc
)

func (rogue *Rogue) makeInstantPoison(procSource PoisonProcSource) *core.Spell {
	isShivProc := procSource == ShivProc

	return rogue.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: 57968, Tag: int32(procSource)},
		SpellSchool: core.SpellSchoolNature,
		ProcMask:    core.ProcMaskSpellDamage,

		DamageMultiplier: []float64{1, 1.07, 1.14, 1.20}[rogue.Talents.VilePoisons],
		CritMultiplier:   rogue.SpellCritMultiplier(),
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := 300 + 0.1*spell.MeleeAttackPower()
			if isShivProc {
				spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMagicHit)
			} else {
				spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
			}
		},
	})
}

func (rogue *Rogue) makeWoundPoison(procSource PoisonProcSource) *core.Spell {
	isShivProc := procSource == ShivProc

	return rogue.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: 57975, Tag: int32(procSource)},
		SpellSchool: core.SpellSchoolNature,
		ProcMask:    core.ProcMaskSpellDamage,

		DamageMultiplier: []float64{1, 1.07, 1.14, 1.20}[rogue.Talents.VilePoisons],
		CritMultiplier:   rogue.SpellCritMultiplier(),
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := 231 + 0.04*spell.MeleeAttackPower()

			var result *core.SpellResult
			if isShivProc {
				result = spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMagicHit)
			} else {
				result = spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
			}

			if result.Landed() {
				rogue.woundPoisonDebuffAuras[target.Index].Activate(sim)
			}
		},
	})
}

var WoundPoisonActionID = core.ActionID{SpellID: 57975}

func (rogue *Rogue) registerWoundPoisonSpell() {
	woundPoisonDebuffAura := core.Aura{
		Label:    "WoundPoison-" + strconv.Itoa(int(rogue.Index)),
		ActionID: WoundPoisonActionID,
		Duration: time.Second * 15,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			if rogue.Talents.SavageCombat > 0 {
				rogue.savageCombatDebuffAuras[aura.Unit.Index].Activate(sim)
			}
			if rogue.Talents.MasterPoisoner > 0 {
				rogue.masterPoisonerDebuffAuras[aura.Unit.Index].Activate(sim)
			}
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			if rogue.Talents.SavageCombat > 0 {
				rogue.savageCombatDebuffAuras[aura.Unit.Index].Deactivate(sim)
			}
			if rogue.Talents.MasterPoisoner > 0 {
				rogue.masterPoisonerDebuffAuras[aura.Unit.Index].Deactivate(sim)
			}
		},
	}
	numTargets := rogue.Env.GetNumTargets()
	for i := int32(0); i < numTargets; i++ {
		target := rogue.Env.GetTargetUnit(i)
		rogue.woundPoisonDebuffAuras = append(rogue.woundPoisonDebuffAuras, target.RegisterAura(woundPoisonDebuffAura))
	}
	rogue.WoundPoison = [3]*core.Spell{
		rogue.makeWoundPoison(NormalProc),
		rogue.makeWoundPoison(DeadlyProc),
		rogue.makeWoundPoison(ShivProc),
	}
}

func (rogue *Rogue) registerInstantPoisonSpell() {
	rogue.InstantPoison = [3]*core.Spell{
		rogue.makeInstantPoison(NormalProc),
		rogue.makeInstantPoison(DeadlyProc),
		rogue.makeInstantPoison(ShivProc),
	}
}

func (rogue *Rogue) GetDeadlyPoisonProcChance(mask core.ProcMask) float64 {
	if mask.Matches(core.ProcMaskMeleeMH) && rogue.Options.MhImbue != proto.Rogue_Options_DeadlyPoison {
		return 0.0
	}
	if mask.Matches(core.ProcMaskMeleeOH) && rogue.Options.OhImbue != proto.Rogue_Options_DeadlyPoison {
		return 0.0
	}
	return 0.3 + 0.04*float64(rogue.Talents.ImprovedPoisons) + rogue.deadlyPoisonProcChanceBonus
}

func (rogue *Rogue) UpdateInstantPoisonPPM(bonusChance float64) {
	procMask := core.GetMeleeProcMaskForHands(
		rogue.Options.MhImbue == proto.Rogue_Options_InstantPoison,
		rogue.Options.OhImbue == proto.Rogue_Options_InstantPoison)

	const basePPM = 0.2 / (1.4 / 60) // ~8.57, the former 20% normalized to a 1.4 speed weapon

	ppm := basePPM * (1 + float64(rogue.Talents.ImprovedPoisons)*0.1 + bonusChance)
	rogue.instantPoisonPPMM = rogue.AutoAttacks.NewPPMManager(ppm, procMask)
}

func (rogue *Rogue) applyInstantPoison() {
	procMask := core.GetMeleeProcMaskForHands(
		rogue.Options.MhImbue == proto.Rogue_Options_InstantPoison,
		rogue.Options.OhImbue == proto.Rogue_Options_InstantPoison)

	if procMask == core.ProcMaskUnknown {
		return
	}
	rogue.UpdateInstantPoisonPPM(0)
	rogue.RegisterAura(core.Aura{
		Label:    "Instant Poison",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !result.Landed() || !spell.ProcMask.Matches(procMask) {
				return
			}
			if rogue.instantPoisonPPMM.Proc(sim, spell.ProcMask, "Instant Poison") {
				rogue.InstantPoison[NormalProc].Cast(sim, result.Target)
			}
		},
	})
}
