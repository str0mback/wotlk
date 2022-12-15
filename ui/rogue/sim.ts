import { RaidBuffs, WeaponType } from '../core/proto/common.js';
import { PartyBuffs } from '../core/proto/common.js';
import { IndividualBuffs } from '../core/proto/common.js';
import { Debuffs } from '../core/proto/common.js';
import { ItemSlot } from '../core/proto/common.js';
import { Spec } from '../core/proto/common.js';
import { Stat, PseudoStat } from '../core/proto/common.js';
import { TristateEffect } from '../core/proto/common.js'
import { Player } from '../core/player.js';
import { Stats } from '../core/proto_utils/stats.js';
import { IndividualSimUI } from '../core/individual_sim_ui.js';

import {
	Rogue_Rotation_AssassinationPriority as AssassinationPriority,
	Rogue_Rotation_CombatPriority as CombatPriority,
	Rogue_Rotation_Frequency as Frequency,
	RogueMajorGlyph,
	Rogue_Options_PoisonImbue,
} from '../core/proto/rogue.js';

import * as IconInputs from '../core/components/icon_inputs.js';
import * as Mechanics from '../core/constants/mechanics.js';
import * as OtherInputs from '../core/components/other_inputs.js';
import * as Tooltips from '../core/constants/tooltips.js';

import * as RogueInputs from './inputs.js';
import * as Presets from './presets.js';

