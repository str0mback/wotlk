package dps

import (
	"github.com/wowsims/wotlk/sim/common"
	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/proto"
	"github.com/wowsims/wotlk/sim/deathknight"
)

func RegisterDpsDeathknight() {
	core.RegisterAgentFactory(
		proto.Player_Deathknight{},
		proto.Spec_SpecDeathknight,
		func(character core.Character, options *proto.Player) core.Agent {
			return NewDpsDeathknight(character, options)
		},
		func(player *proto.Player, spec interface{}) {
			playerSpec, ok := spec.(*proto.Player_Deathknight)
			if !ok {
				panic("Invalid spec value for Deathknight!")
			}
			player.Spec = playerSpec
		},
	)
}

type DpsDeathknight struct {
	*deathknight.Deathknight

	sr SharedRotation
	fr FrostRotation
	ur UnholyRotation

	CustomRotation *common.CustomRotation

	Rotation *proto.Deathknight_Rotation
}

func NewDpsDeathknight(character core.Character, player *proto.Player) *DpsDeathknight {
	dk := player.GetDeathknight()

	dpsDk := &DpsDeathknight{
		Deathknight: deathknight.NewDeathknight(character, dk.Talents, deathknight.DeathknightInputs{
			StartingRunicPower:  dk.Options.StartingRunicPower,
			PrecastGhoulFrenzy:  dk.Options.PrecastGhoulFrenzy,
			PrecastHornOfWinter: dk.Options.PrecastHornOfWinter,
			PetUptime:           dk.Options.PetUptime,
			IsDps:               true,

			RefreshHornOfWinter: dk.Rotation.RefreshHornOfWinter,
			ArmyOfTheDeadType:   dk.Rotation.ArmyOfTheDead,
			StartingPresence:    dk.Rotation.StartingPresence,
			UseAMS:              dk.Rotation.UseAms,
			AvgAMSSuccessRate:   dk.Rotation.AvgAmsSuccessRate,
			AvgAMSHit:           dk.Rotation.AvgAmsHit,
		}),
		Rotation: dk.Rotation,
	}

	dpsDk.Inputs.UnholyFrenzyTarget = &proto.RaidTarget{TargetIndex: -1}
	if dk.Options.UnholyFrenzyTarget != nil {
		dpsDk.Inputs.UnholyFrenzyTarget = dk.Options.UnholyFrenzyTarget
	}

	dpsDk.EnableAutoAttacks(dpsDk, core.AutoAttackOptions{
		MainHand:       dpsDk.WeaponFromMainHand(dpsDk.DefaultMeleeCritMultiplier()),
		OffHand:        dpsDk.WeaponFromOffHand(dpsDk.DefaultMeleeCritMultiplier()),
		AutoSwingMelee: true,
		ReplaceMHSwing: func(sim *core.Simulation, mhSwingSpell *core.Spell) *core.Spell {
			if dpsDk.RuneStrike.CanCast(sim) {
				return dpsDk.RuneStrike.Spell
			} else {
				return nil
			}
		},
	})

	dpsDk.sr.dk = dpsDk
	dpsDk.ur.dk = dpsDk

	return dpsDk
}

func (dk *DpsDeathknight) FrostPointsInBlood() int32 {
	return dk.Talents.Butchery + dk.Talents.Subversion + dk.Talents.BladeBarrier + dk.Talents.DarkConviction
}

func (dk *DpsDeathknight) FrostPointsInUnholy() int32 {
	return dk.Talents.ViciousStrikes + dk.Talents.Virulence + dk.Talents.Epidemic + dk.Talents.RavenousDead + dk.Talents.Necrosis + dk.Talents.BloodCakedBlade
}

