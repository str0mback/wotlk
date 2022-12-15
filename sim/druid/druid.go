package druid

import (
	"time"

	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/proto"
	"github.com/wowsims/wotlk/sim/core/stats"
)

const (
	SpellFlagNaturesGrace = core.SpellFlagAgentReserved1
)

type Druid struct {
	core.Character
	SelfBuffs
	Talents *proto.DruidTalents

	StartingForm DruidForm

	RebirthUsed       bool
	MaulRageThreshold float64
	RebirthTiming     float64
	BleedsActive      int
	AssumeBleedActive bool
	RaidBuffTargets   int
	PrePopBerserk     bool

	Barkskin             *core.Spell
	Berserk              *core.Spell
	DemoralizingRoar     *core.Spell
	Enrage               *core.Spell
	FaerieFire           *core.Spell
	FerociousBite        *core.Spell
	ForceOfNature        *core.Spell
	FrenziedRegeneration *core.Spell
	Hurricane            *core.Spell
	InsectSwarm          *core.Spell
	GiftOfTheWild        *core.Spell
	Lacerate             *core.Spell
	MangleBear           *core.Spell
	MangleCat            *core.Spell
	Maul                 *core.Spell
	Moonfire             *core.Spell
	Rebirth              *core.Spell
	Rake                 *core.Spell
	Rip                  *core.Spell
	SavageRoar           *core.Spell
	Shred                *core.Spell
	Starfire             *core.Spell
	Starfall             *core.Spell
	StarfallSplash       *core.Spell
	SurvivalInstincts    *core.Spell
	SwipeBear            *core.Spell
	SwipeCat             *core.Spell
	TigersFury           *core.Spell
	Typhoon              *core.Spell
	Wrath                *core.Spell

	CatForm  *core.Spell
	BearForm *core.Spell

	InsectSwarmDot    *core.Dot
	LacerateDot       *core.Dot
	MoonfireDot       *core.Dot
	RakeDot           *core.Dot
	RipDot            *core.Dot
	StarfallDot       *core.Dot
	StarfallDotSplash *core.Dot

	BarkskinAura             *core.Aura
	BearFormAura             *core.Aura
	BerserkAura              *core.Aura
	CatFormAura              *core.Aura
	ClearcastingAura         *core.Aura
	DemoralizingRoarAuras    []*core.Aura
	EarthAndMoonAura         *core.Aura
	EnrageAura               *core.Aura
	FaerieFireAura           *core.Aura
	FrenziedRegenerationAura *core.Aura
	MangleAura               *core.Aura
	MaulQueueAura            *core.Aura
	MoonkinT84PCAura         *core.Aura
	NaturesGraceProcAura     *core.Aura
	PredatoryInstinctsAura   *core.Aura
	SavageDefenseAura        *core.Aura
	SurvivalInstinctsAura    *core.Aura
	TigersFuryAura           *core.Aura
	SavageRoarAura           *core.Aura
	SolarEclipseProcAura     *core.Aura
	LunarEclipseProcAura     *core.Aura

	PrimalPrecisionRecoveryMetrics *core.ResourceMetrics
	SavageRoarDurationTable        [6]time.Duration

	LunarICD core.Cooldown
	SolarICD core.Cooldown
	Treant1  *TreantPet
	Treant2  *TreantPet
	Treant3  *TreantPet

	form         DruidForm
	disabledMCDs []*core.MajorCooldown
}

type SelfBuffs struct {
	InnervateTarget *proto.RaidTarget
}

func (druid *Druid) GetCharacter() *core.Character {
	return &druid.Character
}

