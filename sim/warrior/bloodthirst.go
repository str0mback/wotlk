package warrior

import (
	"time"

	"github.com/wowsims/wotlk/sim/core"
)

func (warrior *Warrior) registerBloodthirstSpell(cdTimer *core.Timer) {
	warrior.Bloodthirst = warrior.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: 23881},
		SpellSchool: core.SpellSchoolPhysical,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagIncludeTargetBonusDamage | SpellFlagBloodsurge,

		RageCost: core.RageCostOptions{
			Cost:   20,
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    cdTimer,
				Duration: time.Second * 4,
			},
		},

		BonusCritRating:  core.TernaryFloat64(warrior.HasSetBonus(ItemSetSiegebreakerBattlegear, 4), 10, 0) * core.CritRatingPerCritChance,
		DamageMultiplier: 1 + 0.02*float64(warrior.Talents.UnendingFury) + core.TernaryFloat64(warrior.HasSetBonus(ItemSetOnslaughtBattlegear, 4), 0.05, 0),
		CritMultiplier:   warrior.critMultiplier(mh),
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := 0.5 * spell.MeleeAttackPower()
			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)
			if !result.Landed() {
				spell.IssueRefund(sim)
			}
			core.StartDelayedAction(sim, core.DelayedActionOptions{
				DoAt: sim.CurrentTime + warrior.Bloodthirst.CD.Duration,
				OnAction: func(_ *core.Simulation) {
					if warrior.ShouldInstantSlam(sim) {
						warrior.CastSlam(sim, warrior.CurrentTarget)
					} else if warrior.CanBloodthirst(sim) {
						warrior.Bloodthirst.Cast(sim, warrior.CurrentTarget)
					}
				},
			})
		},
	})
}

func (warrior *Warrior) CanBloodthirst(sim *core.Simulation) bool {
	return warrior.Talents.Bloodthirst && warrior.CurrentRage() >= warrior.Bloodthirst.DefaultCast.Cost && warrior.Bloodthirst.IsReady(sim) && warrior.GCD.IsReady(sim)
}
