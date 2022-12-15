package warlock

import (
	"strconv"
	"time"

	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/proto"
	"github.com/wowsims/wotlk/sim/core/stats"
)

func (warlock *Warlock) registerCurseOfElementsSpell() {
	if warlock.Rotation.Curse != proto.Warlock_Rotation_Elements {
		return
	}
	baseCost := 0.1 * warlock.BaseMana
	warlock.CurseOfElementsAura = core.CurseOfElementsAura(warlock.CurrentTarget)

	warlock.CurseOfElements = warlock.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: 47865},
		SpellSchool: core.SpellSchoolShadow,
		ProcMask:    core.ProcMaskEmpty,

		ResourceType: stats.Mana,
		BaseCost:     baseCost,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				Cost: baseCost * (1 - 0.02*float64(warlock.Talents.Suppression)),
				GCD:  core.GCDDefault - core.TernaryDuration(warlock.Talents.AmplifyCurse, 1, 0)*500*time.Millisecond,
			},
		},

		ThreatMultiplier: 1 - 0.1*float64(warlock.Talents.ImprovedDrainSoul),
		FlatThreatBonus:  0, // TODO

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealOutcome(sim, target, spell.OutcomeMagicHit)
			if result.Landed() {
				warlock.CurseOfElementsAura.Activate(sim)
			}
		},
	})
}

func (warlock *Warlock) ShouldCastCurseOfElements(sim *core.Simulation, target *core.Unit, curse proto.Warlock_Rotation_Curse) bool {
	return curse == proto.Warlock_Rotation_Elements && !warlock.CurseOfElementsAura.IsActive()
}

func (warlock *Warlock) registerCurseOfWeaknessSpell() {
	baseCost := 0.1 * warlock.BaseMana
	warlock.CurseOfWeaknessAura = core.CurseOfWeaknessAura(warlock.CurrentTarget, warlock.Talents.ImprovedCurseOfWeakness)
	warlock.CurseOfWeaknessAura.Duration = time.Minute * 2

	warlock.CurseOfWeakness = warlock.RegisterSpell(core.SpellConfig{
		ActionID:     core.ActionID{SpellID: 50511},
		SpellSchool:  core.SpellSchoolShadow,
		ProcMask:     core.ProcMaskEmpty,
		ResourceType: stats.Mana,
		BaseCost:     baseCost,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				Cost: baseCost * (1 - 0.02*float64(warlock.Talents.Suppression)),
				GCD:  core.GCDDefault - core.TernaryDuration(warlock.Talents.AmplifyCurse, 1, 0)*500*time.Millisecond,
			},
		},

		ThreatMultiplier: 1 - 0.1*float64(warlock.Talents.ImprovedDrainSoul),
		FlatThreatBonus:  0, // TODO

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealOutcome(sim, target, spell.OutcomeMagicHit)
			if result.Landed() {
				warlock.CurseOfWeaknessAura.Activate(sim)
			}
		},
	})
}

func (warlock *Warlock) registerCurseOfTonguesSpell() {
	actionID := core.ActionID{SpellID: 11719}
	baseCost := 0.04 * warlock.BaseMana

	// Empty aura so we can simulate cost/time to keep tongues up
	warlock.CurseOfTonguesAura = warlock.CurrentTarget.GetOrRegisterAura(core.Aura{
		Label:    "Curse of Tongues",
		ActionID: actionID,
		Duration: time.Second * 30,
	})

	warlock.CurseOfTongues = warlock.RegisterSpell(core.SpellConfig{
		ActionID:     actionID,
		SpellSchool:  core.SpellSchoolShadow,
		ProcMask:     core.ProcMaskEmpty,
		ResourceType: stats.Mana,
		BaseCost:     baseCost,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				Cost: baseCost * (1 - 0.02*float64(warlock.Talents.Suppression)),
				GCD:  core.GCDDefault - core.TernaryDuration(warlock.Talents.AmplifyCurse, 1, 0)*500*time.Millisecond,
			},
		},

		ThreatMultiplier: 1 - 0.1*float64(warlock.Talents.ImprovedDrainSoul),
		FlatThreatBonus:  0, // TODO

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealOutcome(sim, target, spell.OutcomeMagicHit)
			if result.Landed() {
				warlock.CurseOfTonguesAura.Activate(sim)
			}
		},
	})
}

