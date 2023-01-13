package rogue

import (
	"math"
	"time"

	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/proto"
	"github.com/wowsims/wotlk/sim/core/stats"
)

func (rogue *Rogue) ApplyTalents() {
	rogue.applyMurder()
	rogue.applySealFate()
	rogue.applyWeaponSpecializations()
	rogue.applyCombatPotency()
	rogue.applyFocusedAttacks()

	rogue.AddStat(stats.Dodge, core.DodgeRatingPerDodgeChance*2*float64(rogue.Talents.LightningReflexes))
	rogue.PseudoStats.MeleeSpeedMultiplier *= []float64{1, 1.03, 1.06, 1.10}[rogue.Talents.LightningReflexes]
	rogue.AddStat(stats.Parry, core.ParryRatingPerParryChance*2*float64(rogue.Talents.Deflection))
	rogue.AddStat(stats.MeleeCrit, core.CritRatingPerCritChance*1*float64(rogue.Talents.Malice))
	rogue.AddStat(stats.MeleeHit, core.MeleeHitRatingPerHitChance*1*float64(rogue.Talents.Precision))
	rogue.AddStat(stats.SpellHit, core.SpellHitRatingPerHitChance*1*float64(rogue.Talents.Precision))
	rogue.AddStat(stats.Expertise, core.ExpertisePerQuarterPercentReduction*5*float64(rogue.Talents.WeaponExpertise))
	rogue.AddStat(stats.ArmorPenetration, core.ArmorPenPerPercentArmor*3*float64(rogue.Talents.SerratedBlades))
	rogue.AutoAttacks.OHConfig.DamageMultiplier *= rogue.dwsMultiplier()

	if rogue.Talents.Deadliness > 0 {
		rogue.MultiplyStat(stats.AttackPower, 1.0+0.02*float64(rogue.Talents.Deadliness))
	}

	if rogue.Talents.SavageCombat > 0 {
		rogue.MultiplyStat(stats.AttackPower, 1.0+0.02*float64(rogue.Talents.SavageCombat))
	}

	if rogue.Talents.SinisterCalling > 0 {
		rogue.MultiplyStat(stats.Agility, 1.0+0.03*float64(rogue.Talents.SinisterCalling))
	}

	rogue.registerOverkillCD()
	rogue.registerHungerForBlood()
	rogue.registerColdBloodCD()
	rogue.registerBladeFlurryCD()
	rogue.registerAdrenalineRushCD()
	rogue.registerKillingSpreeCD()
}

// dwsMultiplier returns the offhand damage multiplier
func (rogue *Rogue) dwsMultiplier() float64 {
	return 1 + 0.1*float64(rogue.Talents.DualWieldSpecialization)
}

func getRelentlessStrikesSpellID(talentPoints int32) int32 {
	if talentPoints == 1 {
		return 14179
	}
	return 58420 + talentPoints
}

func (rogue *Rogue) makeFinishingMoveEffectApplier() func(sim *core.Simulation, numPoints int32) {
	ruthlessnessMetrics := rogue.NewComboPointMetrics(core.ActionID{SpellID: 14161})
	relentlessStrikesMetrics := rogue.NewEnergyMetrics(core.ActionID{SpellID: getRelentlessStrikesSpellID(rogue.Talents.RelentlessStrikes)})

	return func(sim *core.Simulation, numPoints int32) {
		if t := rogue.Talents.Ruthlessness; t > 0 {
			if sim.RandomFloat("Ruthlessness") < 0.2*float64(t) {
				rogue.AddComboPoints(sim, 1, ruthlessnessMetrics)
			}
		}
		if t := rogue.Talents.RelentlessStrikes; t > 0 {
			if sim.RandomFloat("RelentlessStrikes") < 0.04*float64(t*numPoints) {
				rogue.AddEnergy(sim, 25, relentlessStrikesMetrics)
			}
		}
	}
}

func (rogue *Rogue) makeCostModifier() func(baseCost float64) float64 {
	if rogue.HasSetBonus(ItemSetBonescythe, 4) {
		return func(baseCost float64) float64 {
			return math.RoundToEven(0.95 * baseCost)
		}
	}
	return func(baseCost float64) float64 {
		return baseCost
	}
}

func (rogue *Rogue) applyMurder() {
	rogue.PseudoStats.DamageDealtMultiplier *= rogue.murderMultiplier()
}