func (druid *Druid) AddRaidBuffs(raidBuffs *proto.RaidBuffs) {
	raidBuffs.GiftOfTheWild = core.MaxTristate(raidBuffs.GiftOfTheWild, proto.TristateEffect_TristateEffectRegular)
	if druid.Talents.ImprovedMarkOfTheWild == 2 { // probably could work on actually calculating the fraction effect later if we care.
		raidBuffs.GiftOfTheWild = proto.TristateEffect_TristateEffectImproved
	}

	raidBuffs.Thorns = core.MaxTristate(raidBuffs.Thorns, proto.TristateEffect_TristateEffectRegular)
	if druid.Talents.Brambles == 3 {
		raidBuffs.Thorns = proto.TristateEffect_TristateEffectImproved
	}

	if druid.InForm(Moonkin) && druid.Talents.MoonkinForm {
		raidBuffs.MoonkinAura = core.MaxTristate(raidBuffs.MoonkinAura, proto.TristateEffect_TristateEffectRegular)
		if druid.Talents.ImprovedMoonkinForm > 0 {
			// For now, we assume Improved Moonkin Form is maxed-out
			raidBuffs.MoonkinAura = proto.TristateEffect_TristateEffectImproved
		}
	}
	if druid.InForm(Cat|Bear) && druid.Talents.LeaderOfThePack {
		raidBuffs.LeaderOfThePack = core.MaxTristate(raidBuffs.LeaderOfThePack, proto.TristateEffect_TristateEffectRegular)
		if druid.Talents.ImprovedLeaderOfThePack > 0 {
			raidBuffs.LeaderOfThePack = proto.TristateEffect_TristateEffectImproved
		}
	}
}

func (druid *Druid) IsScalableArmorSlot(itemType proto.ItemType) bool {
	switch itemType {
	case
		proto.ItemType_ItemTypeNeck,
		proto.ItemType_ItemTypeFinger,
		proto.ItemType_ItemTypeTrinket,
		proto.ItemType_ItemTypeWeapon:
		return false
	}
	return true
}

func (druid *Druid) ScaleBaseArmor(multiplier float64) float64 {
	addedArmor := 0.0

	for _, item := range druid.Equip {
		if druid.IsScalableArmorSlot(item.Type) {
			addedArmor += multiplier * (item.Stats[stats.Armor] - item.Stats[stats.BonusArmor])
		}
	}

	return addedArmor
}

func (druid *Druid) BalanceCritMultiplier() float64 {
	return druid.SpellCritMultiplier(1, 0.2*float64(druid.Talents.Vengeance))
}

func (druid *Druid) MeleeCritMultiplier(castedForm DruidForm) float64 {
	// Assumes that Predatory Instincts is a primary rather than secondary modifier for now, but this needs to confirmed!
	primaryModifier := 1.0
	if castedForm.Matches(Cat) {
		primaryModifier = []float64{1, 1.03, 1.07, 1.10}[druid.Talents.PredatoryInstincts]
	}
	return druid.Character.MeleeCritMultiplier(primaryModifier, 0)
}

func (druid *Druid) HasMajorGlyph(glyph proto.DruidMajorGlyph) bool {
	return druid.HasGlyph(int32(glyph))
}
func (druid *Druid) HasMinorGlyph(glyph proto.DruidMinorGlyph) bool {
	return druid.HasGlyph(int32(glyph))
}

func (druid *Druid) Initialize() {
	if druid.Talents.PrimalPrecision > 0 {
		druid.PrimalPrecisionRecoveryMetrics = druid.NewEnergyMetrics(core.ActionID{SpellID: 48410})
	}
	druid.registerFaerieFireSpell()
	druid.registerRebirthSpell()
	druid.registerInnervateCD()
}

func (druid *Druid) RegisterBalanceSpells() {
	druid.registerHurricaneSpell()
	druid.registerInsectSwarmSpell()
	druid.registerMoonfireSpell()
	druid.registerStarfireSpell()
	druid.registerWrathSpell()
	druid.registerStarfallSpell()
	druid.registerTyphoonSpell()
	druid.registerForceOfNatureCD()
}

