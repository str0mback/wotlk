package deathknight

import (
	"time"

	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/stats"
)

type Presence uint8

const (
	BloodPresence Presence = 1 << iota
	FrostPresence
	UnholyPresence
	UnsetPresence
)

const presenceEffectCategory = "Presence"

func (dk *Deathknight) PresenceMatches(other Presence) bool {
	return (dk.Presence & other) != 0
}

func (dk *Deathknight) ChangePresence(sim *core.Simulation, newPresence Presence) {
	if dk.PresenceMatches(newPresence) {
		return
	}

	dk.Presence = newPresence
	if dk.PresenceMatches(BloodPresence) {
		dk.BloodPresenceAura.Activate(sim)
		dk.FrostPresenceAura.Deactivate(sim)
		dk.UnholyPresenceAura.Deactivate(sim)
	} else if dk.PresenceMatches(FrostPresence) {
		dk.FrostPresenceAura.Activate(sim)
		dk.BloodPresenceAura.Deactivate(sim)
		dk.UnholyPresenceAura.Deactivate(sim)
	} else if dk.PresenceMatches(UnholyPresence) {
		dk.UnholyPresenceAura.Activate(sim)
		dk.BloodPresenceAura.Deactivate(sim)
		dk.FrostPresenceAura.Deactivate(sim)
	}
}

func (dk *Deathknight) registerBloodPresenceAura(timer *core.Timer) {
	threatMult := 0.8
	threatMultSubversion := 1.0 - dk.subversionThreatBonus()
	//TODO: Include hps bonus
	damageBonusCoeff := 0.15
	staminaMult := 1.0 + 0.04*float64(dk.Talents.ImprovedFrostPresence)
	damageTakenMult := 1.0 - 0.01*float64(dk.Talents.ImprovedFrostPresence)

	baseCost := float64(core.NewRuneCost(0, 1, 0, 0, 0))
	dk.BloodPresence = dk.RegisterSpell(nil, core.SpellConfig{
		ActionID:     core.ActionID{SpellID: 50689},
		ResourceType: stats.RunicPower,
		BaseCost:     baseCost,
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				Cost: baseCost,
			},
			CD: core.Cooldown{
				Timer:    timer,
				Duration: time.Second,
			},
		},
		ApplyEffects: func(sim *core.Simulation, unit *core.Unit, spell *core.Spell) {
			dk.ChangePresence(sim, BloodPresence)
		},
	}, func(sim *core.Simulation) bool {
		return dk.CastCostPossible(sim, 0.0, 1, 0, 0) && dk.BloodPresence.IsReady(sim)
	}, nil)

	// TODO: Probably improve this
	isDps := dk.Talents.HowlingBlast || dk.Talents.SummonGargoyle

	actionID := core.ActionID{SpellID: 50689}
	healthMetrics := dk.NewHealthMetrics(actionID)
	statDep := dk.NewDynamicMultiplyStat(stats.Stamina, staminaMult)

	aura := core.Aura{
		Label:    "Blood Presence",
		ActionID: actionID,
		Duration: core.NeverExpires,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.PseudoStats.ThreatMultiplier *= threatMult
			aura.Unit.PseudoStats.ThreatMultiplier *= threatMultSubversion
			aura.Unit.PseudoStats.DamageTakenMultiplier *= damageTakenMult

			dk.ModifyDamageModifier(damageBonusCoeff)
			aura.Unit.EnableDynamicStatDep(sim, statDep)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.PseudoStats.ThreatMultiplier /= threatMult
			aura.Unit.PseudoStats.ThreatMultiplier /= threatMultSubversion
			aura.Unit.PseudoStats.DamageTakenMultiplier /= damageTakenMult

			dk.ModifyDamageModifier(-damageBonusCoeff)
			aura.Unit.DisableDynamicStatDep(sim, statDep)
		},
	}

	if !isDps {
		aura.OnSpellHitDealt = func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if result.Damage > 0 {
				healthGain := (0.04 * result.Damage) * (1.0 + core.TernaryFloat64(dk.VampiricBloodAura.IsActive(), 0.35, 0.0))
				dk.GainHealth(sim, healthGain, healthMetrics)
			}
		}
	}

	dk.BloodPresenceAura = dk.GetOrRegisterAura(aura)
	dk.BloodPresenceAura.NewExclusiveEffect(presenceEffectCategory, true, core.ExclusiveEffect{})
}

