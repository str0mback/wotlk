package mage

import (
	"time"

	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/proto"
	"github.com/wowsims/wotlk/sim/core/stats"
)

func (mage *Mage) registerFrostboltSpell() {
	baseCost := .11 * mage.BaseMana
	spellCoeff := (3.0/3.5)*0.95 + 0.05*float64(mage.Talents.EmpoweredFrostbolt)

	replProcChance := float64(mage.Talents.EnduringWinter) / 3
	var replSrc core.ReplenishmentSource
	if replProcChance > 0 {
		replSrc = mage.Env.Raid.NewReplenishmentSource(core.ActionID{SpellID: 44561})
	}

	mage.Frostbolt = mage.RegisterSpell(core.SpellConfig{
		ActionID:     core.ActionID{SpellID: 42842},
		SpellSchool:  core.SpellSchoolFrost,
		ProcMask:     core.ProcMaskSpellDamage,
		Flags:        SpellFlagMage | BarrageSpells,
		ResourceType: stats.Mana,
		BaseCost:     baseCost,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				Cost: baseCost,

				GCD:      core.GCDDefault,
				CastTime: time.Second*3 - time.Millisecond*100*time.Duration(mage.Talents.ImprovedFrostbolt+mage.Talents.EmpoweredFrostbolt),
			},
		},

		BonusCritRating: 0 +
			core.TernaryFloat64(mage.HasSetBonus(ItemSetKhadgarsRegalia, 4), 5*core.CritRatingPerCritChance, 0),
		DamageMultiplier: mage.spellDamageMultiplier *
			(1 + .01*float64(mage.Talents.ChilledToTheBone)) *
			core.TernaryFloat64(mage.HasMajorGlyph(proto.MageMajorGlyph_GlyphOfFrostbolt), 1.05, 1),
		CritMultiplier:   mage.SpellCritMultiplier(1, 0.25*float64(mage.Talents.SpellPower)+float64(mage.Talents.IceShards)/3),
		ThreatMultiplier: 1 - (0.1/3)*float64(mage.Talents.FrostChanneling),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := sim.Roll(799, 861) + spellCoeff*spell.SpellPower()
			spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)

			if replProcChance == 1 || sim.RandomFloat("Enduring Winter") < replProcChance {
				mage.Env.Raid.ProcReplenishment(sim, replSrc)
			}
		},
	})
}
