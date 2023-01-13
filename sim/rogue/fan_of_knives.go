package rogue

import (
	"time"

	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/proto"
)

const FanOfKnivesSpellID int32 = 51723

func (rogue *Rogue) makeFanOfKnivesWeaponHitSpell(isMH bool) *core.Spell {
	var procMask core.ProcMask
	var weaponMultiplier float64
	if isMH {
		weaponMultiplier = core.TernaryFloat64(rogue.Equip[proto.ItemSlot_ItemSlotMainHand].WeaponType == proto.WeaponType_WeaponTypeDagger, 1.05, 0.7)
		procMask = core.ProcMaskMeleeMHSpecial
	} else {
		weaponMultiplier = core.TernaryFloat64(rogue.Equip[proto.ItemSlot_ItemSlotOffHand].WeaponType == proto.WeaponType_WeaponTypeDagger, 1.05, 0.7)
		weaponMultiplier *= rogue.dwsMultiplier()
		procMask = core.ProcMaskMeleeOHSpecial
	}

	return rogue.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: FanOfKnivesSpellID},
		SpellSchool: core.SpellSchoolPhysical,
		ProcMask:    procMask,
		Flags:       core.SpellFlagMeleeMetrics,

		DamageMultiplier: weaponMultiplier * (1 +
			0.02*float64(rogue.Talents.FindWeakness) +
			core.TernaryFloat64(rogue.HasMajorGlyph(proto.RogueMajorGlyph_GlyphOfFanOfKnives), 0.2, 0.0)),
		CritMultiplier:   rogue.MeleeCritMultiplier(false),
		ThreatMultiplier: 1,
	})
}

func (rogue *Rogue) registerFanOfKnives() {
	mhSpell := rogue.makeFanOfKnivesWeaponHitSpell(true)
	ohSpell := rogue.makeFanOfKnivesWeaponHitSpell(false)
	results := make([]*core.SpellResult, len(rogue.Env.Encounter.Targets))

	rogue.FanOfKnives = rogue.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: FanOfKnivesSpellID},
		SpellSchool: core.SpellSchoolPhysical,
		Flags:       core.SpellFlagMeleeMetrics,

		EnergyCost: core.EnergyCostOptions{
			Cost: 50,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
		},

		ApplyEffects: func(sim *core.Simulation, unit *core.Unit, spell *core.Spell) {
			// Calc and apply all OH hits first, because MH hits can benefit from a OH felstriker proc.
			for i, aoeTarget := range sim.Encounter.Targets {
				baseDamage := ohSpell.Unit.OHWeaponDamage(sim, ohSpell.MeleeAttackPower())
				baseDamage *= sim.Encounter.AOECapMultiplier()
				results[i] = ohSpell.CalcDamage(sim, &aoeTarget.Unit, baseDamage, ohSpell.OutcomeMeleeWeaponSpecialHitAndCrit)
			}
			for i := range sim.Encounter.Targets {
				ohSpell.DealDamage(sim, results[i])
			}

			for i, aoeTarget := range sim.Encounter.Targets {
				baseDamage := mhSpell.Unit.MHWeaponDamage(sim, mhSpell.MeleeAttackPower())
				baseDamage *= sim.Encounter.AOECapMultiplier()
				results[i] = mhSpell.CalcDamage(sim, &aoeTarget.Unit, baseDamage, mhSpell.OutcomeMeleeWeaponSpecialHitAndCrit)
			}
			for i := range sim.Encounter.Targets {
				mhSpell.DealDamage(sim, results[i])
			}
		},
	})
}
