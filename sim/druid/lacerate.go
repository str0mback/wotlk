package druid

import (
	"strconv"
	"time"

	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/stats"
)

func (druid *Druid) registerLacerateSpell() {
	actionID := core.ActionID{SpellID: 48568}

	cost := 15.0 - float64(druid.Talents.ShreddingAttacks)
	refundAmount := cost * 0.8

	tickDamage := 320.0 / 5
	initialDamage := 88.0
	if druid.Equip[core.ItemSlotRanged].ID == 27744 { // Idol of Ursoc
		tickDamage += 8
		initialDamage += 8
	}

	mangleAura := core.MangleAura(druid.CurrentTarget)

	druid.Lacerate = druid.RegisterSpell(core.SpellConfig{
		ActionID:     actionID,
		SpellSchool:  core.SpellSchoolPhysical,
		ProcMask:     core.ProcMaskMeleeMHSpecial,
		Flags:        core.SpellFlagMeleeMetrics,
		ResourceType: stats.Rage,
		BaseCost:     cost,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				Cost: cost,
				GCD:  core.GCDDefault,
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 1 *
			core.TernaryFloat64(druid.HasSetBonus(ItemSetLasherweaveBattlegear, 2), 1.2, 1) *
			core.TernaryFloat64(druid.HasSetBonus(ItemSetDreamwalkerBattlegear, 2), 1.05, 1),
		CritMultiplier:   druid.MeleeCritMultiplier(Bear),
		ThreatMultiplier: 0.5,
		FlatThreatBonus:  267,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := initialDamage + 0.01*spell.MeleeAttackPower()
			if mangleAura.IsActive() {
				baseDamage *= 1.3
			}

			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)

			if result.Landed() {
				if druid.LacerateDot.IsActive() {
					druid.LacerateDot.Refresh(sim)
					druid.LacerateDot.AddStack(sim)
					druid.LacerateDot.TakeSnapshot(sim, true)
				} else {
					druid.LacerateDot.Activate(sim)
					druid.LacerateDot.SetStacks(sim, 1)
					druid.LacerateDot.TakeSnapshot(sim, true)
				}
			} else {
				druid.AddRage(sim, refundAmount, druid.RageRefundMetrics)
			}

			spell.DealDamage(sim, result)
		},
	})

	druid.LacerateDot = core.NewDot(core.Dot{
		Spell: druid.RegisterSpell(core.SpellConfig{
			ActionID:    actionID,
			SpellSchool: core.SpellSchoolPhysical,
			ProcMask:    core.ProcMaskMeleeMHSpecial,
			Flags:       core.SpellFlagMeleeMetrics,

			DamageMultiplier: 1 *
				core.TernaryFloat64(druid.HasSetBonus(ItemSetLasherweaveBattlegear, 2), 1.2, 1) *
				core.TernaryFloat64(druid.HasSetBonus(ItemSetMalfurionsBattlegear, 2), 1.05, 1),
			CritMultiplier:   druid.MeleeCritMultiplier(Bear),
			ThreatMultiplier: 0.5,
		}),
		Aura: druid.CurrentTarget.RegisterAura(druid.applyRendAndTear(core.Aura{
			Label:     "Lacerate-" + strconv.Itoa(int(druid.Index)),
			ActionID:  actionID,
			MaxStacks: 5,
			Duration:  time.Second * 15,
		})),
		NumberOfTicks: 5,
		TickLength:    time.Second * 3,

		OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
			dot.SnapshotBaseDamage = tickDamage + 0.01*dot.Spell.MeleeAttackPower()
			dot.SnapshotBaseDamage *= float64(dot.Aura.GetStacks())

			if !isRollover {
				attackTable := dot.Spell.Unit.AttackTables[target.UnitIndex]
				dot.SnapshotCritChance = dot.Spell.PhysicalCritChance(target, attackTable)
				dot.SnapshotAttackerMultiplier = dot.Spell.AttackerDamageMultiplier(attackTable)
			}
		},
		OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
			if druid.Talents.PrimalGore {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeSnapshotCrit)
			} else {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.Spell.OutcomeAlwaysHit)
			}
		},
	})
}

func (druid *Druid) CanLacerate(sim *core.Simulation) bool {
	return druid.CurrentRage() >= druid.Lacerate.DefaultCast.Cost
}