func (warlock *Warlock) registerCurseOfAgonySpell() {
	actionID := core.ActionID{SpellID: 47864}
	baseCost := 0.1 * warlock.BaseMana
	numberOfTicks := int32(12)
	totalBaseDmg := 1740.0
	if warlock.HasMajorGlyph(proto.WarlockMajorGlyph_GlyphOfCurseOfAgony) {
		numberOfTicks += 2
		totalBaseDmg += 2 * totalBaseDmg * 0.056 // Glyphed ticks
	}

	warlock.CurseOfAgony = warlock.RegisterSpell(core.SpellConfig{
		ActionID:     actionID,
		SpellSchool:  core.SpellSchoolShadow,
		ProcMask:     core.ProcMaskEmpty,
		ResourceType: stats.Mana,
		BaseCost:     baseCost,
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				Cost: baseCost * (1 - 0.02*float64(warlock.Talents.Suppression)),
				GCD:  core.GCDDefault - core.TernaryDuration(warlock.Talents.AmplifyCurse, 1, 0)*500*time.Millisecond,
			},
		},

		DamageMultiplierAdditive: 1 +
			0.03*float64(warlock.Talents.ShadowMastery) +
			0.01*float64(warlock.Talents.Contagion) +
			0.05*float64(warlock.Talents.ImprovedCurseOfAgony),
		ThreatMultiplier: 1 - 0.1*float64(warlock.Talents.ImprovedDrainSoul),
		FlatThreatBonus:  0, // TODO : curses flat threat on application

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealOutcome(sim, target, spell.OutcomeMagicHit)
			if result.Landed() {
				warlock.CurseOfDoomDot.Cancel(sim)
				warlock.CurseOfAgonyDot.Apply(sim)
			}
		},
	})
	warlock.CurseOfAgonyDot = core.NewDot(core.Dot{
		Spell: warlock.CurseOfAgony,
		Aura: warlock.CurrentTarget.RegisterAura(core.Aura{
			Label:    "CurseofAgony-" + strconv.Itoa(int(warlock.Index)),
			ActionID: actionID,
		}),
		NumberOfTicks: numberOfTicks,
		TickLength:    time.Second * 2,
		OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
			// Ignored: CoA ramp up effect
			dot.SnapshotBaseDamage = totalBaseDmg/float64(numberOfTicks) + 0.1*dot.Spell.SpellPower()
			dot.SnapshotAttackerMultiplier = dot.Spell.AttackerDamageMultiplier(dot.Spell.Unit.AttackTables[target.UnitIndex])
		},
		OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
			dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
		},
	})
}

func (warlock *Warlock) registerCurseOfDoomSpell() {
	actionID := core.ActionID{SpellID: 47867}
	baseCost := 0.15 * warlock.BaseMana

	warlock.CurseOfDoom = warlock.RegisterSpell(core.SpellConfig{
		ActionID:     actionID,
		SpellSchool:  core.SpellSchoolShadow,
		ProcMask:     core.ProcMaskEmpty,
		ResourceType: stats.Mana,
		BaseCost:     baseCost,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				Cost: baseCost * (1 - 0.02*float64(warlock.Talents.Suppression)),
				GCD:  core.GCDDefault - core.TernaryDuration(warlock.Talents.AmplifyCurse, 1, 0)*500*time.Millisecond,
			},
			CD: core.Cooldown{
				Timer:    warlock.NewTimer(),
				Duration: time.Minute,
			},
		},

		DamageMultiplierAdditive: 1 +
			0.03*float64(warlock.Talents.ShadowMastery),
		ThreatMultiplier: 1 - 0.1*float64(warlock.Talents.ImprovedDrainSoul),
		FlatThreatBonus:  0, // TODO

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealOutcome(sim, target, spell.OutcomeMagicHit)
			if result.Landed() {
				warlock.CurseOfAgonyDot.Cancel(sim)
				warlock.CurseOfDoomDot.Apply(sim)
			}
		},
	})

	warlock.CurseOfDoomDot = core.NewDot(core.Dot{
		Spell: warlock.CurseOfDoom,
		Aura: warlock.CurrentTarget.RegisterAura(core.Aura{
			Label:    "CurseofDoom-" + strconv.Itoa(int(warlock.Index)),
			ActionID: actionID,
		}),
		NumberOfTicks: 1,
		TickLength:    time.Minute,
		OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
			dot.SnapshotBaseDamage = 7300 + 2*dot.Spell.SpellPower()
			dot.SnapshotAttackerMultiplier = dot.Spell.AttackerDamageMultiplier(dot.Spell.Unit.AttackTables[target.UnitIndex])
		},
		OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
			dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
		},
	})
}
