import { Exporter } from '../core/components/exporters';
import { Importer } from '../core/components/importers';
import { MAX_PARTY_SIZE } from '../core/party';
import { RaidSimSettings } from '../core/proto/ui';
import { EventID, TypedEvent } from '../core/typed_event';
import { Party as PartyProto, Player as PlayerProto, Raid as RaidProto } from '../core/proto/api';
import {
	Class,
	Encounter as EncounterProto,
	EquipmentSpec,
	Faction,
	ItemSpec,
	MobType,
	Profession,
	RaidTarget,
	Spec,
	Target as TargetProto,
} from '../core/proto/common';
import { nameToClass } from '../core/proto_utils/names';
import {
	DruidSpecs,
	DeathknightSpecs,
	MageSpecs,
	PriestSpecs,
	RogueSpecs,
	getTalentTreePoints,
	makeDefaultBlessings,
	specTypeFunctions,
	withSpecProto,
	isTankSpec,
	playerToSpec,
} from '../core/proto_utils/utils';
import { MAX_NUM_PARTIES } from '../core/raid';
import { Player } from '../core/player';
import { Target } from '../core/target';
import { sortByProperty } from '../core/utils';

import { playerPresets, PresetSpecSettings } from './presets';
import { RaidSimUI } from './raid_sim_ui';

declare var $: any;
declare var tippy: any;

export class RaidJsonImporter extends Importer {
	private readonly simUI: RaidSimUI;
	constructor(parent: HTMLElement, simUI: RaidSimUI) {
		super(parent, simUI, 'JSON Import', true);
		this.simUI = simUI;

		this.descriptionElem.innerHTML = `
			<p>
				Import settings from a JSON text file, which can be created using the JSON Export feature of this site.
			</p>
			<p>
				To import, paste the JSON text below and click, 'Import'.
			</p>
		`;
	}

	onImport(data: string) {
		const settings = RaidSimSettings.fromJsonString(data, { ignoreUnknownFields: true });
		this.simUI.fromProto(TypedEvent.nextEventID(), settings);
		this.close();
	}
}

export class RaidJsonExporter extends Exporter {
	private readonly simUI: RaidSimUI;

	constructor(parent: HTMLElement, simUI: RaidSimUI) {
		super(parent, simUI, 'JSON Export', true);
		this.simUI = simUI;
		this.init();
	}

	getData(): string {
		return JSON.stringify(RaidSimSettings.toJson(this.simUI.toProto()), null, 2);
	}
}

export class RaidWCLImporter extends Importer {

	private queryCounter: number = 0;

	private readonly simUI: RaidSimUI;
	constructor(parent: HTMLElement, simUI: RaidSimUI) {
		super(parent, simUI, 'WCL Import', false);
		this.simUI = simUI;
		this.textElem.classList.add('small-textarea');
		this.descriptionElem.innerHTML = `
			<p>
				Imports the entire raid from a WCL report.<br>
			</p>
			<p>
				To import, paste the WCL report and fight link (https://classic.warcraftlogs.com/reports/REPORTID#fight=FIGHTID).<br>
				Include the fight ID or else the first fight in the report will be used.<br>
			</p>
			<p>
				The following are imported directly from the report:
				<ul>
					<li>Player Name</li>
					<li>Equipment (items, enchants, and gems)</li>
					<li>Faction (Alliance / Horde)</li>
					<li>Encounter: If the import link has a fight ID we try to match with a preset Encounter. Note that many Encounters are still unimplemented.</li>
				</ul>

				The following are not available directly from the report data, but we try to infer them:
				<ul>
					<li>Talents: Log data only gives us the tree summary (e.g. '51/20/0') so we match this with the closest preset talent build.</li>
					<li>Glyphs: Glyphs are absent from log data, but we pair them with the inferred Talents.</li>
					<li>Professions: Inferred from profession-locked items/enchants/gems.</li>
					<li>Buff assignments (Innervate, Unholy Frenzy, etc): Inferred from casts.</li>
				</ul>

				The following are not imported, and instead use spec-specific defaults:
				<ul>
					<li>Race</li>
					<li>Rotation / Spec-specific options</li>
					<li>Consumes</li>
					<li>Paladin Blessings</li>
					<li>Party Composition</li>
				</ul>
			</p>
		`;
	}

