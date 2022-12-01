import { CustomRotation } from '../core/proto/common.js';
import { Race, RaidTarget } from '../core/proto/common.js';
import { Spec } from '../core/proto/common.js';
import { NO_TARGET } from '../core/proto_utils/utils.js';
import { ActionId } from '../core/proto_utils/action_id.js';
import { Player } from '../core/player.js';
import { Sim } from '../core/sim.js';
import { IndividualSimUI } from '../core/individual_sim_ui.js';
import { Target } from '../core/target.js';
import { EventID, TypedEvent } from '../core/typed_event.js';

import {
	HealingPriest,
	HealingPriest_Rotation as PriestRotation,
	HealingPriest_Rotation_RotationType as RotationType,
	HealingPriest_Rotation_SpellOption as SpellOption,
} from '../core/proto/priest.js';

import * as InputHelpers from '../core/components/input_helpers.js';

// Configuration for spec-specific UI elements on the settings tab.
// These don't need to be in a separate file but it keeps things cleaner.

export const SelfPowerInfusion = InputHelpers.makeSpecOptionsBooleanIconInput<Spec.SpecHealingPriest>({
	fieldName: 'powerInfusionTarget',
	id: ActionId.fromSpellId(10060),
	extraCssClasses: [
		'within-raid-sim-hide',
	],
	getValue: (player: Player<Spec.SpecHealingPriest>) => player.getSpecOptions().powerInfusionTarget?.targetIndex != NO_TARGET,
	setValue: (eventID: EventID, player: Player<Spec.SpecHealingPriest>, newValue: boolean) => {
		const newOptions = player.getSpecOptions();
		newOptions.powerInfusionTarget = RaidTarget.create({
			targetIndex: newValue ? 0 : NO_TARGET,
		});
		player.setSpecOptions(eventID, newOptions);
	},
});

export const InnerFire = InputHelpers.makeSpecOptionsBooleanIconInput<Spec.SpecHealingPriest>({
	fieldName: 'useInnerFire',
	id: ActionId.fromSpellId(48168),
});

export const Shadowfiend = InputHelpers.makeSpecOptionsBooleanIconInput<Spec.SpecHealingPriest>({
	fieldName: 'useShadowfiend',
	id: ActionId.fromSpellId(34433),
});

export const HealingPriestRotationConfig = {
	inputs: [
		InputHelpers.makeRotationEnumInput<Spec.SpecHealingPriest, RotationType>({
			fieldName: 'type',
			label: 'Type',
			values: [
				{ name: 'Cycle', value: RotationType.Cycle },
				{ name: 'Custom', value: RotationType.Custom },
			],
		}),
		InputHelpers.makeCustomRotationInput<Spec.SpecHealingPriest, SpellOption>({
			fieldName: 'customRotation',
			numColumns: 2,
			showCastsPerMinute: true,
			values: [
				{ actionId: ActionId.fromSpellId(48063), value: SpellOption.GreaterHeal },
				{ actionId: ActionId.fromSpellId(48071), value: SpellOption.FlashHeal },
				{ actionId: ActionId.fromSpellId(48068), value: SpellOption.Renew },
				{ actionId: ActionId.fromSpellId(48066), value: SpellOption.PowerWordShield },
				{ actionId: ActionId.fromSpellId(48089), value: SpellOption.CircleOfHealing },
				{ actionId: ActionId.fromSpellId(48072), value: SpellOption.PrayerOfHealing },
				{ actionId: ActionId.fromSpellId(48113), value: SpellOption.PrayerOfMending },
				{ actionId: ActionId.fromSpellId(53007), value: SpellOption.Penance },
				{ actionId: ActionId.fromSpellId(48120), value: SpellOption.BindingHeal },
			],
			showWhen: (player: Player<Spec.SpecHealingPriest>) => player.getRotation().type == RotationType.Custom,
		}),
	],
};