func (dk *Deathknight) registerFrostPresenceAura(timer *core.Timer) {

	baseCost := float64(core.NewRuneCost(0, 0, 1, 0, 0))
	dk.FrostPresence = dk.RegisterSpell(nil, core.SpellConfig{
		ActionID:     core.ActionID{SpellID: 48263},
		ResourceType: stats.RunicPower,
		BaseCost:     baseCost,
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				Cost: baseCost,
			},
			CD: core.Cooldown{
				Timer:    timer,
				Duration: time.Second,
			},
		},
		ApplyEffects: func(sim *core.Simulation, unit *core.Unit, spell *core.Spell) {
			dk.ChangePresence(sim, FrostPresence)
		},
	}, func(sim *core.Simulation) bool {
		return dk.CastCostPossible(sim, 0.0, 0, 1, 0) && dk.FrostPresence.IsReady(sim)
	}, nil)

	threatMult := 2.0735
	stamDep := dk.NewDynamicMultiplyStat(stats.Stamina, 1.08)
	armorDep := dk.NewDynamicMultiplyStat(stats.Armor, 1.6)
	dk.FrostPresenceAura = dk.GetOrRegisterAura(core.Aura{
		Label:    "Frost Presence",
		ActionID: core.ActionID{SpellID: 48263},
		Duration: core.NeverExpires,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.PseudoStats.ThreatMultiplier *= threatMult

			aura.Unit.EnableDynamicStatDep(sim, stamDep)
			aura.Unit.EnableDynamicStatDep(sim, armorDep)

			dk.IcyTouch.ThreatMultiplier *= 7
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.PseudoStats.ThreatMultiplier /= threatMult

			aura.Unit.DisableDynamicStatDep(sim, stamDep)
			aura.Unit.DisableDynamicStatDep(sim, armorDep)

			dk.IcyTouch.ThreatMultiplier /= 7
		},
	})
	dk.FrostPresenceAura.NewExclusiveEffect(presenceEffectCategory, true, core.ExclusiveEffect{})
}

func (dk *Deathknight) registerUnholyPresenceAura(timer *core.Timer) {
	threatMultSubversion := 1.0 - dk.subversionThreatBonus()

	baseCost := float64(core.NewRuneCost(0, 0, 0, 1, 0))
	dk.UnholyPresence = dk.RegisterSpell(nil, core.SpellConfig{
		ActionID:     core.ActionID{SpellID: 48265},
		ResourceType: stats.RunicPower,
		BaseCost:     baseCost,
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				Cost: baseCost,
			},
			CD: core.Cooldown{
				Timer:    timer,
				Duration: time.Second,
			},
		},
		ApplyEffects: func(sim *core.Simulation, unit *core.Unit, spell *core.Spell) {
			dk.ChangePresence(sim, UnholyPresence)
		},
	}, func(sim *core.Simulation) bool {
		return dk.CastCostPossible(sim, 0.0, 0, 0, 1) && dk.UnholyPresence.IsReady(sim)
	}, nil)

	runeCd := 10 * time.Second
	impUp := 500 * time.Millisecond * time.Duration(dk.Talents.ImprovedUnholyPresence)
	stamDep := dk.NewDynamicMultiplyStat(stats.Stamina, 1.0+0.04*float64(dk.Talents.ImprovedFrostPresence))
	dk.UnholyPresenceAura = dk.GetOrRegisterAura(core.Aura{
		Label:    "Unholy Presence",
		ActionID: core.ActionID{SpellID: 48265},
		Duration: core.NeverExpires,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			if dk.Talents.ImprovedUnholyPresence > 0 {
				aura.Unit.SetRuneCd(runeCd - impUp)
			}
			aura.Unit.PseudoStats.ThreatMultiplier *= threatMultSubversion
			aura.Unit.EnableDynamicStatDep(sim, stamDep)
			dk.MultiplyMeleeSpeed(sim, 1.15)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			if dk.Talents.ImprovedUnholyPresence > 0 {
				aura.Unit.SetRuneCd(runeCd)
			}
			aura.Unit.PseudoStats.ThreatMultiplier /= threatMultSubversion
			aura.Unit.DisableDynamicStatDep(sim, stamDep)
			dk.MultiplyMeleeSpeed(sim, 1/1.15)
		},
	})
	dk.UnholyPresenceAura.NewExclusiveEffect(presenceEffectCategory, true, core.ExclusiveEffect{})
}

func (dk *Deathknight) GetModifiedGCD() time.Duration {
	if dk.UnholyPresenceAura.IsActive() {
		return time.Second
	} else {
		return core.GCDDefault
	}
}

func (dk *Deathknight) registerPresences() {
	presenceTimer := dk.NewTimer()
	dk.registerBloodPresenceAura(presenceTimer)
	dk.registerUnholyPresenceAura(presenceTimer)
	dk.registerFrostPresenceAura(presenceTimer)
}