func (rogue *Rogue) murderMultiplier() float64 {
	return 1.0 + 0.02*float64(rogue.Talents.Murder)
}

func (rogue *Rogue) registerHungerForBlood() {
	if !rogue.Talents.HungerForBlood {
		return
	}
	actionID := core.ActionID{SpellID: 51662}
	multiplier := 1.05
	if rogue.HasMajorGlyph(proto.RogueMajorGlyph_GlyphOfHungerForBlood) {
		multiplier += 0.03
	}
	rogue.HungerForBloodAura = rogue.RegisterAura(core.Aura{
		Label:    "Hunger for Blood",
		ActionID: actionID,
		Duration: time.Minute,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			rogue.PseudoStats.DamageDealtMultiplier *= multiplier
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			rogue.PseudoStats.DamageDealtMultiplier *= 1 / multiplier
		},
	})

	rogue.HungerForBlood = rogue.RegisterSpell(core.SpellConfig{
		ActionID: actionID,

		EnergyCost: core.EnergyCostOptions{
			Cost: 15,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
		},
		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			rogue.HungerForBloodAura.Activate(sim)
		},
	})
}

func (rogue *Rogue) preyOnTheWeakMultiplier(_ *core.Unit) float64 {
	// TODO: Use the following predicate if/when health values are modeled,
	//  but note that this would have to be applied dynamically in that case.
	//if rogue.CurrentTarget != nil &&
	//rogue.CurrentTarget.HasHealthBar() &&
	//rogue.CurrentTarget.CurrentHealthPercent() < rogue.CurrentHealthPercent()
	return 1 + 0.04*float64(rogue.Talents.PreyOnTheWeak)
}

func (rogue *Rogue) registerColdBloodCD() {
	if !rogue.Talents.ColdBlood {
		return
	}

	actionID := core.ActionID{SpellID: 14177}
	var affectedSpells []*core.Spell

	coldBloodAura := rogue.RegisterAura(core.Aura{
		Label:    "Cold Blood",
		ActionID: actionID,
		Duration: core.NeverExpires,
		OnInit: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range rogue.Spellbook {
				if spell.Flags.Matches(SpellFlagBuilder | SpellFlagFinisher) {
					affectedSpells = append(affectedSpells, spell)
				}
			}
		},
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range affectedSpells {
				spell.BonusCritRating += 100 * core.CritRatingPerCritChance
			}
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range affectedSpells {
				spell.BonusCritRating -= 100 * core.CritRatingPerCritChance
			}
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			for _, affectedSpell := range affectedSpells {
				if spell == affectedSpell {
					aura.Deactivate(sim)
				}
			}
		},
	})

	coldBloodSpell := rogue.RegisterSpell(core.SpellConfig{
		ActionID: actionID,

		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    rogue.NewTimer(),
				Duration: time.Minute * 3,
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			coldBloodAura.Activate(sim)
		},
	})

	rogue.AddMajorCooldown(core.MajorCooldown{
		Spell: coldBloodSpell,
		Type:  core.CooldownTypeDPS,
	})
}

func (rogue *Rogue) applySealFate() {
	if rogue.Talents.SealFate == 0 {
		return
	}

	procChance := 0.2 * float64(rogue.Talents.SealFate)
	cpMetrics := rogue.NewComboPointMetrics(core.ActionID{SpellID: 14195})

	rogue.RegisterAura(core.Aura{
		Label:    "Seal Fate",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !spell.Flags.Matches(SpellFlagBuilder) {
				return
			}

			if !result.Outcome.Matches(core.OutcomeCrit) {
				return
			}

			if sim.Proc(procChance, "Seal Fate") {
				rogue.AddComboPoints(sim, 1, cpMetrics)
			}
		},
	})
}

