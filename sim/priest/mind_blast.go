package priest

import (
	"time"

	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/proto"
	"github.com/wowsims/wotlk/sim/core/stats"
)

func (priest *Priest) registerMindBlastSpell() {
	baseCost := priest.BaseMana*0.17 - core.TernaryFloat64(priest.HasSetBonus(ItemSetValorous, 2), (priest.BaseMana*0.17)*0.1, 0)

	miserySpellCoeff := 0.429 * (1 + 0.05*float64(priest.Talents.Misery))

	normMod := (1 + 0.02*float64(priest.Talents.Darkness)) *
		core.TernaryFloat64(priest.HasSetBonus(ItemSetAbsolution, 4), 1.1, 1)
	swpMod := normMod * (1 + 0.02*float64(priest.Talents.TwistedFaith))

	var replSrc core.ReplenishmentSource
	if priest.Talents.VampiricTouch {
		replSrc = priest.Env.Raid.NewReplenishmentSource(core.ActionID{SpellID: 48160})
	}

	priest.MindBlast = priest.RegisterSpell(core.SpellConfig{
		ActionID:     core.ActionID{SpellID: 48127},
		SpellSchool:  core.SpellSchoolShadow,
		ProcMask:     core.ProcMaskSpellDamage,
		ResourceType: stats.Mana,
		BaseCost:     baseCost,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				Cost:     baseCost * (1 - 0.05*float64(priest.Talents.FocusedMind)),
				GCD:      core.GCDDefault,
				CastTime: time.Millisecond * 1500,
			},
			CD: core.Cooldown{
				Timer:    priest.NewTimer(),
				Duration: time.Second*8 - time.Millisecond*500*time.Duration(priest.Talents.ImprovedMindBlast),
			},
		},

		BonusHitRating:   0 + float64(priest.Talents.ShadowFocus)*1*core.SpellHitRatingPerHitChance,
		BonusCritRating:  float64(priest.Talents.MindMelt) * 2 * core.CritRatingPerCritChance,
		DamageMultiplier: 1,
		CritMultiplier:   priest.SpellCritMultiplier(1, float64(priest.Talents.ShadowPower)/5),
		ThreatMultiplier: 1 - 0.08*float64(priest.Talents.ShadowAffinity),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := sim.Roll(997, 1053) + miserySpellCoeff*spell.SpellPower()

			if priest.ShadowWordPainDot.IsActive() {
				baseDamage *= swpMod
			} else {
				baseDamage *= normMod
			}

			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
			if result.Landed() {
				priest.AddShadowWeavingStack(sim)
			}
			if result.DidCrit() && priest.HasGlyph(int32(proto.PriestMajorGlyph_GlyphOfShadow)) {
				priest.ShadowyInsightAura.Activate(sim)
			}
			if result.DidCrit() && priest.ImprovedSpiritTap != nil {
				priest.ImprovedSpiritTap.Activate(sim)
			}
			spell.DealDamage(sim, result)

			if priest.Talents.VampiricTouch && priest.VampiricTouchDot.IsActive() {
				priest.Env.Raid.ProcReplenishment(sim, replSrc)
			}
		},
	})
}