export class RogueSimUI extends IndividualSimUI<Spec.SpecRogue> {
	constructor(parentElem: HTMLElement, player: Player<Spec.SpecRogue>) {
		super(parentElem, player, {
			cssClass: 'rogue-sim-ui',
			// List any known bugs / issues here and they'll be shown on the site.
			knownIssues: [
				'Rotations are not fully optimized, especially for non-standard setups.',
				'Subtlety specs are not currently supported.'
			],
			warnings: [
				(simUI: IndividualSimUI<Spec.SpecRogue>) => {
					return {
						updateOn: simUI.sim.encounter.changeEmitter,
						getContent: () => {
							var hasNoArmor = false
							for (const target of simUI.sim.encounter.getTargets()) {
								if (target.getStats().getStat(Stat.StatArmor) <= 0) {
									hasNoArmor = true
									break
								}
							}
							if (hasNoArmor) {
								return 'One or more targets have no armor! Check advanced encounter settings.';
							} else {
								return '';
							}
						},
					};
				},
				(simUI: IndividualSimUI<Spec.SpecRogue>) => {
					return {
						updateOn: simUI.player.changeEmitter,
						getContent: () => {
							if (
								simUI.player.getTalents().mutilate &&
								(simUI.player.getGear().getEquippedItem(ItemSlot.ItemSlotMainHand)?.item.weaponType != WeaponType.WeaponTypeDagger ||
									simUI.player.getGear().getEquippedItem(ItemSlot.ItemSlotOffHand)?.item.weaponType != WeaponType.WeaponTypeDagger)
							) {
								return '\'Mutilate\' talent selected, but daggers not equipped in both hands!';
							} else {
								return '';
							}
						},
					};
				},
				(simUI: IndividualSimUI<Spec.SpecRogue>) => {
					return {
						updateOn: simUI.player.changeEmitter,
						getContent: () => {
							if (simUI.player.getRotation().useFeint && !simUI.player.getMajorGlyphs().includes(RogueMajorGlyph.GlyphOfFeint)) {
								return '\'Use Feint\' selected, but missing Glyph of Feint!';
							} else {
								return '';
							}
						},
					};
				},
				(simUI: IndividualSimUI<Spec.SpecRogue>) => {
					return {
						updateOn: simUI.player.changeEmitter,
						getContent: () => {
							const mhWeaponSpeed = simUI.player.getGear().getEquippedItem(ItemSlot.ItemSlotMainHand)?.item.weaponSpeed;
							const ohWeaponSpeed = simUI.player.getGear().getEquippedItem(ItemSlot.ItemSlotOffHand)?.item.weaponSpeed;
							const mhImbue = simUI.player.getSpecOptions().mhImbue;
							const ohImbue = simUI.player.getSpecOptions().ohImbue;
							if (typeof mhWeaponSpeed == 'undefined' || typeof ohWeaponSpeed == 'undefined' || !simUI.player.getSpecOptions().applyPoisonsManually) {
								return '';
							}
							if ((mhWeaponSpeed < ohWeaponSpeed) && (ohImbue == Rogue_Options_PoisonImbue.DeadlyPoison)) {
								return 'Deadly poison applied to slower (off hand) weapon!';
							}
							if ((ohWeaponSpeed < mhWeaponSpeed) && (mhImbue == Rogue_Options_PoisonImbue.DeadlyPoison)) {
								return 'Deadly poison applied to slower (main hand) weapon!';
							}
							return '';
						},
					};
				},
			],

			// All stats for which EP should be calculated.
			epStats: [
				Stat.StatAgility,
				Stat.StatStrength,
				Stat.StatAttackPower,
				Stat.StatMeleeHit,
				Stat.StatMeleeCrit,
				Stat.StatSpellHit,
				Stat.StatSpellCrit,
				Stat.StatMeleeHaste,
				Stat.StatArmorPenetration,
				Stat.StatExpertise,
			],
			epPseudoStats: [
				PseudoStat.PseudoStatMainHandDps,
				PseudoStat.PseudoStatOffHandDps,
			],
			// Reference stat against which to calculate EP.
			epReferenceStat: Stat.StatAttackPower,
			// Which stats to display in the Character Stats section, at the bottom of the left-hand sidebar.
			displayStats: [
				Stat.StatHealth,
				Stat.StatStamina,
				Stat.StatAgility,
				Stat.StatStrength,
				Stat.StatAttackPower,
				Stat.StatMeleeHit,
				Stat.StatSpellHit,
				Stat.StatMeleeCrit,
				Stat.StatSpellCrit,
				Stat.StatMeleeHaste,
				Stat.StatArmorPenetration,
				Stat.StatExpertise,
			],

			defaults: {
				// Default equipped gear.
				gear: Presets.PRERAID_PRESET_ASSASSINATION.gear,
				// Default EP weights for sorting gear in the gear picker.
				epWeights: Stats.fromMap({
					[Stat.StatAgility]: 1.86,
					[Stat.StatStrength]: 1.14,
					[Stat.StatAttackPower]: 1,
					[Stat.StatSpellCrit] : 0.28,
					[Stat.StatSpellHit] : 0.08,
					[Stat.StatMeleeHit]: 1.39,
					[Stat.StatMeleeCrit]: 1.32,
					[Stat.StatMeleeHaste]: 1.48,
					[Stat.StatArmorPenetration]: 0.84,
					[Stat.StatExpertise]: 0.98,
				}, {
					[PseudoStat.PseudoStatMainHandDps]: 2.94,
					[PseudoStat.PseudoStatOffHandDps]: 2.45,
				}),
				// Default consumes settings.
				consumes: Presets.DefaultConsumes,
				// Default rotation settings.
				rotation: Presets.DefaultRotation,
				// Default talents.
				talents: Presets.AssassinationTalents.data,
				// Default spec-specific settings.
				specOptions: Presets.DefaultOptions,
				// Default raid/party buffs settings.
				raidBuffs: RaidBuffs.create({
					giftOfTheWild: TristateEffect.TristateEffectImproved,
					bloodlust: true,
					strengthOfEarthTotem: TristateEffect.TristateEffectImproved,
					icyTalons: true,
					leaderOfThePack: TristateEffect.TristateEffectImproved,
					abominationsMight: true,
					swiftRetribution: true,
					elementalOath: true,
					sanctifiedRetribution: true,
				}),
				partyBuffs: PartyBuffs.create({
				}),
				individualBuffs: IndividualBuffs.create({
					blessingOfKings: true,
					blessingOfMight: TristateEffect.TristateEffectImproved,
				}),
				debuffs: Debuffs.create({
					heartOfTheCrusader: true,
					mangle: true,
					sunderArmor: true,
					faerieFire: TristateEffect.TristateEffectImproved,
					shadowMastery: true,
					earthAndMoon: true,
					bloodFrenzy: true,
				}),
			},

			playerInputs: {
				inputs: [
					RogueInputs.ApplyPoisonsManually
				]
			},
			// IconInputs to include in the 'Player' section on the settings tab.
			playerIconInputs: [
				RogueInputs.MainHandImbue,
				RogueInputs.OffHandImbue,
			],
			// Inputs to include in the 'Rotation' section on the settings tab.
			rotationInputs: RogueInputs.RogueRotationConfig,
			// Buff and Debuff inputs to include/exclude, overriding the EP-based defaults.
			includeBuffDebuffInputs: [
				IconInputs.SpellCritBuff,
				IconInputs.SpellCritDebuff,
				IconInputs.SpellHitDebuff,
				IconInputs.SpellDamageDebuff
			],
			excludeBuffDebuffInputs: [
			],
			// Inputs to include in the 'Other' section on the settings tab.
			otherInputs: {
				inputs: [
					RogueInputs.StartingOverkillDuration,
					OtherInputs.StartingConjured,
					OtherInputs.TankAssignment,
					OtherInputs.InFrontOfTarget,
				],
			},
			encounterPicker: {
				// Whether to include 'Execute Duration (%)' in the 'Encounter' section of the settings tab.
				showExecuteProportion: false,
			},

			presets: {
				// Preset talents that the user can quickly select.
				talents: [
					Presets.AssassinationTalents,
					Presets.CombatTalents,
				],
				// Preset gear configurations that the user can quickly select.
				gear: [
					Presets.PRERAID_PRESET_ASSASSINATION,
					Presets.PRERAID_PRESET_COMBAT,
					Presets.P1_PRESET_ASSASSINATION,
					Presets.P1_PRESET_COMBAT,
				],
			},
		})
		this.player.changeEmitter.on((c) => {
			const rotation = this.player.getRotation()
			const options = this.player.getSpecOptions()
			const encounter = this.sim.encounter
			if (this.player.getTalents().mutilate) {
				if (rotation.assassinationFinisherPriority == AssassinationPriority.AssassinationPriorityUnknown) {
					rotation.assassinationFinisherPriority = Presets.DefaultRotation.assassinationFinisherPriority;
				}
				rotation.combatFinisherPriority = CombatPriority.CombatPriorityUnknown;
			} else {
				rotation.assassinationFinisherPriority = AssassinationPriority.AssassinationPriorityUnknown;
				if (rotation.combatFinisherPriority == CombatPriority.CombatPriorityUnknown) {
					rotation.combatFinisherPriority = Presets.DefaultRotation.combatFinisherPriority;
				}
			}
			this.player.setRotation(c, rotation)
			if (!options.applyPoisonsManually) {
				const mhWeaponSpeed = this.player.getGear().getEquippedItem(ItemSlot.ItemSlotMainHand)?.item.weaponSpeed;
				const ohWeaponSpeed = this.player.getGear().getEquippedItem(ItemSlot.ItemSlotOffHand)?.item.weaponSpeed;
				if (typeof mhWeaponSpeed == 'undefined' || typeof ohWeaponSpeed == 'undefined') {
					return
				}
				if (encounter.getNumTargets() > 3) {
					options.mhImbue = Rogue_Options_PoisonImbue.InstantPoison
					options.ohImbue = Rogue_Options_PoisonImbue.InstantPoison
				} else {
					if (mhWeaponSpeed <= ohWeaponSpeed) { 
						options.mhImbue = Rogue_Options_PoisonImbue.DeadlyPoison 
						options.ohImbue = Rogue_Options_PoisonImbue.InstantPoison
					} else {
						options.mhImbue = Rogue_Options_PoisonImbue.InstantPoison
						options.ohImbue = Rogue_Options_PoisonImbue.DeadlyPoison
					}
				}
			}
			this.player.setSpecOptions(c, options)
		});
		this.sim.encounter.changeEmitter.on((c) => {
			const rotation = this.player.getRotation()
			const options = this.player.getSpecOptions()
			const encounter = this.sim.encounter
			if (this.sim.encounter.getNumTargets() > 3) {
				if (rotation.multiTargetSliceFrequency == Frequency.FrequencyUnknown) {
					rotation.multiTargetSliceFrequency = Presets.DefaultRotation.multiTargetSliceFrequency;
				}
			} else {
				rotation.multiTargetSliceFrequency = Frequency.FrequencyUnknown;
			}
			this.player.setRotation(c, rotation)
			if (!options.applyPoisonsManually) {
				const mhWeaponSpeed = this.player.getGear().getEquippedItem(ItemSlot.ItemSlotMainHand)?.item.weaponSpeed;
				const ohWeaponSpeed = this.player.getGear().getEquippedItem(ItemSlot.ItemSlotOffHand)?.item.weaponSpeed;
				if (typeof mhWeaponSpeed == 'undefined' || typeof ohWeaponSpeed == 'undefined') {
					return
				}
				if (encounter.getNumTargets() > 3) {
					options.mhImbue = Rogue_Options_PoisonImbue.InstantPoison
					options.ohImbue = Rogue_Options_PoisonImbue.InstantPoison
				} else {
					if (mhWeaponSpeed <= ohWeaponSpeed) { 
						options.mhImbue = Rogue_Options_PoisonImbue.DeadlyPoison 
						options.ohImbue = Rogue_Options_PoisonImbue.InstantPoison
					} else {
						options.mhImbue = Rogue_Options_PoisonImbue.InstantPoison
						options.ohImbue = Rogue_Options_PoisonImbue.DeadlyPoison
					}
				}
			}
			this.player.setSpecOptions(c, options)
		});
	}
}
