package deathknight

import (
	"time"

	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/proto"
	"github.com/wowsims/wotlk/sim/core/stats"
)

func (dk *Deathknight) registerDeathAndDecaySpell() {
	actionID := core.ActionID{SpellID: 49938}
	glyphBonus := core.TernaryFloat64(dk.HasMajorGlyph(proto.DeathknightMajorGlyph_GlyphOfDeathAndDecay), 1.2, 1.0)

	baseCost := float64(core.NewRuneCost(15, 1, 1, 1, 0))
	dk.DeathAndDecay = dk.RegisterSpell(nil, core.SpellConfig{
		ActionID:     actionID,
		SpellSchool:  core.SpellSchoolShadow,
		ProcMask:     core.ProcMaskSpellDamage,
		ResourceType: stats.RunicPower,
		BaseCost:     baseCost,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:  core.GCDDefault,
				Cost: baseCost,
			},
			ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
				cast.GCD = dk.GetModifiedGCD()
			},
			CD: core.Cooldown{
				Timer:    dk.NewTimer(),
				Duration: time.Second*30 - time.Second*5*time.Duration(dk.Talents.Morbidity),
			},
		},

		DamageMultiplier: glyphBonus * dk.scourgelordsPlateDamageBonus(),
		ThreatMultiplier: 1.9,
		CritMultiplier:   dk.DefaultMeleeCritMultiplier(),

		ApplyEffects: func(sim *core.Simulation, unit *core.Unit, spell *core.Spell) {
			dk.DeathAndDecayDot.Apply(sim)
			dk.DeathAndDecayDot.TickOnce(sim)
		},
	}, func(sim *core.Simulation) bool {
		return dk.CastCostPossible(sim, 0.0, 1, 1, 1) && dk.DeathAndDecay.IsReady(sim)
	}, nil)

	dk.DeathAndDecayDot = core.NewDot(core.Dot{
		Aura: dk.RegisterAura(core.Aura{
			Label:    "Death and Decay",
			ActionID: actionID,
		}),
		NumberOfTicks: 10,
		TickLength:    time.Second * 1,
		OnSnapshot: func(sim *core.Simulation, _ *core.Unit, dot *core.Dot, _ bool) {
			target := dk.CurrentTarget
			dot.SnapshotBaseDamage = 62 + 0.0475*dk.getImpurityBonus(dot.Spell)
			dot.SnapshotCritChance = dot.Spell.SpellCritChance(target)
		},
		OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
			for _, aoeTarget := range sim.Encounter.Targets {
				// DnD recalculates attack multipliers dynamically on every tick so this is here on purpose
				dot.SnapshotAttackerMultiplier = dot.Spell.AttackerDamageMultiplier(dot.Spell.Unit.AttackTables[aoeTarget.UnitIndex]) * dk.RoRTSBonus(&aoeTarget.Unit)
				dot.CalcAndDealPeriodicSnapshotDamage(sim, &aoeTarget.Unit, dot.OutcomeMagicHitAndSnapshotCrit)
			}
		},
	})
	dk.DeathAndDecayDot.Spell = dk.DeathAndDecay.Spell
}
