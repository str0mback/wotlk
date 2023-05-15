package tank

import (
	"testing"

	_ "github.com/wowsims/wotlk/sim/common" // imported to get item effects included.
	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/proto"
)

func init() {
	RegisterTankDeathknight()
}

func TestBloodTank(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator(core.CharacterSuiteConfig{
		Class:      proto.Class_ClassDeathknight,
		Race:       proto.Race_RaceOrc,
		OtherRaces: []proto.Race{proto.Race_RaceHuman},

		GearSet:     core.GearSetCombo{Label: "Blood Tank P1", GearSet: BloodP1Gear},
		Talents:     BloodTankTalents,
		Glyphs:      Glyphs,
		Consumes:    FullConsumes,
		SpecOptions: core.SpecOptionsCombo{Label: "Basic", SpecOptions: PlayerOptionsBloodTank},

		IsTank:          true,
		InFrontOfTarget: true,

		ItemFilter: core.ItemFilter{
			ArmorType: proto.ArmorType_ArmorTypePlate,

			WeaponTypes: []proto.WeaponType{
				proto.WeaponType_WeaponTypeAxe,
				proto.WeaponType_WeaponTypeSword,
				proto.WeaponType_WeaponTypeMace,
			},
		},
	}))
}

var BloodTankTalents = "005510153330330220102013-3050505100023101-002"
var Glyphs = &proto.Glyphs{
	Major1: int32(proto.DeathknightMajorGlyph_GlyphOfDarkCommand),
	Major2: int32(proto.DeathknightMajorGlyph_GlyphOfObliterate),
	Major3: int32(proto.DeathknightMajorGlyph_GlyphOfVampiricBlood),
}

var PlayerOptionsBloodTank = &proto.Player_TankDeathknight{
	TankDeathknight: &proto.TankDeathknight{
		Options:  deathKnightOptions,
		Rotation: bloodTankRotation,
	},
}

var bloodTankRotation = &proto.TankDeathknight_Rotation{
	Presence:            proto.TankDeathknight_Rotation_Frost,
	Opener:              proto.TankDeathknight_Rotation_Threat,
	OptimizationSetting: proto.TankDeathknight_Rotation_Hps,
	BloodSpell:          proto.TankDeathknight_Rotation_BloodStrike,
	BloodTapPrio:        proto.TankDeathknight_Rotation_Defensive,
}

var deathKnightOptions = &proto.TankDeathknight_Options{
	StartingRunicPower: 0,
}

var FullRaidBuffs = &proto.RaidBuffs{
	GiftOfTheWild:         proto.TristateEffect_TristateEffectImproved,
	PowerWordFortitude:    proto.TristateEffect_TristateEffectImproved,
	AbominationsMight:     true,
	SwiftRetribution:      true,
	Bloodlust:             true,
	StrengthOfEarthTotem:  proto.TristateEffect_TristateEffectImproved,
	LeaderOfThePack:       proto.TristateEffect_TristateEffectImproved,
	SanctifiedRetribution: true,
	DevotionAura:          proto.TristateEffect_TristateEffectImproved,
	RetributionAura:       true,
	IcyTalons:             true,
}
var FullPartyBuffs = &proto.PartyBuffs{
	HeroicPresence: true,
}
var FullIndividualBuffs = &proto.IndividualBuffs{
	BlessingOfKings:     true,
	BlessingOfMight:     proto.TristateEffect_TristateEffectImproved,
	BlessingOfSanctuary: true,
}

var FullConsumes = &proto.Consumes{
	Flask:         proto.Flask_FlaskOfStoneblood,
	DefaultPotion: proto.Potions_IndestructiblePotion,
	PrepopPotion:  proto.Potions_IndestructiblePotion,
	Food:          proto.Food_FoodDragonfinFilet,
}

var FullDebuffs = &proto.Debuffs{
	SunderArmor:        true,
	Mangle:             true,
	DemoralizingShout:  proto.TristateEffect_TristateEffectImproved,
	JudgementOfLight:   true,
	FaerieFire:         proto.TristateEffect_TristateEffectRegular,
	Misery:             true,
	FrostFever:         proto.TristateEffect_TristateEffectImproved,
	BloodFrenzy:        true,
	EbonPlaguebringer:  true,
	HeartOfTheCrusader: true,
}

var BloodP1Gear = core.EquipmentSpecFromJsonString(`{"items": [
	{
		"id": 40565,
		"enchant": 3878,
		"gems": [
			41380,
			36767
		]
	},
	{
		"id": 40387
	},
	{
		"id": 39704,
		"enchant": 3852,
		"gems": [
			40008
		]
	},
	{
		"id": 40252,
		"enchant": 3605
	},
	{
		"id": 40559,
		"gems": [
			40008,
			40022
		]
	},
	{
		"id": 40306,
		"enchant": 3850,
		"gems": [
			40008,
			0
		]
	},
	{
		"id": 40563,
		"enchant": 3860,
		"gems": [
			40008,
			0
		]
	},
	{
		"id": 39759,
		"gems": [
			40008,
			40008
		]
	},
	{
		"id": 40567,
		"enchant": 3822,
		"gems": [
			40008,
			40008
		]
	},
	{
		"id": 40297,
		"enchant": 3232
	},
	{
		"id": 40718
	},
	{
		"id": 40107
	},
	{
		"id": 44063,
		"gems": [
			36767,
			36767
		]
	},
	{
		"id": 42341,
		"gems": [
			40008,
			40008
		]
	},
	{
		"id": 40406,
		"enchant": 3847
	},
	{},
	{
		"id": 40207
	}
]}`)