func (dk *DpsDeathknight) SetupRotations() {
	if dk.Rotation.AutoRotation {
		bl, fr, uh := deathknight.PointsInTalents(dk.Talents)

		if uh > fr && uh > bl {
			// Unholy
			dk.Rotation.BtGhoulFrenzy = false
			dk.Rotation.UseEmpowerRuneWeapon = true
			dk.Rotation.HoldErwArmy = true
			dk.Rotation.UseGargoyle = true
			dk.Rotation.ArmyOfTheDead = proto.Deathknight_Rotation_AsMajorCd
			dk.Rotation.BloodTap = proto.Deathknight_Rotation_GhoulFrenzy
			dk.Rotation.FirstDisease = proto.Deathknight_Rotation_FrostFever
			dk.Rotation.StartingPresence = proto.Deathknight_Rotation_Unholy
			dk.Rotation.BlPresence = proto.Deathknight_Rotation_Blood
			dk.Rotation.Presence = proto.Deathknight_Rotation_Blood

			mh := dk.GetMHWeapon()
			oh := dk.GetOHWeapon()

			if mh != nil && oh != nil {
				// DW
				dk.Rotation.BloodRuneFiller = proto.Deathknight_Rotation_BloodBoil
				dk.Rotation.UseDeathAndDecay = true
			} else {
				// 2h
				if dk.Env.GetNumTargets() > 1 {
					dk.Rotation.BloodRuneFiller = proto.Deathknight_Rotation_BloodBoil
					dk.Rotation.UseDeathAndDecay = true
				} else {
					dk.Rotation.BloodRuneFiller = proto.Deathknight_Rotation_BloodStrike
					dk.Rotation.UseDeathAndDecay = false
				}
			}
			// Always use DnD if you have the glyph.
			if dk.HasMajorGlyph(proto.DeathknightMajorGlyph_GlyphOfDeathAndDecay) {
				dk.Rotation.UseDeathAndDecay = true
			}
		} else if fr > uh && fr > bl {
			// Frost rotations here.
		} else if bl > fr && bl > uh {
			// Blood rotations here.
		} else {
			// some weird spec where two trees are equal...
		}
	}
	dk.ur.ffFirst = dk.Rotation.FirstDisease == proto.Deathknight_Rotation_FrostFever
	dk.ur.hasGod = dk.HasMajorGlyph(proto.DeathknightMajorGlyph_GlyphOfDisease)

	dk.RotationSequence.Clear()

	dk.Inputs.FuStrike = deathknight.FuStrike_Obliterate

	dk.CustomRotation = dk.makeCustomRotation()
	if dk.CustomRotation == nil || dk.Rotation.FrostRotationType == proto.Deathknight_Rotation_SingleTarget {
		dk.Rotation.FrostRotationType = proto.Deathknight_Rotation_SingleTarget
		if (dk.Talents.BloodOfTheNorth == 3) && (dk.Talents.Epidemic == 0) {
			if dk.Rotation.UseEmpowerRuneWeapon {
				if dk.Rotation.DesyncRotation {
					dk.setupFrostSubBloodDesyncERWOpener()
				} else {
					dk.setupFrostSubBloodERWOpener()
				}
			} else {
				dk.setupFrostSubBloodNoERWOpener()
			}
		} else if (dk.Talents.BloodOfTheNorth == 3) && (dk.Talents.Epidemic == 2) {
			dk.Rotation.FrostRotationType = proto.Deathknight_Rotation_SingleTarget
			if dk.Rotation.UseEmpowerRuneWeapon {
				dk.setupFrostSubUnholyERWOpener()
			} else {
				panic("you can't unh sub without ERW in the opener...yet")
				dk.setupFrostSubUnholyERWOpener()
			}
		} else if dk.Talents.SummonGargoyle {
			dk.setupUnholyRotations()
		} else if dk.Talents.DancingRuneWeapon {
			dk.setupBloodRotations()
		}
	} else {
		dk.setupCustomRotations()
	}
}

func (dk *DpsDeathknight) GetDeathknight() *deathknight.Deathknight {
	return dk.Deathknight
}

func (dk *DpsDeathknight) Initialize() {
	dk.Deathknight.Initialize()
	dk.ur.gargoyleSnapshot = core.NewSnapshotManager(dk.GetCharacter())
	dk.setupProcTrackers()
	dk.fr.Initialize(dk)
}

func (dk *DpsDeathknight) setupProcTrackers() {
	snapshotManager := dk.ur.gargoyleSnapshot

	snapshotManager.AddProc(40211, "Potion of Speed", true)
	snapshotManager.AddProc(54999, "Hyperspeed Acceleration", true)
	snapshotManager.AddProc(26297, "Berserking (Troll)", true)
	snapshotManager.AddProc(33697, "Blood Fury", true)

	snapshotManager.AddProc(53344, "Rune Of The Fallen Crusader Proc", false)
	snapshotManager.AddProc(55379, "Thundering Skyflare Diamond Proc", false)
	snapshotManager.AddProc(59620, "Berserking MH Proc", false)
	snapshotManager.AddProc(59620, "Berserking OH Proc", false)
	snapshotManager.AddProc(59626, "Black Magic Proc", false)

	snapshotManager.AddProc(42987, "DMC Greatness Strength Proc", false)

	snapshotManager.AddProc(47115, "Deaths Verdict Strength Proc", false)
	snapshotManager.AddProc(47131, "Deaths Verdict H Strength Proc", false)
	snapshotManager.AddProc(47303, "Deaths Choice Strength Proc", false)
	snapshotManager.AddProc(47464, "Deaths Choice H Strength Proc", false)

	snapshotManager.AddProc(71484, "Deathbringer's Will Strength Proc", false)
	snapshotManager.AddProc(71492, "Deathbringer's Will Haste Proc", false)
	snapshotManager.AddProc(71561, "Deathbringer's Will H Strength Proc", false)
	snapshotManager.AddProc(71560, "Deathbringer's Will H Haste Proc", false)

	snapshotManager.AddProc(37390, "Meteorite Whetstone Proc", false)
	snapshotManager.AddProc(39229, "Embrace of the Spider Proc", false)
	snapshotManager.AddProc(40684, "Mirror of Truth Proc", false)
	snapshotManager.AddProc(40767, "Sonic Booster Proc", false)
	snapshotManager.AddProc(43573, "Tears of Bitter Anguish Proc", false)
	snapshotManager.AddProc(44308, "Signet of Edward the Odd Proc", false)
	snapshotManager.AddProc(44914, "Anvil of Titans Proc", false)
	snapshotManager.AddProc(45286, "Pyrite Infuser Proc", false)
	snapshotManager.AddProc(45522, "Blood of the Old God Proc", false)
	snapshotManager.AddProc(45609, "Comet's Trail Proc", false)
	snapshotManager.AddProc(45866, "Elemental Focus Stone Proc", false)
	snapshotManager.AddProc(47214, "Banner of Victory Proc", false)
	snapshotManager.AddProc(49074, "Coren's Chromium Coaster Proc", false)
	snapshotManager.AddProc(50342, "Whispering Fanged Skull Proc", false)
	snapshotManager.AddProc(50343, "Whispering Fanged Skull H Proc", false)
	snapshotManager.AddProc(50401, "Ashen Band of Unmatched Vengeance Proc", false)
	snapshotManager.AddProc(50402, "Ashen Band of Endless Vengeance Proc", false)
	snapshotManager.AddProc(52571, "Ashen Band of Unmatched Might Proc", false)
	snapshotManager.AddProc(52572, "Ashen Band of Endless Might Proc", false)
	snapshotManager.AddProc(54569, "Sharpened Twilight Scale Proc", false)
	snapshotManager.AddProc(54590, "Sharpened Twilight Scale H Proc", false)
}

