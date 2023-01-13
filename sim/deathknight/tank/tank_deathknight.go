package tank

import (
	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/proto"
	"github.com/wowsims/wotlk/sim/deathknight"
)

func RegisterTankDeathknight() {
	core.RegisterAgentFactory(
		proto.Player_TankDeathknight{},
		proto.Spec_SpecTankDeathknight,
		func(character core.Character, options *proto.Player) core.Agent {
			return NewTankDeathknight(character, options)
		},
		func(player *proto.Player, spec interface{}) {
			playerSpec, ok := spec.(*proto.Player_TankDeathknight)
			if !ok {
				panic("Invalid spec value for Tank Deathknight!")
			}
			player.Spec = playerSpec
		},
	)
}

type TankDeathknight struct {
	*deathknight.Deathknight

	switchIT   bool
	BloodSpell *deathknight.RuneSpell

	Rotation *proto.TankDeathknight_Rotation
}

func NewTankDeathknight(character core.Character, options *proto.Player) *TankDeathknight {
	dkOptions := options.GetTankDeathknight()

	tankDk := &TankDeathknight{
		Deathknight: deathknight.NewDeathknight(character, deathknight.DeathknightInputs{
			IsDps:              false,
			StartingRunicPower: dkOptions.Options.StartingRunicPower,
		}, options.TalentsString),
		Rotation: dkOptions.Rotation,
	}

	tankDk.Inputs.UnholyFrenzyTarget = dkOptions.Options.UnholyFrenzyTarget

	tankDk.EnableAutoAttacks(tankDk, core.AutoAttackOptions{
		MainHand:       tankDk.WeaponFromMainHand(tankDk.DefaultMeleeCritMultiplier()),
		OffHand:        tankDk.WeaponFromOffHand(tankDk.DefaultMeleeCritMultiplier()),
		AutoSwingMelee: true,
		ReplaceMHSwing: func(sim *core.Simulation, mhSwingSpell *core.Spell) *core.Spell {
			if tankDk.RuneStrike.CanCast(sim) {
				return tankDk.RuneStrike.Spell
			} else {
				return nil
			}
		},
	})

	return tankDk
}

func (dk *TankDeathknight) GetDeathknight() *deathknight.Deathknight {
	return dk.Deathknight
}

func (dk *TankDeathknight) Initialize() {
	dk.Deathknight.Initialize()
}

func (dk *TankDeathknight) SetupRotations() {
	dk.RotationSequence.Clear()

	if dk.Rotation.Opener == proto.TankDeathknight_Rotation_Regular {
		dk.setupTankRegularERWOpener()
	} else if dk.Rotation.Opener == proto.TankDeathknight_Rotation_Threat {
		dk.setupTankThreatERWOpener()
	}

	if dk.Rotation.OptimizationSetting == proto.TankDeathknight_Rotation_Hps {
		dk.RotationSequence.NewAction(dk.TankRA_Hps)
	} else if dk.Rotation.OptimizationSetting == proto.TankDeathknight_Rotation_Tps {
		dk.RotationSequence.NewAction(dk.TankRA_Tps)
	} else if dk.Rotation.OptimizationSetting == proto.TankDeathknight_Rotation_Dps {
		dk.RotationSequence.NewAction(dk.TankRA_Hps)
	}

	if dk.Rotation.BloodSpell == proto.TankDeathknight_Rotation_BloodStrike {
		dk.BloodSpell = dk.BloodStrike
	} else if dk.Rotation.BloodSpell == proto.TankDeathknight_Rotation_BloodBoil {
		dk.BloodSpell = dk.BloodBoil
	} else if dk.Rotation.BloodSpell == proto.TankDeathknight_Rotation_HeartStrike {
		if dk.HeartStrike != nil {
			dk.BloodSpell = dk.HeartStrike
		} else {
			dk.BloodSpell = dk.BloodStrike
		}
	}
}

func (dk *TankDeathknight) Reset(sim *core.Simulation) {
	dk.Deathknight.Reset(sim)

	dk.switchIT = false

	dk.Presence = deathknight.UnsetPresence
	if dk.Rotation.Presence == proto.TankDeathknight_Rotation_Blood {
		dk.ChangePresence(sim, deathknight.BloodPresence)
	} else if dk.Rotation.Presence == proto.TankDeathknight_Rotation_Frost {
		dk.ChangePresence(sim, deathknight.FrostPresence)
	} else if dk.Rotation.Presence == proto.TankDeathknight_Rotation_Unholy {
		dk.ChangePresence(sim, deathknight.UnholyPresence)
	}

	dk.SetupRotations()
}