	private token: string = '';
	private async getWCLBearerToken(): Promise<string> {
		if (this.token == '') {
			const response = await fetch('https://classic.warcraftlogs.com/oauth/token', {
				'method': 'POST',
				'headers': {
					'Authorization': 'Basic ' + btoa('963d31c8-7efa-4dde-87cf-1b254a8a2f8c:lRJVhujEEnF96xfUoxVHSpnqKN9v8bTqGEjutsO3'),
				},
				body: new URLSearchParams({
					'grant_type': 'client_credentials',
				}),
			})
			const json = await response.json();
			this.token = json.access_token;
		}
		return this.token;
	}

	private async queryWCL(query: string): Promise<any> {
		const token = await this.getWCLBearerToken();
		const headers = {
			'Content-Type': 'application/json',
			'Authorization': `Bearer ${token}`,
			'Accept': 'application/json',
		};

		const queryURL = `https://classic.warcraftlogs.com/api/v2/client?query=${query}`;
		this.queryCounter++;

		// Query WCL
		const res = await fetch(encodeURI(queryURL), {
			'method': 'GET',
			'headers': headers,
		});

		const result = await res.json();
		if (result?.errors?.length) {
			const errorStr = result.errors.map((e: any) => e.message).join('\n');
			throw new Error(`GraphQL error: ${errorStr}\n\nQuery: ${query}`);
		} else {
			console.debug(`WCL query: ${query}\n\nResult: ${JSON.stringify(result)}`);
		}
		return result;
	}