func (rogue *Rogue) applyWeaponSpecializations() {
	mhWeapon := rogue.GetMHWeapon()
	ohWeapon := rogue.GetOHWeapon()
	// https://wotlk.wowhead.com/spell=13964/sword-specialization, proc mask = 20.
	hackAndSlashMask := core.ProcMaskUnknown
	if mhWeapon != nil && mhWeapon.ID != 0 {
		switch mhWeapon.WeaponType {
		case proto.WeaponType_WeaponTypeSword, proto.WeaponType_WeaponTypeAxe:
			hackAndSlashMask |= core.ProcMaskMeleeMH
		case proto.WeaponType_WeaponTypeDagger, proto.WeaponType_WeaponTypeFist:
			rogue.OnSpellRegistered(func(spell *core.Spell) {
				if spell.ProcMask.Matches(core.ProcMaskMeleeMH) {
					spell.BonusCritRating += 1 * core.CritRatingPerCritChance * float64(rogue.Talents.CloseQuartersCombat)
				}
			})
		case proto.WeaponType_WeaponTypeMace:
			rogue.OnSpellRegistered(func(spell *core.Spell) {
				if spell.ProcMask.Matches(core.ProcMaskMeleeMH) {
					spell.BonusArmorPenRating += 3 * core.ArmorPenPerPercentArmor * float64(rogue.Talents.MaceSpecialization)
				}
			})
		}
	}
	if ohWeapon != nil && ohWeapon.ID != 0 {
		switch ohWeapon.WeaponType {
		case proto.WeaponType_WeaponTypeSword, proto.WeaponType_WeaponTypeAxe:
			hackAndSlashMask |= core.ProcMaskMeleeOH
		case proto.WeaponType_WeaponTypeDagger, proto.WeaponType_WeaponTypeFist:
			rogue.OnSpellRegistered(func(spell *core.Spell) {
				if spell.ProcMask.Matches(core.ProcMaskMeleeOH) {
					spell.BonusCritRating += 1 * core.CritRatingPerCritChance * float64(rogue.Talents.CloseQuartersCombat)
				}
			})
		case proto.WeaponType_WeaponTypeMace:
			rogue.OnSpellRegistered(func(spell *core.Spell) {
				if spell.ProcMask.Matches(core.ProcMaskMeleeOH) {
					spell.BonusArmorPenRating += 3 * core.ArmorPenPerPercentArmor * float64(rogue.Talents.MaceSpecialization)
				}
			})
		}
	}

	rogue.registerHackAndSlash(hackAndSlashMask)
}

func (rogue *Rogue) applyCombatPotency() {
	if rogue.Talents.CombatPotency == 0 {
		return
	}

	const procChance = 0.2
	energyBonus := 3.0 * float64(rogue.Talents.CombatPotency)
	energyMetrics := rogue.NewEnergyMetrics(core.ActionID{SpellID: 35553})

	rogue.RegisterAura(core.Aura{
		Label:    "Combat Potency",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !result.Landed() {
				return
			}

			// https://wotlk.wowhead.com/spell=35553/combat-potency, proc mask = 8838608.
			if !spell.ProcMask.Matches(core.ProcMaskMeleeOH) {
				return
			}

			// Fan of Knives OH hits do not proc combat potency
			if spell.IsSpellAction(FanOfKnivesSpellID) {
				return
			}

			if sim.RandomFloat("Combat Potency") > procChance {
				return
			}

			rogue.AddEnergy(sim, energyBonus, energyMetrics)
		},
	})
}

func (rogue *Rogue) applyFocusedAttacks() {
	if rogue.Talents.FocusedAttacks == 0 {
		return
	}

	procChance := []float64{0, 0.33, 0.66, 1}[rogue.Talents.FocusedAttacks]
	energyMetrics := rogue.NewEnergyMetrics(core.ActionID{SpellID: 51637})

	rogue.RegisterAura(core.Aura{
		Label:    "Focused Attacks",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !spell.ProcMask.Matches(core.ProcMaskMelee) || !result.DidCrit() {
				return
			}
			// Fan of Knives OH hits do not trigger focused attacks
			if spell.ProcMask.Matches(core.ProcMaskMeleeOH) && spell.IsSpellAction(FanOfKnivesSpellID) {
				return
			}
			if sim.Proc(procChance, "Focused Attacks") {
				rogue.AddEnergy(sim, 2, energyMetrics)
			}
		},
	})
}

var BladeFlurryActionID = core.ActionID{SpellID: 13877}