func (druid *Druid) RegisterFeralSpells(maulRageThreshold float64) {
	druid.registerBarkskinCD()
	druid.registerBerserkCD()
	druid.registerCatFormSpell()
	druid.registerBearFormSpell()
	druid.registerDemoralizingRoarSpell()
	druid.registerEnrageSpell()
	druid.registerFerociousBiteSpell()
	druid.registerFrenziedRegenerationCD()
	druid.registerMangleBearSpell()
	druid.registerMangleCatSpell()
	druid.registerMaulSpell(maulRageThreshold)
	druid.registerLacerateSpell()
	druid.registerRakeSpell()
	druid.registerRipSpell()
	druid.registerSavageDefensePassive()
	druid.registerSavageRoarSpell()
	druid.registerShredSpell()
	druid.registerSurvivalInstinctsCD()
	druid.registerSwipeBearSpell()
	druid.registerSwipeCatSpell()
	druid.registerTigersFurySpell()
	druid.registerFakeGotw()
}

func (druid *Druid) Reset(_ *core.Simulation) {
	druid.BleedsActive = 0
	druid.form = druid.StartingForm
	druid.disabledMCDs = []*core.MajorCooldown{}
	druid.RebirthUsed = false
	druid.LunarICD.Timer.Reset()
	druid.SolarICD.Timer.Reset()
}

func New(char core.Character, form DruidForm, selfBuffs SelfBuffs, talents *proto.DruidTalents) *Druid {
	druid := &Druid{
		Character:    char,
		SelfBuffs:    selfBuffs,
		Talents:      talents,
		StartingForm: form,
		form:         form,
	}
	druid.EnableManaBar()

	druid.AddStatDependency(stats.Strength, stats.AttackPower, 2)
	// Druids get 0.012 crit per agi at level 80, roughly 1 per 83.33
	druid.AddStatDependency(stats.Agility, stats.MeleeCrit, (1.0/83.33)*core.CritRatingPerCritChance)
	// Druid get 0.0209 dodge per agi (before dr), roughly 1 per 47.16
	druid.AddStatDependency(stats.Agility, stats.Dodge, (1.0/47.16)*core.DodgeRatingPerDodgeChance)

	// Druids get extra melee haste
	druid.PseudoStats.MeleeHasteRatingPerHastePercent /= 1.3

	// Base dodge is unaffected by Diminishing Returns
	druid.PseudoStats.BaseDodge += 0.0559

	if druid.Talents.ForceOfNature {
		druid.Treant1 = druid.NewTreant()
		druid.Treant2 = druid.NewTreant()
		druid.Treant3 = druid.NewTreant()
	}

	return druid
}

func init() {
	core.BaseStats[core.BaseStatsKey{Race: proto.Race_RaceTauren, Class: proto.Class_ClassDruid}] = stats.Stats{
		stats.Health:      6892, // 8227 health shown on naked character (would include tauren bonus)
		stats.Strength:    94,
		stats.Agility:     78,
		stats.Stamina:     99,
		stats.Intellect:   139,
		stats.Spirit:      161,
		stats.Mana:        3496,                                // 5301 mana shown on naked character
		stats.SpellCrit:   1.85 * core.CritRatingPerCritChance, // Class-specific constant
		stats.MeleeCrit:   7.48 * core.CritRatingPerCritChance, // 8.41% chance to crit shown on naked character screen
		stats.AttackPower: -20,
	}
	core.BaseStats[core.BaseStatsKey{Race: proto.Race_RaceNightElf, Class: proto.Class_ClassDruid}] = stats.Stats{
		stats.Health:      7237, // 8217 health shown on naked character
		stats.Strength:    85,
		stats.Agility:     86,
		stats.Stamina:     98,
		stats.Intellect:   143,
		stats.Spirit:      159,
		stats.Mana:        3496,                                // 5361 mana shown on naked character
		stats.SpellCrit:   1.85 * core.CritRatingPerCritChance, // Class-specific constant
		stats.MeleeCrit:   7.48 * core.CritRatingPerCritChance, // 8.51% chance to crit shown on naked character screen
		stats.AttackPower: -20,
	}
}

// Agent is a generic way to access underlying druid on any of the agents (for example balance druid.)
type DruidAgent interface {
	GetDruid() *Druid
}