func (dk *DpsDeathknight) setupGargoyleCooldowns() {
	dk.ur.gargoyleSnapshot.ClearMajorCooldowns()

	// hyperspeed accelerators
	dk.gargoyleCooldownSync(core.ActionID{SpellID: 54758}, false)

	// berserking (troll)
	dk.gargoyleCooldownSync(core.ActionID{SpellID: 26297}, false)

	// blood fury (orc)
	dk.gargoyleCooldownSync(core.ActionID{SpellID: 33697}, false)

	// potion of speed
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 40211}, true)

	// active ap trinkets
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 35937}, false)
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 36871}, false)
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 37166}, false)
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 37556}, false)
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 37557}, false)
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 38080}, false)
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 38081}, false)
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 38761}, false)
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 39257}, false)
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 45263}, false)
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 46086}, false)
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 47734}, false)

	// active haste trinkets
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 36972}, false)
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 37558}, false)
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 37560}, false)
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 37562}, false)
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 38070}, false)
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 38258}, false)
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 38259}, false)
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 38764}, false)
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 40531}, false)
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 43836}, false)
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 45466}, false)
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 46088}, false)
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 48722}, false)
	dk.gargoyleCooldownSync(core.ActionID{ItemID: 50260}, false)
}

func (dk *DpsDeathknight) gargoyleCooldownSync(actionID core.ActionID, isPotion bool) {
	if majorCd := dk.Character.GetMajorCooldown(actionID); majorCd != nil {

		majorCd.ShouldActivate = func(sim *core.Simulation, character *core.Character) bool {
			return dk.ur.activatingGargoyle || (dk.SummonGargoyle.CD.TimeToReady(sim) > majorCd.Spell.CD.Duration && !isPotion) || dk.SummonGargoyle.CD.ReadyAt() > dk.Env.Encounter.Duration
		}

		dk.ur.gargoyleSnapshot.AddMajorCooldown(majorCd)
	}
}

func (dk *DpsDeathknight) Reset(sim *core.Simulation) {
	dk.Deathknight.Reset(sim)

	dk.sr.Reset(sim)
	dk.fr.Reset(sim)
	dk.ur.Reset(sim)

	dk.SetupRotations()

	dk.Presence = deathknight.UnsetPresence

	b, f, u := deathknight.PointsInTalents(dk.Talents)

	if f > u && f > b {
		if dk.Rotation.Presence == proto.Deathknight_Rotation_Blood {
			dk.ChangePresence(sim, deathknight.BloodPresence)
		} else if dk.Rotation.Presence == proto.Deathknight_Rotation_Frost {
			dk.ChangePresence(sim, deathknight.FrostPresence)
		} else if dk.Rotation.Presence == proto.Deathknight_Rotation_Unholy {
			dk.ChangePresence(sim, deathknight.UnholyPresence)
		}
	}

	if u > f && u > b {
		if dk.Rotation.StartingPresence == proto.Deathknight_Rotation_Unholy {
			dk.ChangePresence(sim, deathknight.UnholyPresence)
		} else if dk.Talents.SummonGargoyle {
			dk.ChangePresence(sim, deathknight.BloodPresence)
		}
	}
}