func (rogue *Rogue) registerBladeFlurryCD() {
	if !rogue.Talents.BladeFlurry {
		return
	}

	var curDmg float64
	bfHit := rogue.RegisterSpell(core.SpellConfig{
		ActionID:    BladeFlurryActionID,
		SpellSchool: core.SpellSchoolPhysical,
		ProcMask:    core.ProcMaskEmpty, // No proc mask, so it won't proc itself.
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagNoOnCastComplete | core.SpellFlagIgnoreAttackerModifiers,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealDamage(sim, target, curDmg, spell.OutcomeAlwaysHit)
		},
	})

	const hasteBonus = 1.2
	const inverseHasteBonus = 1 / 1.2

	dur := time.Second * 15

	rogue.BladeFlurryAura = rogue.RegisterAura(core.Aura{
		Label:    "Blade Flurry",
		ActionID: BladeFlurryActionID,
		Duration: dur,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			rogue.MultiplyMeleeSpeed(sim, hasteBonus)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			rogue.MultiplyMeleeSpeed(sim, inverseHasteBonus)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if sim.GetNumTargets() < 2 {
				return
			}
			if result.Damage == 0 || !spell.ProcMask.Matches(core.ProcMaskMelee) {
				return
			}
			// Fan of Knives off-hand hits are not cloned
			if spell.IsSpellAction(FanOfKnivesSpellID) && spell.ProcMask.Matches(core.ProcMaskMeleeOH) {
				return
			}

			// Undo armor reduction to get the raw damage value.
			curDmg = result.Damage / result.ResistanceMultiplier

			bfHit.Cast(sim, rogue.Env.NextTargetUnit(result.Target))
			bfHit.SpellMetrics[result.Target.UnitIndex].Casts--
		},
	})

	cooldownDur := time.Minute * 2
	rogue.BladeFlurry = rogue.RegisterSpell(core.SpellConfig{
		ActionID: BladeFlurryActionID,

		EnergyCost: core.EnergyCostOptions{
			Cost: 25,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    rogue.NewTimer(),
				Duration: cooldownDur,
			},
			ModifyCast: func(s1 *core.Simulation, s2 *core.Spell, c *core.Cast) {
				if rogue.HasMajorGlyph(proto.RogueMajorGlyph_GlyphOfBladeFlurry) {
					c.Cost = 0
				}
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			rogue.BladeFlurryAura.Activate(sim)
		},
	})

	rogue.AddMajorCooldown(core.MajorCooldown{
		Spell:    rogue.BladeFlurry,
		Type:     core.CooldownTypeDPS,
		Priority: core.CooldownPriorityDefault,
		CanActivate: func(sim *core.Simulation, character *core.Character) bool {
			return rogue.CurrentEnergy() >= rogue.BladeFlurry.DefaultCast.Cost
		},
		ShouldActivate: func(sim *core.Simulation, character *core.Character) bool {
			if sim.GetRemainingDuration() > cooldownDur+dur {
				// We'll have enough time to cast another BF, so use it immediately to make sure we get the 2nd one.
				return true
			}

			// Since this is our last BF, wait until we have SND / procs up.
			sndTimeRemaining := rogue.SliceAndDiceAura.RemainingDuration(sim)
			// TODO: Wait for dst/mongoose procs
			return sndTimeRemaining >= time.Second
		},
	})
}

var AdrenalineRushActionID = core.ActionID{SpellID: 13750}

func (rogue *Rogue) registerAdrenalineRushCD() {
	if !rogue.Talents.AdrenalineRush {
		return
	}

	rogue.AdrenalineRushAura = rogue.RegisterAura(core.Aura{
		Label:    "Adrenaline Rush",
		ActionID: AdrenalineRushActionID,
		Duration: core.TernaryDuration(rogue.HasMajorGlyph(proto.RogueMajorGlyph_GlyphOfAdrenalineRush), time.Second*20, time.Second*15),
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			rogue.ResetEnergyTick(sim)
			rogue.ApplyEnergyTickMultiplier(1.0)
			rogue.rotationItems = rogue.planRotation(sim)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			rogue.ResetEnergyTick(sim)
			rogue.ApplyEnergyTickMultiplier(-1.0)
			rogue.rotationItems = rogue.planRotation(sim)
		},
	})

	adrenalineRushSpell := rogue.RegisterSpell(core.SpellConfig{
		ActionID: AdrenalineRushActionID,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    rogue.NewTimer(),
				Duration: time.Minute * 5,
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			rogue.AdrenalineRushAura.Activate(sim)
		},
	})

	rogue.AddMajorCooldown(core.MajorCooldown{
		Spell:    adrenalineRushSpell,
		Type:     core.CooldownTypeDPS,
		Priority: core.CooldownPriorityBloodlust,
		ShouldActivate: func(sim *core.Simulation, character *core.Character) bool {
			thresh := 45.0
			return rogue.CurrentEnergy() <= thresh
		},
	})
}

func (rogue *Rogue) registerKillingSpreeCD() {
	if !rogue.Talents.KillingSpree {
		return
	}
	rogue.registerKillingSpreeSpell()
}
