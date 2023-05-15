package warrior

import (
	"time"

	"github.com/wowsims/wotlk/sim/core"
)

func (warrior *Warrior) registerDemoralizingShoutSpell() {
	warrior.DemoralizingShoutAuras = warrior.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return core.DemoralizingShoutAura(target, warrior.Talents.BoomingVoice, warrior.Talents.ImprovedDemoralizingShout)
	})

	warrior.DemoralizingShout = warrior.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: 25203},
		SpellSchool: core.SpellSchoolPhysical,
		ProcMask:    core.ProcMaskEmpty,

		RageCost: core.RageCostOptions{
			Cost: 10 - float64(warrior.Talents.FocusedRage),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
		},

		ThreatMultiplier: 1,
		FlatThreatBonus:  63.2,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			for _, aoeTarget := range sim.Encounter.TargetUnits {
				result := spell.CalcAndDealOutcome(sim, aoeTarget, spell.OutcomeMagicHit)
				if result.Landed() {
					warrior.DemoralizingShoutAuras.Get(aoeTarget).Activate(sim)
				}
			}
		},
	})
}

func (warrior *Warrior) ShouldDemoralizingShout(sim *core.Simulation, target *core.Unit, filler bool, maintainOnly bool) bool {
	if !warrior.DemoralizingShout.CanCast(sim, target) {
		return false
	}

	if filler {
		return true
	}

	return maintainOnly &&
		warrior.DemoralizingShoutAuras.Get(target).ShouldRefreshExclusiveEffects(sim, time.Second*2)
}
