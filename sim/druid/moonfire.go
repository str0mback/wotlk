package druid

import (
	"strconv"
	"time"

	"github.com/wowsims/wotlk/sim/core/proto"

	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/stats"
)

func (druid *Druid) registerMoonfireSpell() {
	actionID := core.ActionID{SpellID: 48463}
	baseCost := 0.21 * druid.BaseMana

	numTicks := druid.moonfireTicks()

	druid.Moonfire = druid.RegisterSpell(core.SpellConfig{
		ActionID:     core.ActionID{SpellID: 48463},
		SpellSchool:  core.SpellSchoolArcane,
		ProcMask:     core.ProcMaskSpellDamage,
		Flags:        SpellFlagNaturesGrace,
		ResourceType: stats.Mana,
		BaseCost:     baseCost,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				Cost: baseCost * (1 - 0.03*float64(druid.Talents.Moonglow)),
				GCD:  core.GCDDefault,
			},
		},

		BonusCritRating: float64(druid.Talents.ImprovedMoonfire) * 5 * core.CritRatingPerCritChance,
		DamageMultiplier: 1 +
			0.05*float64(druid.Talents.ImprovedMoonfire) +
			[]float64{0.0, 0.03, 0.06, 0.1}[druid.Talents.Moonfury] -
			core.TernaryFloat64(druid.HasMajorGlyph(proto.DruidMajorGlyph_GlyphOfMoonfire), 0.9, 0),

		CritMultiplier:   druid.BalanceCritMultiplier(),
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := sim.Roll(406, 476) + 0.15*spell.SpellPower()
			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
			if result.Landed() {
				druid.MoonfireDot.NumberOfTicks = numTicks
				druid.MoonfireDot.Apply(sim)
			}
			spell.DealDamage(sim, result)
		},
	})

	starfireBonusCrit := float64(druid.Talents.ImprovedInsectSwarm) * core.CritRatingPerCritChance
	dotCanCrit := druid.HasSetBonus(ItemSetMalfurionsRegalia, 2)
	druid.MoonfireDot = core.NewDot(core.Dot{
		Spell: druid.RegisterSpell(core.SpellConfig{
			ActionID:    core.ActionID{SpellID: 48463},
			SpellSchool: core.SpellSchoolArcane,
			ProcMask:    core.ProcMaskSpellDamage,

			DamageMultiplier: 1 +
				0.05*float64(druid.Talents.ImprovedMoonfire) +
				0.01*float64(druid.Talents.Genesis) +
				[]float64{0.0, 0.03, 0.06, 0.1}[druid.Talents.Moonfury] +
				core.TernaryFloat64(druid.HasMajorGlyph(proto.DruidMajorGlyph_GlyphOfMoonfire), 0.75, 0),

			CritMultiplier:   druid.BalanceCritMultiplier(),
			ThreatMultiplier: 1,
		}),
		Aura: druid.CurrentTarget.RegisterAura(core.Aura{
			Label:    "Moonfire Dot-" + strconv.Itoa(int(druid.Index)),
			ActionID: actionID,
			OnGain: func(aura *core.Aura, sim *core.Simulation) {
				druid.Starfire.BonusCritRating += starfireBonusCrit
			},
			OnExpire: func(aura *core.Aura, sim *core.Simulation) {
				druid.Starfire.BonusCritRating -= starfireBonusCrit
			},
		}),
		NumberOfTicks: druid.moonfireTicks(),
		TickLength:    time.Second * 3,

		OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
			dot.SnapshotBaseDamage = 200 + 0.13*dot.Spell.SpellPower()
			attackTable := dot.Spell.Unit.AttackTables[target.UnitIndex]
			dot.SnapshotCritChance = dot.Spell.SpellCritChance(target)
			dot.SnapshotAttackerMultiplier = dot.Spell.AttackerDamageMultiplier(attackTable)
		},
		OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
			if dotCanCrit {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeSnapshotCrit)
			} else {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			}
		},
	})
}

func (druid *Druid) moonfireTicks() int32 {
	return 4 +
		core.TernaryInt32(druid.Talents.NaturesSplendor, 1, 0) +
		core.TernaryInt32(druid.HasSetBonus(ItemSetThunderheartRegalia, 2), 1, 0)
}
