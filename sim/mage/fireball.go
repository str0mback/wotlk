package mage

import (
	"strconv"
	"time"

	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/proto"
	"github.com/wowsims/wotlk/sim/core/stats"
)

func (mage *Mage) registerFireballSpell() {
	actionID := core.ActionID{SpellID: 42833}
	baseCost := .19 * mage.BaseMana
	spellCoeff := 1 + 0.05*float64(mage.Talents.EmpoweredFire)

	hasGlyph := mage.HasMajorGlyph(proto.MageMajorGlyph_GlyphOfFireball)

	mage.Fireball = mage.RegisterSpell(core.SpellConfig{
		ActionID:     actionID,
		SpellSchool:  core.SpellSchoolFire,
		ProcMask:     core.ProcMaskSpellDamage,
		Flags:        SpellFlagMage | BarrageSpells | HotStreakSpells,
		MissileSpeed: 22,
		ResourceType: stats.Mana,
		BaseCost:     baseCost,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				Cost: baseCost,
				GCD:  core.GCDDefault,
				CastTime: time.Millisecond*3500 -
					time.Millisecond*100*time.Duration(mage.Talents.ImprovedFireball) -
					core.TernaryDuration(hasGlyph, time.Millisecond*150, 0),
			},
		},

		BonusCritRating: 0 +
			float64(mage.Talents.CriticalMass)*2*core.CritRatingPerCritChance +
			float64(mage.Talents.ImprovedScorch)*core.CritRatingPerCritChance +
			core.TernaryFloat64(mage.HasSetBonus(ItemSetKhadgarsRegalia, 4), 5*core.CritRatingPerCritChance, 0),
		DamageMultiplier: mage.spellDamageMultiplier *
			(1 + 0.02*float64(mage.Talents.SpellImpact)) *
			(1 + .04*float64(mage.Talents.TormentTheWeak)),
		CritMultiplier:   mage.SpellCritMultiplier(1, mage.bonusCritDamage),
		ThreatMultiplier: 1 - 0.1*float64(mage.Talents.BurningSoul),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := sim.Roll(898, 1143) + spellCoeff*spell.SpellPower()
			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				if result.Landed() && !hasGlyph {
					mage.FireballDot.Apply(sim)
				}
				spell.DealDamage(sim, result)
			})
		},
	})

	target := mage.CurrentTarget
	mage.FireballDot = core.NewDot(core.Dot{
		Spell: mage.RegisterSpell(core.SpellConfig{
			ActionID:    actionID,
			SpellSchool: core.SpellSchoolFire,
			ProcMask:    core.ProcMaskSpellDamage,
			Flags:       SpellFlagMage | BarrageSpells | HotStreakSpells,

			DamageMultiplier: mage.Fireball.DamageMultiplier,
			ThreatMultiplier: mage.Fireball.ThreatMultiplier,
		}),
		Aura: target.RegisterAura(core.Aura{
			Label:    "Fireball-" + strconv.Itoa(int(mage.Index)),
			ActionID: actionID,
		}),
		NumberOfTicks: 4,
		TickLength:    time.Second * 2,
		OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, _ bool) {
			dot.SnapshotBaseDamage = 116.0 / 4.0
			dot.SnapshotAttackerMultiplier = dot.Spell.AttackerDamageMultiplier(dot.Spell.Unit.AttackTables[target.UnitIndex])
		},
		OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
			dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
		},
	})
}
