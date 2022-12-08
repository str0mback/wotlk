package feral

import (
	"time"

	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/proto"
	"github.com/wowsims/wotlk/sim/druid"
)

func RegisterFeralDruid() {
	core.RegisterAgentFactory(
		proto.Player_FeralDruid{},
		proto.Spec_SpecFeralDruid,
		func(character core.Character, options *proto.Player) core.Agent {
			return NewFeralDruid(character, options)
		},
		func(player *proto.Player, spec interface{}) {
			playerSpec, ok := spec.(*proto.Player_FeralDruid)
			if !ok {
				panic("Invalid spec value for Feral Druid!")
			}
			player.Spec = playerSpec
		},
	)
}

func NewFeralDruid(character core.Character, options *proto.Player) *FeralDruid {
	feralOptions := options.GetFeralDruid()
	selfBuffs := druid.SelfBuffs{}

	selfBuffs.InnervateTarget = &proto.RaidTarget{TargetIndex: -1}
	if feralOptions.Options.InnervateTarget != nil {
		selfBuffs.InnervateTarget = feralOptions.Options.InnervateTarget
	}

	cat := &FeralDruid{
		Druid:   druid.New(character, druid.Cat, selfBuffs, feralOptions.Talents),
		latency: time.Duration(feralOptions.Options.LatencyMs) * time.Millisecond,
	}

	cat.AssumeBleedActive = feralOptions.Options.AssumeBleedActive
	cat.maxRipTicks = cat.MaxRipTicks()
	cat.prepopOoc = feralOptions.Options.PrepopOoc
	cat.RaidBuffTargets = int(core.MaxInt32(feralOptions.Rotation.RaidTargets, 1))
	cat.PrePopBerserk = feralOptions.Options.PrePopBerserk
	cat.setupRotation(feralOptions.Rotation)

	// Passive Cat Form threat reduction
	cat.PseudoStats.ThreatMultiplier *= 0.71

	cat.EnableEnergyBar(100.0, cat.OnEnergyGain)

	cat.EnableRageBar(core.RageBarOptions{RageMultiplier: 1, MHSwingSpeed: 2.5}, func(sim *core.Simulation) {})

	cat.EnableAutoAttacks(cat, core.AutoAttackOptions{
		// Base paw weapon.
		MainHand: core.Weapon{
			BaseDamageMin:              43,
			BaseDamageMax:              66,
			SwingSpeed:                 1.0,
			NormalizedSwingSpeed:       1.0,
			SwingDuration:              time.Second,
			CritMultiplier:             cat.MeleeCritMultiplier(druid.Cat),
			MeleeAttackRatingPerDamage: core.MeleeAttackRatingPerDamage,
		},
		AutoSwingMelee: true,
	})

	return cat
}

type FeralDruid struct {
	*druid.Druid

	Rotation FeralDruidRotation

	prepopOoc      bool
	missChance     float64
	readyToShift   bool
	readyToGift    bool
	waitingForTick bool
	latency        time.Duration
	maxRipTicks    int32
	berserkUsed    bool
}

func (cat *FeralDruid) GetDruid() *druid.Druid {
	return cat.Druid
}

func (cat *FeralDruid) MissChance() float64 {
	at := cat.AttackTables[cat.CurrentTarget.UnitIndex]
	miss := at.BaseMissChance - cat.Shred.PhysicalHitChance(cat.CurrentTarget)
	dodge := at.BaseDodgeChance - cat.Shred.ExpertisePercentage() - cat.CurrentTarget.PseudoStats.DodgeReduction
	return miss + dodge
}

func (cat *FeralDruid) Initialize() {
	cat.Druid.Initialize()
	cat.RegisterFeralSpells(0)
	cat.DelayDPSCooldownsForArmorDebuffs(time.Second * 10)
}

func (cat *FeralDruid) Reset(sim *core.Simulation) {
	cat.Druid.Reset(sim)
	cat.Druid.ClearForm(sim)
	cat.CatFormAura.Activate(sim)
	cat.readyToShift = false
	cat.waitingForTick = false
	cat.berserkUsed = false

	if cat.prepopOoc && cat.Talents.OmenOfClarity {
		cat.ClearcastingAura.Activate(sim)
	}
	if cat.PrePopBerserk && cat.Talents.Berserk {
		cat.Berserk.CD.UsePrePull(sim, time.Second)
	}
}