	private async parseURL(url: string): Promise<wclUrlData> {
		const match = url.match(/classic\.warcraftlogs\.com\/reports\/([a-zA-Z0-9:]+)(#.*fight=((\d+)|(last)))?/);
		if (!match) {
			throw new Error(`Invalid WCL URL ${url}, must look like "classic.warcraftlogs.com/reports/XXXX"`);
		}

		const urlData = {
			reportID: match[1],
			fightID: '',
		}

		// If the URL has a Fight ID in it, use it
		if (match[2] && match[3] && match[3] != 'last') {
			urlData.fightID = match[3];
		} else {
			// Make a separate query to get the corresponding ReportFights
			const fightDataQuery = `{
				reportData {
					report(code: "${urlData.reportID}") {
						fights(killType: Kills, translate: true) {
							id, name
						}
					}
				}
			}`;

			const fightData = await this.queryWCL(fightDataQuery);
			const fights = fightData.data.reportData.report.fights;

			if (match[3] == 'last') {
				urlData.fightID = String(fights[fights.length - 1].id)
			} else {
				// Default to using the first Fight
				urlData.fightID = String(fights[0].id);
			}
		}

		console.debug(`Importing WCL report: ${JSON.stringify(urlData)}`);
		return urlData;
	}

	private async getRateLimit(): Promise<wclRateLimitData> {
		const query = `{
	    rateLimitData {
	      limitPerHour, pointsSpentThisHour, pointsResetIn
	    }
	  }`;
		const result = await this.queryWCL(query);
		const data = result['data']['rateLimitData'] as wclRateLimitData;
		return data;
	}

	async onImport(importLink: string) {
		this.importButton.disabled = true;
		this.rootElem.style.cursor = 'wait';
		try {
			await this.doImport(importLink);
		} catch (error) {
			alert('Failed import from WCL: ' + error);
		}
		this.importButton.disabled = false
		this.rootElem.style.removeProperty('cursor');
	}

	async doImport(importLink: string) {
		if (!importLink.length) {
			throw new Error('No import link provided!');
		}

		const urlData = await this.parseURL(importLink);
		const rateLimit = await this.getRateLimit();

		const reportDataQuery = `{
			reportData {
				report(code: "${urlData.reportID}") {
					guild {
						name faction {id}
					}
					playerDetails: table(fightIDs: [${urlData.fightID}], endTime: 99999999, dataType: Casts, killType: All, viewBy: Default)
					fights(fightIDs: [${urlData.fightID}]) {
						startTime, endTime, id, name
					}
					innervates: table(fightIDs: [${urlData.fightID}], dataType:Casts, endTime: 99999999, sourceClass: "Druid", abilityID: 29166),
					powerInfusion: table(fightIDs: [${urlData.fightID}], dataType:Casts, endTime: 99999999, sourceClass: "Priest", abilityID: 10060)
					tricksOfTheTrade: table(fightIDs: [${urlData.fightID}], dataType:Casts, endTime: 99999999, sourceClass: "Rogue", abilityID: 57933)
					unholyFrenzy: table(fightIDs: [${urlData.fightID}], dataType:Casts, endTime: 99999999, sourceClass: "DeathKnight", abilityID: 49016)
				}
			}
		}`;
		const reportData = await this.queryWCL(reportDataQuery);

		// Process the report data.
		const wclData = reportData.data.reportData.report; // TODO: Typings?
		const playerData: wclPlayer[] = wclData.playerDetails.data.entries;

		// If defined in log, use that faction. Otherwise default to UI setting.
		const faction = (wclData.guild?.faction?.id || this.simUI.raidPicker?.getCurrentFaction() || Faction.Horde) as Faction;

		const wclPlayers = playerData.map(wclPlayer => new WCLSimPlayer(wclPlayer, this.simUI, faction, TypedEvent.nextEventID()));
		await this.inferPartyComposition(urlData, wclData, wclPlayers);

		TypedEvent.freezeAllAndDo(() => {
			const eventID = TypedEvent.nextEventID();
			this.inferAssignments(eventID, wclData, wclPlayers);
			const numPaladins = playerData.filter(player => player.type == 'Paladin').length;
			const settings = RaidSimSettings.create({
				encounter: this.getEncounterProto(wclData),
				raid: this.getRaidProto(wclPlayers),
				blessings: makeDefaultBlessings(numPaladins),
			});

			// Clear the raid out to avoid any taint issues.
			this.simUI.clearRaid(eventID);
			this.simUI.fromProto(eventID, settings);
		});

		this.close();
	}

	// Assigns the raidIndex field for all players.
	private async inferPartyComposition(urlData: wclUrlData, wclData: any, wclPlayers: WCLSimPlayer[]) {
		// If generateParties is true, we will generate parties based on the party buffers.
		// Slower but more accurate way to generate the raid sim.
		// Generates players into the groups that they were in during the fight.
		// If the rate limit is close to max, then it will create the raid parties 'randomly'.
		//const rateLimitBuffer = 30; // WCL Query point buffer
		//const generateParties: boolean = rateLimitStart.pointsSpentThisHour + rateLimitBuffer < rateLimitStart.limitPerHour;
		//if (generateParties) {
		//	// Can't be a forEach because we need to wait for the query to finish on each iteration later on.
		//	for (const player of wclPlayers) {
		//		const partyFull = player.partyMembers.length >= MAX_PARTY_SIZE;

		//		// Skip players that have already been assigned to a party.
		//		// player.partyAssigned || player.partyFound || player.partyMembers.length > 0
		//		if (partyFull) {
		//			continue;
		//		}

		//		const auraIDs: number[] = player.getPartyAuraIds();
		//		if (!auraIDs.length) {
		//			console.warn('No party aura ids found for partyBuff player ' + player.name);
		//			continue;
		//		}

		//		const fight: { startTime: number, endTime: number, id: number, name: string } = wclData.fights[0];
		//		let auraBuffQueries = auraIDs.map((auraID) => `{
		//			reportData {
		//				report(code: "${urlData.reportID}") {
		//					table(startTime: ${fight.startTime}, endTime: ${fight.endTime}, sourceID: ${player.id}, abilityID: ${auraID}, fightIDs: [${urlData.fightID}],dataType:Buffs,viewBy:Target,hostilityType:Friendlies)
		//				}
		//			}
		//		}`);

		//		let auraTargets: wclAura[] = [];

		//		// Can't be a forEach because we need to await each query.
		//		for (let i = 0; i < auraBuffQueries.length; i++) {
		//			if (auraTargets.length >= MAX_PARTY_SIZE || partyFull) {
		//				break;
		//			}

		//			let auraQueryRes = await this.queryWCL(auraBuffQueries[i]);
		//			if (auraQueryRes) {
		//				let playerAuras: wclAura[] = auraQueryRes.data?.reportData?.report?.table?.data?.auras ?? [];
		//				if (playerAuras.length) {

		//					playerAuras = playerAuras.filter((auraTarget) => auraTarget.type !== 'Pet')
		//						.sort((a, b) => a.bands[0].startTime - b.bands[0].startTime)
		//						.filter((auraTarget, index) => index < 5);

		//					const uniqueAuraTargets = playerAuras.filter((auraTarget) => !auraTargets.some((target) => target.name === auraTarget.name));
		//					auraTargets.push(...uniqueAuraTargets);
		//				}
		//			}
		//		}

		//		if (auraTargets.length === 0) {
		//			continue;
		//		}

		//		// Only need the member names at this point.
		//		player.partyMembers = auraTargets.map((auraTarget) => auraTarget.name);

		//		let partyMembers = wclPlayers
		//			.filter(raidMember => player.partyMembers.includes(raidMember.name))
		//			.filter(raidMember => !raidMember.partyAssigned);

		//		const totalPartyMembers = partyMembers.length;
		//	}
		//}

		// Assign remaining players into open slots.
		const allRaidIndexes = [...Array(25).keys()];
		const nextFreeIndex = () => allRaidIndexes.find(idx => !wclPlayers.some(p => p.raidIndex == idx)) ?? -1;
		wclPlayers
			.filter(player => player.raidIndex == -1)
			.forEach(player => {
				const nextIdx = nextFreeIndex();
				if (nextIdx == -1) {
					throw new Error('Invalid next idx');
				}
				player.raidIndex = nextIdx;
			});
	}

	private inferAssignments(eventID: EventID, wclData: any, wclPlayers: WCLSimPlayer[]) {
		const processBuffCastData = (buffCastData: wclBuffCastsData[]): { player: WCLSimPlayer, target: WCLSimPlayer }[] => {
			const playerCasts: { player: WCLSimPlayer, target: WCLSimPlayer }[] = [];
			if (buffCastData.length) {
				buffCastData.forEach(cast => {
					const sourcePlayer = wclPlayers.find((player) => player.name === cast.name);
					const targetPlayer = wclPlayers.find((player) => player.name === cast.targets[0].name);

					if (sourcePlayer && targetPlayer) {
						playerCasts.push({ player: sourcePlayer, target: targetPlayer });
					}
				});
			}
			return playerCasts;
		}

		processBuffCastData(wclData.innervates.data.entries).forEach(cast => {
			if (cast.player.player.getClass() == Class.ClassDruid) {
				const player = cast.player.player as Player<DruidSpecs>;
				const options = player.getSpecOptions();
				options.innervateTarget = cast.target.toRaidTarget();
				player.setSpecOptions(eventID, options);
			}
		});
		processBuffCastData(wclData.powerInfusion.data.entries).forEach(cast => {
			if (cast.player.player.getClass() == Class.ClassPriest) {
				const player = cast.player.player as Player<PriestSpecs>;
				const options = player.getSpecOptions();
				options.powerInfusionTarget = cast.target.toRaidTarget();
				player.setSpecOptions(eventID, options);
			}
		});
		processBuffCastData(wclData.tricksOfTheTrade.data.entries).forEach(cast => {
			if (cast.player.player.getClass() == Class.ClassRogue) {
				const player = cast.player.player as Player<RogueSpecs>;
				const options = player.getSpecOptions();
				options.tricksOfTheTradeTarget = cast.target.toRaidTarget();
				player.setSpecOptions(eventID, options);
			}
		});
		processBuffCastData(wclData.unholyFrenzy.data.entries).forEach(cast => {
			if (cast.player.player.getClass() == Class.ClassDeathknight) {
				const player = cast.player.player as Player<DeathknightSpecs>;
				const options = player.getSpecOptions();
				options.unholyFrenzyTarget = cast.target.toRaidTarget();
				player.setSpecOptions(eventID, options);
			}
		});
	}

	private getEncounterProto(wclData: any): EncounterProto {
		const fight: { startTime: number, endTime: number, id: number, name: string } = wclData.fights[0];

		const encounter = EncounterProto.create({
			duration: (fight.endTime - fight.startTime) / 1000,
			targets: [],
		});

		// Use the preset encounter if it exists.
		let closestEncounterPreset = this.simUI.sim.db.getAllPresetEncounters().find(enc => enc.path.includes(fight.name));
		if (closestEncounterPreset && closestEncounterPreset.targets.length) {
			closestEncounterPreset.targets
				.map(mob => mob.target as TargetProto)
				.filter(target => target !== undefined)
				.forEach(target => encounter.targets.push(target));
		}

		// Build a manual target list if no preset encounter exists.
		if (encounter.targets.length === 0) {
			encounter.targets.push(Target.defaultProto());
		}

		return encounter;
	}

	private getRaidProto(wclPlayers: WCLSimPlayer[]): RaidProto {
		const raid = RaidProto.create({
			parties: [...new Array(MAX_NUM_PARTIES).keys()].map(p => PartyProto.create({
				players: [...new Array(5).keys()].map(p => PlayerProto.create()),
			})),
		});

		wclPlayers
			.forEach(player => {
				const positionInParty = player.raidIndex % 5;
				const partyIdx = (player.raidIndex - positionInParty) / 5;
				const playerProto = player.player.toProto();
				raid.parties[partyIdx].players[positionInParty] = playerProto;

				if (isTankSpec(playerToSpec(playerProto))) {
					raid.tanks.push(player.toRaidTarget());
				}
			});

		return raid;
	}
}

class WCLSimPlayer {
	public readonly data: wclPlayer;
	public readonly id: number;
	public readonly name: string;
	public readonly type: string;
	public raidIndex: number = -1;

	private readonly simUI: RaidSimUI;
	private readonly fullType: string;
	private readonly spec: Spec|null;
	private readonly faction: Faction;

	readonly player: Player<any>;
	readonly preset: PresetSpecSettings<any>;

	constructor(data: wclPlayer, simUI: RaidSimUI, faction: Faction = Faction.Unknown, eventID: EventID) {
		this.simUI = simUI;
		this.data = data;

		this.name = data.name;
		this.id = data.id;
		this.type = data.type;
		this.faction = faction;

		const wclSpec = data.icon.split('-')[1];
		this.fullType = this.type + wclSpec;
		console.log(`WCL spec: ${this.fullType}`);

		const foundSpec = fullTypeToSpec[this.fullType] ?? null;
		if (foundSpec == null) {
			throw new Error('Player type not implemented: ' + this.fullType);
		}
		this.spec = foundSpec;
		this.player = new Player(this.spec, simUI.sim);

		this.preset = WCLSimPlayer.getMatchingPreset(foundSpec, data.talents);
		if (this.preset === undefined) {
			throw new Error('Could not find matching preset: ' + JSON.stringify({
				'name': this.name,
				'type': this.fullType,
				'talents': data.talents,
			}).toString());
		}

		// Apply preset defaults.
		this.player.applySharedDefaults(eventID);
		this.player.setRace(eventID, this.preset.defaultFactionRaces[this.faction]);
		this.player.setTalentsString(eventID, this.preset.talents.talentsString);
		this.player.setGlyphs(eventID, this.preset.talents.glyphs!);
		this.player.setConsumes(eventID, this.preset.consumes);
		this.player.setRotation(eventID, this.preset.rotation);
		this.player.setSpecOptions(eventID, this.preset.specOptions);

		// Apply settings from report data.
		this.player.setName(eventID, data.name);
		this.player.setGear(eventID, simUI.sim.db.lookupEquipmentSpec(EquipmentSpec.create({
			items: data.gear.map(gear => ItemSpec.create({
				id: gear.id,
				enchant: gear.permanentEnchant,
				gems: gear.gems ? gear.gems.map(gemInfo => gemInfo.id) : [],
			})),
		})));

		let professions = this.player.getGear().getProfessionRequirements();
		if (professions.length == 0) {
			professions = [Profession.Engineering, Profession.Jewelcrafting];
		} else if (professions.length == 1) {
			if (professions[0] != Profession.Engineering) {
				professions.push(Profession.Engineering);
			} else {
				professions.push(Profession.Jewelcrafting);
			}
		}
		this.player.setProfessions(eventID, professions);
	}

	private static getMatchingPreset(spec: Spec, talents: wclTalents[]): PresetSpecSettings<Spec> {
		const matchingPresets = playerPresets.filter((preset) => preset.spec == spec);
		let presetIdx = 0;

		if (matchingPresets && matchingPresets.length > 1) {
			let distance = 999;
			// Search talents and find the preset that the players talents most closely match.
			matchingPresets.forEach((preset, i) => {
				const presetTalents = getTalentTreePoints(preset.talents.talentsString);
				// Diff the distance to the preset.
				const newDistance = presetTalents.reduce((acc, v, i) => acc += Math.abs(talents[i]?.guid - presetTalents[i]), 0);

				// If this is the best distance, assign this preset.
				if (newDistance < distance) {
					presetIdx = i;
					distance = newDistance;
				}
			});
		}
		return matchingPresets[presetIdx];
	}

	public toRaidTarget(): RaidTarget {
		return RaidTarget.create({
			targetIndex: this.raidIndex,
		});
	}

	public getPartyAuraIds(): Array<number> {
		// TODO: Update this function for WOTLK
		return [];

		const allSpecClassAuras: any = {
			'Paladin': [
				19746, // Concentration Aura
				27149, // Devotion Aura,
				27150, // Retribution Aura
			],
			'Warrior': [
			],
			'Warlock': [
				27268, // Pet Imp: Blood Pact
				18696, // Improved Imp: Blood Pact
			],
		};

		// Reused for the plethora of Feral specs.
		const feralDruidSpecAuras = [
			24932, // Improved Leader of the Pack // at least 0,32,0
			// 17007, // Leader of the Pack // at least 0,31,0
		];

		// TODO: Could additionally filter out buff IDs based on minimum req talent strings?
		const specSpecificAuras: any = {
			'RetributionPaladin': [
				20092, // Improved Retribution Aura // at least 0,0,16
				20218, // Sanctity Aura // at least 0,0,21
				31870, // Improved Sanctity Aura // at least 0,0,22
			],
			'GuardianDruid': [...feralDruidSpecAuras],
			'WardenDruid': [...feralDruidSpecAuras],
			'FeralDruid': [...feralDruidSpecAuras],
			'BalanceDruid': [
				24907, // Moonkin Aura // at least 31,0,0
			],
			'RestorationDruid': [
				34123, // Tree of Life // at least 0,0,41
			],
			'MarksmanHunter': [
				27066, // Trueshot Aura // at least 0,32,0
			],
			'EnhancementShaman': [
				30811, // Unleashed Rage // at least 0,36,0
			],
			// 'ElementalShaman': [] // Totem buffs do not show up in logs. Leaving for future reference.
		};

		const classAuras = allSpecClassAuras[this.type] ?? [];
		const specAuras = specSpecificAuras[this.fullType] ?? [];

		const reliableAuras = [
			...specAuras, ...classAuras,
		];

		return reliableAuras;
	}
}

const fullTypeToSpec: Record<string, Spec> = {
	'DeathKnightBlood': Spec.SpecTankDeathknight,
	'DeathKnightLichborne': Spec.SpecTankDeathknight,
	'DeathKnightRuneblade': Spec.SpecDeathknight,
	'DeathKnightFrost': Spec.SpecDeathknight,
	'DeathKnightUnholy': Spec.SpecDeathknight,
	'DruidBalance': Spec.SpecBalanceDruid,
	'DruidFeral': Spec.SpecFeralDruid,
	'DruidWarden': Spec.SpecFeralTankDruid,
	'DruidGuardian': Spec.SpecFeralTankDruid,
	'DruidRestoration': Spec.SpecRestorationDruid,
	'HunterBeastMastery': Spec.SpecHunter,
	'HunterSurvival': Spec.SpecHunter,
	'HunterMarksmanship': Spec.SpecHunter,
	'MageArcane': Spec.SpecMage,
	'MageFire': Spec.SpecMage,
	'MageFrost': Spec.SpecMage,
	'PaladinHoly': Spec.SpecHolyPaladin,
	'PaladinJusticar': Spec.SpecProtectionPaladin,
	'PaladinProtection': Spec.SpecProtectionPaladin,
	'PaladinRetribution': Spec.SpecRetributionPaladin,
	'PriestHoly': Spec.SpecHealingPriest,
	'PriestDiscipline': Spec.SpecHealingPriest,
	'PriestShadow': Spec.SpecShadowPriest,
	'PriestSmite': Spec.SpecSmitePriest,
	'RogueAssassination': Spec.SpecRogue,
	'RogueCombat': Spec.SpecRogue,
	'RogueSubtlety': Spec.SpecRogue,
	'ShamanElemental': Spec.SpecElementalShaman,
	'ShamanEnhancement': Spec.SpecEnhancementShaman,
	'ShamanRestoration': Spec.SpecRestorationShaman,
	'WarlockDestruction': Spec.SpecWarlock,
	'WarlockAffliction': Spec.SpecWarlock,
	'WarlockDemonology': Spec.SpecWarlock,
	'WarriorArms': Spec.SpecWarrior,
	'WarriorFury': Spec.SpecWarrior,
	'WarriorChampion': Spec.SpecWarrior,
	'WarriorWarrior': Spec.SpecWarrior,
	'WarriorGladiator': Spec.SpecWarrior,
	'WarriorProtection': Spec.SpecProtectionWarrior,
};

interface wclUrlData {
	reportID: string,
	fightID: string,
}

interface wclBuffCastsData {
	name: string;
	targets: {
		name: string;
		type: string;
	}[];
}

interface wclRateLimitData {
	limitPerHour: number,
	pointsSpentThisHour: number,
	pointsResetIn: number
}

// Typed interface for WCL talents
interface wclTalents {
	name: string;
	guid: number;
	type: number;
	abilityIcon: string;
}

// Typed interface for WCL Gems
interface wclGems {
	id: number;
	itemLevel: number;
	icon: string;
}

// Typed interface for WCL Gear
interface wclGear {
	id: number;
	slot: number;
	quality: number;
	icon: string;
	name: string;
	itemLevel: number;
	permanentEnchant: number;
	permanentEnchantName: string;
	temporaryEnchant: number;
	gems?: wclGems[];
}

// Typed interface for WCL Player Data
interface wclPlayer {
	name: string;
	id: number;
	guid?: number;
	type: string; // Paladin, Mage, etc.
	icon: string; // Paladin-Justicar, Mage-Fire, etc.
	itemLevel?: number;
	total?: number;
	activeTime?: number;
	activeTimeReduced?: number;
	abilities?: unknown[]; // Don't care about abilities.
	damageAbilities?: unknown[];
	targets?: unknown[];
	talents: wclTalents[];
	gear: wclGear[];
}

interface wclAura {
	name: string;
	id: number;
	guid: number;
	type: string;
	icon: string;
	totalUptime: number;
	totalUses: number;
	bands: {
		startTime: number,
		endTime: number,
	}[];
}
