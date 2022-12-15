import { EquippedItem } from '../proto_utils/equipped_item.js';
import { getEmptyGemSocketIconUrl, gemMatchesSocket } from '../proto_utils/gems.js';
import { setGemSocketCssClass } from '../proto_utils/gems.js';
import { Stats } from '../proto_utils/stats.js';
import { Class, GemColor } from '../proto/common.js';
import { HandType } from '../proto/common.js';
import { WeaponType } from '../proto/common.js';
import { ItemQuality } from '../proto/common.js';
import { ItemSlot } from '../proto/common.js';
import { ItemType } from '../proto/common.js';
import { Profession } from '../proto/common.js';
import { getEnchantDescription, getUniqueEnchantString } from '../proto_utils/enchants.js';
import { ActionId } from '../proto_utils/action_id.js';
import { slotNames } from '../proto_utils/names.js';
import { setItemQualityCssClass } from '../css_utils.js';
import { Player } from '../player.js';
import { EventID, TypedEvent } from '../typed_event.js';
import { formatDeltaTextElem } from '../utils.js';
import { getEnumValues } from '../utils.js';
import {
	UIEnchant as Enchant,
	UIGem as Gem,
	UIItem as Item,
} from '../proto/ui.js';

import { Component } from './component.js';
import { FiltersMenu } from './filters_menu.js';
import { Popup } from './popup.js';
import { makePhaseSelector } from './other_inputs.js';
import { makeShow1hWeaponsSelector } from './other_inputs.js';
import { makeShow2hWeaponsSelector } from './other_inputs.js';
import { makeShowMatchingGemsSelector } from './other_inputs.js';

declare var $: any;
declare var tippy: any;
declare var WowSim: any;

export class GearPicker extends Component {
	// ItemSlot is used as the index
	readonly itemPickers: Array<ItemPicker>;

	constructor(parent: HTMLElement, player: Player<any>) {
		super(parent, 'gear-picker-root');

		const leftSide = document.createElement('div');
		leftSide.classList.add('gear-picker-left');
		this.rootElem.appendChild(leftSide);

		const rightSide = document.createElement('div');
		rightSide.classList.add('gear-picker-right');
		this.rootElem.appendChild(rightSide);

		const leftItemPickers = [
			ItemSlot.ItemSlotHead,
			ItemSlot.ItemSlotNeck,
			ItemSlot.ItemSlotShoulder,
			ItemSlot.ItemSlotBack,
			ItemSlot.ItemSlotChest,
			ItemSlot.ItemSlotWrist,
			ItemSlot.ItemSlotMainHand,
			ItemSlot.ItemSlotOffHand,
			ItemSlot.ItemSlotRanged,
		].map(slot => new ItemPicker(leftSide, player, slot));

		const rightItemPickers = [
			ItemSlot.ItemSlotHands,
			ItemSlot.ItemSlotWaist,
			ItemSlot.ItemSlotLegs,
			ItemSlot.ItemSlotFeet,
			ItemSlot.ItemSlotFinger1,
			ItemSlot.ItemSlotFinger2,
			ItemSlot.ItemSlotTrinket1,
			ItemSlot.ItemSlotTrinket2,
		].map(slot => new ItemPicker(rightSide, player, slot));

		this.itemPickers = leftItemPickers.concat(rightItemPickers).sort((a, b) => a.slot - b.slot);
	}
}

class ItemPicker extends Component {
	readonly slot: ItemSlot;

	private readonly player: Player<any>;
	private readonly iconElem: HTMLAnchorElement;
	private readonly nameElem: HTMLAnchorElement;
	private readonly enchantElem: HTMLAnchorElement;
	private readonly socketsContainerElem: HTMLElement;

	// All items and enchants that are eligible for this slot
	private _items: Array<Item> = [];
	private _enchants: Array<Enchant> = [];

	private _equippedItem: EquippedItem | null = null;


	constructor(parent: HTMLElement, player: Player<any>, slot: ItemSlot) {
		super(parent, 'item-picker-root');
		this.slot = slot;
		this.player = player;

		this.rootElem.innerHTML = `
      <a class="item-picker-icon">
        <div class="item-picker-sockets-container">
        </div>
      </a>
      <div class="item-picker-labels-container">
        <a class="item-picker-name"></a><br>
        <a class="item-picker-enchant"></a>
      </div>
    `;

		this.iconElem = this.rootElem.getElementsByClassName('item-picker-icon')[0] as HTMLAnchorElement;
		this.nameElem = this.rootElem.getElementsByClassName('item-picker-name')[0] as HTMLAnchorElement;
		this.enchantElem = this.rootElem.getElementsByClassName('item-picker-enchant')[0] as HTMLAnchorElement;
		this.socketsContainerElem = this.rootElem.getElementsByClassName('item-picker-sockets-container')[0] as HTMLElement;

		this.item = player.getEquippedItem(slot);
		player.sim.waitForInit().then(() => {
			this._items = this.player.getItems(this.slot);
			this._enchants = this.player.getEnchants(this.slot);

			const onClickStart = (event: Event) => {
				event.preventDefault();
				const selectorModal = new SelectorModal(this.rootElem.closest('.individual-sim-ui')!, this.player, this.slot, this._equippedItem, this._items, this._enchants);
			};
			const onClickEnd = (event: Event) => {
				event.preventDefault();
			};
			this.iconElem.addEventListener('click', onClickStart);
			this.iconElem.addEventListener('touchstart', onClickStart);
			this.iconElem.addEventListener('touchend', onClickEnd);
			this.nameElem.addEventListener('click', onClickStart);
			this.nameElem.addEventListener('touchstart', onClickStart);
			this.nameElem.addEventListener('touchend', onClickEnd);

			// Make enchant name open enchant tab.
			this.enchantElem.addEventListener('click', (ev: Event) => {
				ev.preventDefault();
				const selectorModal = new SelectorModal(this.rootElem.closest('.individual-sim-ui')!, this.player, this.slot, this._equippedItem, this._items, this._enchants);
				selectorModal.openTab(1);
			});
			this.enchantElem.addEventListener('touchstart', (ev: Event) => {
				ev.preventDefault();
				const selectorModal = new SelectorModal(this.rootElem.closest('.individual-sim-ui')!, this.player, this.slot, this._equippedItem, this._items, this._enchants);
				selectorModal.openTab(1);
			});
			this.enchantElem.addEventListener('touchend', onClickEnd);
		});
		player.gearChangeEmitter.on(() => {
			this.item = player.getEquippedItem(slot);
		});
		player.professionChangeEmitter.on(() => {
			if (this._equippedItem != null) {
				this.player.setWowheadData(this._equippedItem, this.iconElem);
			}
		});

		// Use hacky wowhead xhr override to 'preprocess' tooltips
		WowSim.WhOnLoadHook = (a:any) => {
			if (a.tooltip) {
				// This fixes wowhead being able to parse 'pcs' aka set bonus highlighting in tooltip
				// Their internal regex looks for 'href="/item=' but for wotlk we get 'href="/wotlk/item="'
				a.tooltip = (<String>a.tooltip).replaceAll("href=\"/wotlk/item", "href=\"/item");
			}
			return a;
		}
	}

	set item(newItem: EquippedItem | null) {
		// Clear everything first
		this.nameElem.removeAttribute('data-wowhead');
		this.nameElem.removeAttribute('href');
		this.iconElem.style.backgroundImage = `url('${getEmptySlotIconUrl(this.slot)}')`;
		this.iconElem.removeAttribute('data-wowhead');
		this.iconElem.removeAttribute('href');
		this.enchantElem.removeAttribute('data-wowhead');

		this.nameElem.textContent = slotNames[this.slot];
		setItemQualityCssClass(this.nameElem, null);

		this.enchantElem.innerHTML = '';
		this.socketsContainerElem.innerHTML = '';

		if (newItem != null) {
			this.nameElem.textContent = newItem.item.name;
			if (newItem.item.heroic) {
				var heroic_span = document.createElement('span');
				heroic_span.style.color = "green";
				heroic_span.style.marginLeft = "3px";
				heroic_span.innerText = "[H]";
				this.nameElem.appendChild(heroic_span);
			}

			setItemQualityCssClass(this.nameElem, newItem.item.quality);

			this.player.setWowheadData(newItem, this.iconElem);
			this.player.setWowheadData(newItem, this.nameElem);
			newItem.asActionId().fill().then(filledId => {
				filledId.setBackgroundAndHref(this.iconElem);
				filledId.setWowheadHref(this.nameElem);
			});

			if (newItem.enchant) {
				getEnchantDescription(newItem.enchant).then(description => {
					this.enchantElem.textContent = description;
				});
				// Make enchant text hover have a tooltip.
				if (newItem.enchant.spellId) {
					this.enchantElem.setAttribute('data-wowhead', `domain=wotlk&spell=${newItem.enchant.spellId}`);
				} else {
					this.enchantElem.setAttribute('data-wowhead', `domain=wotlk&item=${newItem.enchant.itemId}`);
				}
			}

			newItem.allSocketColors().forEach((socketColor, gemIdx) => {
				const gemIconElem = document.createElement('img');
				gemIconElem.classList.add('item-picker-gem-icon');
				setGemSocketCssClass(gemIconElem, socketColor);
				if (newItem.gems[gemIdx] == null) {
					gemIconElem.src = getEmptyGemSocketIconUrl(socketColor);
				} else {
					ActionId.fromItemId(newItem.gems[gemIdx]!.id).fill().then(filledId => {
						gemIconElem.src = filledId.iconUrl;
					});
				}
				this.socketsContainerElem.appendChild(gemIconElem);

				if (gemIdx == newItem.numPossibleSockets - 1 && [ItemType.ItemTypeWrist, ItemType.ItemTypeHands].includes(newItem.item.type)) {
					const updateProfession = () => {
						if (this.player.isBlacksmithing()) {
							gemIconElem.style.removeProperty('display');
						} else {
							gemIconElem.style.display = 'none';
						}
					};
					this.player.professionChangeEmitter.on(updateProfession);
					updateProfession();
				}
			});
		}
		this._equippedItem = newItem;
	}
}

class SelectorModal extends Popup {
	private player: Player<any>;
	private readonly tabsElem: HTMLElement;
	private readonly contentElem: HTMLElement;

	constructor(parent: HTMLElement, player: Player<any>, slot: ItemSlot, equippedItem: EquippedItem | null, eligibleItems: Array<Item>, eligibleEnchants: Array<Enchant>) {
		super(parent);
		this.player = player;

		this.rootElem.classList.add('selector-modal');
		this.rootElem.innerHTML = `
			<ul class="nav nav-tabs selector-modal-tabs">
			</ul>
			<div class="tab-content selector-modal-tab-content">
			</div>
		`;

		this.addCloseButton();
		this.tabsElem = this.rootElem.getElementsByClassName('selector-modal-tabs')[0] as HTMLElement;
		this.contentElem = this.rootElem.getElementsByClassName('selector-modal-tab-content')[0] as HTMLElement;

		this.setData(slot, equippedItem, eligibleItems, eligibleEnchants);
	}

	openTab(idx: number) {
		const elems = this.tabsElem.getElementsByClassName("selector-modal-item-tab");
		(elems[idx] as HTMLElement).click();
	}

	setData(slot: ItemSlot, equippedItem: EquippedItem | null, eligibleItems: Array<Item>, eligibleEnchants: Array<Enchant>) {
		this.tabsElem.innerHTML = '';
		this.contentElem.innerHTML = '';

		this.addTab(
			'Items',
			slot,
			equippedItem,
			eligibleItems.map(item => {
				return {
					item: item,
					id: item.id,
					actionId: ActionId.fromItem(item),
					name: item.name,
					quality: item.quality,
					heroic: item.heroic,
					phase: item.phase,
					baseEP: this.player.computeItemEP(item, slot),
					ignoreEPFilter: false,
					onEquip: (eventID, item: Item) => {
						const equippedItem = this.player.getEquippedItem(slot);
						if (equippedItem) {
							this.player.equipItem(eventID, slot, equippedItem.withItem(item));
						} else {
							this.player.equipItem(eventID, slot, new EquippedItem(item));
						}
					},
				};
			}),
			item => this.player.computeItemEP(item, slot),
			equippedItem => equippedItem?.item,
			GemColor.GemColorUnknown,
			eventID => {
				this.player.equipItem(eventID, slot, null);
			});

		this.addTab(
			'Enchants',
			slot,
			equippedItem,
			eligibleEnchants.map(enchant => {
				return {
					item: enchant,
					id: enchant.effectId,
					actionId: enchant.spellId ? ActionId.fromSpellId(enchant.spellId) : ActionId.fromItemId(enchant.itemId),
					name: enchant.name,
					quality: enchant.quality,
					phase: enchant.phase || 1,
					baseEP: this.player.computeStatsEP(new Stats(enchant.stats)),
					ignoreEPFilter: true,
					heroic: false,
					onEquip: (eventID, enchant: Enchant) => {
						const equippedItem = this.player.getEquippedItem(slot);
						if (equippedItem)
							this.player.equipItem(eventID, slot, equippedItem.withEnchant(enchant));
					},
				};
			}),
			enchant => this.player.computeEnchantEP(enchant),
			equippedItem => equippedItem?.enchant,
			GemColor.GemColorUnknown,
			eventID => {
				const equippedItem = this.player.getEquippedItem(slot);
				if (equippedItem)
					this.player.equipItem(eventID, slot, equippedItem.withEnchant(null));
			});

		this.addGemTabs(slot, equippedItem);
	}

	private addGemTabs(slot: ItemSlot, equippedItem: EquippedItem | null) {
		if (equippedItem == undefined) {
			return;
		}

		const socketBonusEP = this.player.computeStatsEP(new Stats(equippedItem.item.socketBonus)) / (equippedItem.item.gemSockets.length || 1);
		equippedItem.curSocketColors(this.player.isBlacksmithing()).forEach((socketColor, socketIdx) => {
			this.addTab(
				'Gem ' + (socketIdx + 1),
				slot,
				equippedItem,
				this.player.getGems(socketColor).map((gem: Gem) => {
					return {
						item: gem,
						id: gem.id,
						actionId: ActionId.fromItemId(gem.id),
						name: gem.name,
						quality: gem.quality,
						phase: gem.phase,
						heroic: false,
						baseEP: this.player.computeStatsEP(new Stats(gem.stats)),
						ignoreEPFilter: true,
						onEquip: (eventID, gem: Gem) => {
							const equippedItem = this.player.getEquippedItem(slot);
							if (equippedItem)
								this.player.equipItem(eventID, slot, equippedItem.withGem(gem, socketIdx));
						},
					};
				}),
				gem => {
					let gemEP = this.player.computeGemEP(gem);
					if (gemMatchesSocket(gem, socketColor)) {
						gemEP += socketBonusEP;
					}
					return gemEP;
				},
				equippedItem => equippedItem?.gems[socketIdx],
				socketColor,
				eventID => {
					const equippedItem = this.player.getEquippedItem(slot);
					if (equippedItem)
						this.player.equipItem(eventID, slot, equippedItem.withGem(null, socketIdx));
				},
				tabAnchor => {
					tabAnchor.classList.add('selector-modal-tab-gem-icon');
					setGemSocketCssClass(tabAnchor, socketColor);

					const updateGemIcon = () => {
						const equippedItem = this.player.getEquippedItem(slot);
						const gem = equippedItem?.gems[socketIdx];

						if (gem) {
							ActionId.fromItemId(gem.id).fill().then(filledId => {
								tabAnchor.style.backgroundImage = `url('${filledId.iconUrl}')`;
							});
						} else {
							const url = getEmptyGemSocketIconUrl(socketColor);
							tabAnchor.style.backgroundImage = `url('${url}')`;
						}
					};

					this.player.gearChangeEmitter.on(updateGemIcon);
					this.addOnDisposeCallback(() => this.player.gearChangeEmitter.off(updateGemIcon));
					updateGemIcon();
				});
		});
	}

	/**
	 * Adds one of the tabs for the item selector menu.
	 *
	 * T is expected to be Item, Enchant, or Gem. Tab menus for all 3 looks extremely
	 * similar so this function uses extra functions to do it generically.
	 */
	private addTab<T>(
		label: string,
		slot: ItemSlot,
		equippedItem: EquippedItem | null,
		itemData: Array<ItemData<T>>,
		computeEP: (item: T) => number,
		equippedToItemFn: (equippedItem: EquippedItem | null) => (T | null | undefined),
		socketColor: GemColor,
		onRemove: (eventID: EventID) => void,
		setTabContent?: (tabElem: HTMLAnchorElement) => void) {
		if (itemData.length == 0) {
			return;
		}

		if (slot == ItemSlot.ItemSlotTrinket1 || slot == ItemSlot.ItemSlotTrinket2) {
			// Trinket EP is weird so just sort by ilvl instead.
			itemData.sort((dataA, dataB) => (dataB.item as unknown as Item).ilvl - (dataA.item as unknown as Item).ilvl);
		} else {
			itemData.sort((dataA, dataB) => {
				const diff = computeEP(dataB.item) - computeEP(dataA.item);
				// if EP is same, sort by ilvl
				if (Math.abs(diff) < 0.01) {
					return (dataB.item as unknown as Item).ilvl - (dataA.item as unknown as Item).ilvl;
				}
				return diff;
			});
		}

		const tabContentId = (label + '-tab').split(' ').join('');
		const selected = label === 'Items';

		const tabFragment = document.createElement('fragment');
		tabFragment.innerHTML = `
			<li class="nav-item">
				<a
					class="nav-link selector-modal-item-tab ${selected ? 'active' : ''}"
					data-content-id="${tabContentId}"
					data-bs-toggle="tab"
					data-bs-target="#${tabContentId}"
					type="button"
					role="tab"
					aria-controls="${tabContentId}"
					aria-selected="${selected}"
				></a>
			</li>
		`;

		const tabElem = tabFragment.children[0] as HTMLElement;
		const tabAnchor = tabElem.getElementsByClassName('selector-modal-item-tab')[0] as HTMLAnchorElement;
		tabAnchor.dataset.label = label;
		if (setTabContent) {
			setTabContent(tabAnchor);
		} else {
			tabAnchor.textContent = label;
		}

		this.tabsElem.appendChild(tabElem);

		const tabContentFragment = document.createElement('fragment');
		tabContentFragment.innerHTML = `
			<div
				id="${tabContentId}"
				class="selector-modal-tab-pane tab-pane fade ${selected ? 'active show' : ''}"
			>
				<div class="selector-modal-tab-content-header">
					<input class="selector-modal-search form-control" type="text" placeholder="Search...">
					<button class="selector-modal-filters-button btn btn-primary">Filters</button>
					<div class="sim-input selector-modal-boolean-option selector-modal-show-1h-weapons"></div>
					<div class="sim-input selector-modal-boolean-option selector-modal-show-2h-weapons"></div>
					<div class="sim-input selector-modal-boolean-option selector-modal-show-matching-gems"></div>
					<div class="selector-modal-phase-selector"></div>
					<button class="selector-modal-remove-button btn btn-danger">Unequip Item</button>
				</div>
				<div style="width: 100%;height: 30px;font-size: 18px;">
					<span style="float:left">Item</span>
					<span style="float:right">EP(+/-)<span class="ep-help fas fa-search" style="font-size:10px"></span></span>
				</div>
				<ul class="selector-modal-list"></ul>
			</div>
		`;
		
		const tabContent = tabContentFragment.children[0] as HTMLElement;

		this.contentElem.appendChild(tabContent);

		const helpIcon = tabContent.getElementsByClassName("ep-help").item(0);
		tippy(helpIcon, {'content': 'These values are computed using stat weights which can be edited using the "Stat Weights" button.'});
		const show1hWeaponsSelector = makeShow1hWeaponsSelector(tabContent.getElementsByClassName('selector-modal-show-1h-weapons')[0] as HTMLElement, this.player.sim);
		const show2hWeaponsSelector = makeShow2hWeaponsSelector(tabContent.getElementsByClassName('selector-modal-show-2h-weapons')[0] as HTMLElement, this.player.sim);
		if (!(label == 'Items' && (slot == ItemSlot.ItemSlotMainHand || (slot == ItemSlot.ItemSlotOffHand && this.player.getClass() == Class.ClassWarrior)))) {
			(tabContent.getElementsByClassName('selector-modal-show-1h-weapons')[0] as HTMLElement).style.display = 'none';
			(tabContent.getElementsByClassName('selector-modal-show-2h-weapons')[0] as HTMLElement).style.display = 'none';
		}

		const showMatchingGemsSelector = makeShowMatchingGemsSelector(tabContent.getElementsByClassName('selector-modal-show-matching-gems')[0] as HTMLElement, this.player.sim);
		if (!label.startsWith('Gem')) {
			(tabContent.getElementsByClassName('selector-modal-show-matching-gems')[0] as HTMLElement).style.display = 'none';
		}

		const phaseSelector = makePhaseSelector(tabContent.getElementsByClassName('selector-modal-phase-selector')[0] as HTMLElement, this.player.sim);

		const filtersButton = tabContent.getElementsByClassName('selector-modal-filters-button')[0] as HTMLElement;
		if (FiltersMenu.anyFiltersForSlot(slot)) {
			filtersButton.addEventListener('click', () => new FiltersMenu(this.rootElem, this.player, slot));
		} else {
			filtersButton.style.display = 'none';
		}

		if (label == 'Items') {
			tabElem.classList.add('active', 'in');
			tabContent.classList.add('active', 'in');
		}

		const listElem = tabContent.getElementsByClassName('selector-modal-list')[0] as HTMLElement;
		const initialFilters = this.player.sim.getFilters();
		let lastFavElem: HTMLElement|null = null;

		const listItemElems = itemData.map((itemData, itemIdx) => {
			const item = itemData.item;
			const itemEP = computeEP(item);

			const listItemElem = document.createElement('li');
			listItemElem.classList.add('selector-modal-list-item');
			listElem.appendChild(listItemElem);

			listItemElem.dataset.idx = String(itemIdx);

			listItemElem.innerHTML = `
				<div class="selector-modal-list-label-cell">
					<a class="selector-modal-list-item-icon"></a>
					<a class="selector-modal-list-item-name">${itemData.heroic ? itemData.name + "<span style=\"color:green\">[H]</span>" : itemData.name}</a>
				</div>
				<div>
					<span class="selector-modal-list-item-favorite fa-star"></span>
				</div>
				<div class="selector-modal-list-item-ep">
					<span class="selector-modal-list-item-ep-value">${itemEP < 9.95 ? itemEP.toFixed(1) : Math.round(itemEP)}</span>
				</div>
				<div class="selector-modal-list-item-ep">
					<span class="selector-modal-list-item-ep-delta"></span>
				</div>
      `;

			if (slot == ItemSlot.ItemSlotTrinket1 || slot == ItemSlot.ItemSlotTrinket2) {
				const epElem = listItemElem.getElementsByClassName('selector-modal-list-item-ep')[0] as HTMLElement;
				epElem.style.display = 'none';
			}

			const iconElem = listItemElem.getElementsByClassName('selector-modal-list-item-icon')[0] as HTMLAnchorElement;
			const nameElem = listItemElem.getElementsByClassName('selector-modal-list-item-name')[0] as HTMLAnchorElement;
			itemData.actionId.fill().then(filledId => {
				filledId.setWowheadHref(iconElem);
				filledId.setWowheadHref(nameElem);
				iconElem.style.backgroundImage = `url('${filledId.iconUrl}')`;
			});

			setItemQualityCssClass(nameElem, itemData.quality);

			const onclick = (event: Event) => {
				event.preventDefault();
				itemData.onEquip(TypedEvent.nextEventID(), item);

				// If the item changes, the gem slots might change, so remove and recreate the gem tabs
				if (Item.is(item)) {
					this.removeTabs('Gem');
					this.addGemTabs(slot, this.player.getEquippedItem(slot));
				}
			};
			nameElem.addEventListener('click', onclick);
			iconElem.addEventListener('click', onclick);

			const favoriteElem = listItemElem.getElementsByClassName('selector-modal-list-item-favorite')[0] as HTMLElement;
			tippy(favoriteElem, {'content': 'Add to Favorites'});
			const setFavorite = (isFavorite: boolean) => {
				const filters = this.player.sim.getFilters();
				if (label == 'Items') {
					const favId = itemData.id;
					if (isFavorite) {
						filters.favoriteItems.push(favId);
					} else {
						const favIdx = filters.favoriteItems.indexOf(favId);
						if (favIdx != -1) {
							filters.favoriteItems.splice(favIdx, 1);
						}
					}
				} else if (label == 'Enchants') {
					const favId = getUniqueEnchantString(item as unknown as Enchant);
					if (isFavorite) {
						filters.favoriteEnchants.push(favId);
					} else {
						const favIdx = filters.favoriteEnchants.indexOf(favId);
						if (favIdx != -1) {
							filters.favoriteEnchants.splice(favIdx, 1);
						}
					}
				} else if (label.startsWith('Gem')) {
					const favId = itemData.id;
					if (isFavorite) {
						filters.favoriteGems.push(favId);
					} else {
						const favIdx = filters.favoriteGems.indexOf(favId);
						if (favIdx != -1) {
							filters.favoriteGems.splice(favIdx, 1);
						}
					}
				}
				this.player.sim.setFilters(TypedEvent.nextEventID(), filters);

				// Reorder and update this element.
				const curItemElems = Array.from(listElem.children) as Array<HTMLElement>;
				if (isFavorite) {
					// Use same sorting order (based on idx) among the favorited elems.
					const nextElem = curItemElems.find(elem => elem.dataset.fav == 'false' || parseInt(elem.dataset.idx!) > itemIdx);
					if (nextElem) {
						listElem.insertBefore(listItemElem, nextElem);
					} else {
						listElem.appendChild(listItemElem);
					}

					favoriteElem.classList.add('fa-solid');
					favoriteElem.classList.remove('fa-regular');
					listItemElem.dataset.fav = 'true';
				} else {
					// Put back in original spot. itemIdx will usually be a very good starting point for the search.
					// Need to search in both directions to handle all cases of favorited elems / itemIdx location.
					let curIdx = itemIdx;
					while (curIdx > 0 && curItemElems[curIdx].dataset.fav == 'false' && parseInt(curItemElems[curIdx].dataset.idx!) > itemIdx) {
						curIdx--;
					}
					while (curIdx < curItemElems.length && (curItemElems[curIdx].dataset.fav == 'true' || parseInt(curItemElems[curIdx].dataset.idx!) < itemIdx)) {
						curIdx++;
					}
					if (curIdx == curItemElems.length) {
						listElem.appendChild(listItemElem);
					} else {
						listElem.insertBefore(listItemElem, curItemElems[curIdx]);
					}

					favoriteElem.classList.remove('fa-solid');
					favoriteElem.classList.add('fa-regular');
					listItemElem.dataset.fav = 'false';
				}
			};
			favoriteElem.addEventListener('click', () => setFavorite(listItemElem.dataset.fav == 'false'));

			let isFavorite = false;
			if (label == 'Items') {
				isFavorite = initialFilters.favoriteItems.includes(itemData.id);
			} else if (label == 'Enchants') {
				isFavorite = initialFilters.favoriteEnchants.includes(getUniqueEnchantString(item as unknown as Enchant));
			} else if (label.startsWith('Gem')) {
				isFavorite = initialFilters.favoriteGems.includes(itemData.id);
			}
			if (isFavorite) {
				favoriteElem.classList.add('fa-solid');
				listItemElem.dataset.fav = 'true';
				if (lastFavElem == null) {
					listElem.prepend(listItemElem);
				} else {
					lastFavElem.after(listItemElem)
				}
				lastFavElem = listItemElem;
			} else {
				favoriteElem.classList.add('fa-regular');
				listItemElem.dataset.fav = 'false';
			}

			return listItemElem;
		});

		const removeButton = tabContent.getElementsByClassName('selector-modal-remove-button')[0] as HTMLButtonElement;
		removeButton.addEventListener('click', event => {
			listItemElems.forEach(elem => elem.classList.remove('active'));
			onRemove(TypedEvent.nextEventID());
		});

		const updateSelected = () => {
			const newEquippedItem = this.player.getEquippedItem(slot);
			const newItem = equippedToItemFn(newEquippedItem);

			const newItemId = newItem ? (label == 'Enchants' ? (newItem as unknown as Enchant).effectId : (newItem as unknown as Item|Gem).id) : 0;
			const newEP = newItem ? computeEP(newItem) : 0;

			listItemElems.forEach(elem => {
				const listItemIdx = parseInt(elem.dataset.idx!);
				const listItemData = itemData[listItemIdx];
				const listItem = listItemData.item;

				elem.classList.remove('active');
				if (listItemData.id == newItemId) {
					elem.classList.add('active');
				}

				const epDeltaElem = elem.getElementsByClassName('selector-modal-list-item-ep-delta')[0] as HTMLSpanElement;
				epDeltaElem.textContent = '';
				if (listItem) {
					const listItemEP = computeEP(listItem);
					formatDeltaTextElem(epDeltaElem, newEP, listItemEP, 0);
				}
			});
		};
		this.player.gearChangeEmitter.on(updateSelected);
		this.addOnDisposeCallback(() => this.player.gearChangeEmitter.off(updateSelected));
		updateSelected();

		const applyFilters = () => {
			let validItemElems = listItemElems;
			const currentEquippedItem = this.player.getEquippedItem(slot);

			if (label == 'Items') {
				validItemElems = this.player.filterItemData(
						validItemElems,
						elem => itemData[parseInt(elem.dataset.idx!)].item as unknown as Item,
						slot);
			} else if (label == 'Enchants') {
				validItemElems = this.player.filterEnchantData(
						validItemElems,
						elem => itemData[parseInt(elem.dataset.idx!)].item as unknown as Enchant,
						slot,
						currentEquippedItem);
			} else if (label.startsWith('Gem')) {
				validItemElems = this.player.filterGemData(
						validItemElems,
						elem => itemData[parseInt(elem.dataset.idx!)].item as unknown as Gem,
						slot,
						socketColor);
			}

			validItemElems = validItemElems.filter(elem => {
				const listItemData = itemData[parseInt(elem.dataset.idx!)];

				if (listItemData.phase > this.player.sim.getPhase()) {
					return false;
				}

				if (searchInput.value.length > 0) {
					const searchQuery = searchInput.value.toLowerCase().split(" ");
					const name = listItemData.name.toLowerCase();

					var include = true;
					searchQuery.forEach(v => {
						if (!name.includes(v))
							include = false;
					});
					if (!include) {
						return false;
					}
				}

				return true;
			});

			let numShown = 0;
			listItemElems.forEach(elem => {
				if (validItemElems.includes(elem)) {
					elem.classList.remove('hidden');
					numShown++;
					if (numShown % 2 == 0) {
						elem.classList.remove('odd');
					} else {
						elem.classList.add('odd');
					}
				} else {
					elem.classList.add('hidden');
				}
			});
		};

		const searchInput = tabContent.getElementsByClassName('selector-modal-search')[0] as HTMLInputElement;
		searchInput.addEventListener('input', applyFilters);
		searchInput.addEventListener("keyup", ev => {
			if (ev.key == "Enter") {
				listItemElems.find(ele => {
					if (ele.classList.contains("hidden")) {
						return false;
					}
					const nameElem = ele.getElementsByClassName('selector-modal-list-item-name')[0] as HTMLElement;
					nameElem.click();
					return true;
				});
			}
		});

		this.player.sim.phaseChangeEmitter.on(applyFilters);
		this.player.sim.filtersChangeEmitter.on(applyFilters);
		this.player.gearChangeEmitter.on(applyFilters);
		this.addOnDisposeCallback(() => {
			this.player.sim.phaseChangeEmitter.off(applyFilters);
			this.player.sim.filtersChangeEmitter.off(applyFilters);
			this.player.gearChangeEmitter.off(applyFilters);
		});

		applyFilters();
	}

	private removeTabs(labelSubstring: string) {
		const tabElems = Array.prototype.slice.call(this.tabsElem.getElementsByClassName('selector-modal-item-tab'))
			.filter(tab => tab.dataset.label.includes(labelSubstring));

		const contentElems = tabElems
			.map(tabElem => document.getElementById(tabElem.dataset.contentId!.substring(1)))
			.filter(tabElem => Boolean(tabElem));

		tabElems.forEach(elem => elem.parentElement.remove());
		contentElems.forEach(elem => elem!.remove());
	}
}

interface ItemData<T> {
	item: T,
	name: string,
	id: number,
	actionId: ActionId,
	quality: ItemQuality,
	phase: number,
	baseEP: number,
	ignoreEPFilter: boolean,
	heroic: boolean,
	onEquip: (eventID: EventID, item: T) => void,
}

const emptySlotIcons: Record<ItemSlot, string> = {
	[ItemSlot.ItemSlotHead]: 'https://cdn.seventyupgrades.com/item-slots/Head.jpg',
	[ItemSlot.ItemSlotNeck]: 'https://cdn.seventyupgrades.com/item-slots/Neck.jpg',
	[ItemSlot.ItemSlotShoulder]: 'https://cdn.seventyupgrades.com/item-slots/Shoulders.jpg',
	[ItemSlot.ItemSlotBack]: 'https://cdn.seventyupgrades.com/item-slots/Back.jpg',
	[ItemSlot.ItemSlotChest]: 'https://cdn.seventyupgrades.com/item-slots/Chest.jpg',
	[ItemSlot.ItemSlotWrist]: 'https://cdn.seventyupgrades.com/item-slots/Wrists.jpg',
	[ItemSlot.ItemSlotHands]: 'https://cdn.seventyupgrades.com/item-slots/Hands.jpg',
	[ItemSlot.ItemSlotWaist]: 'https://cdn.seventyupgrades.com/item-slots/Waist.jpg',
	[ItemSlot.ItemSlotLegs]: 'https://cdn.seventyupgrades.com/item-slots/Legs.jpg',
	[ItemSlot.ItemSlotFeet]: 'https://cdn.seventyupgrades.com/item-slots/Feet.jpg',
	[ItemSlot.ItemSlotFinger1]: 'https://cdn.seventyupgrades.com/item-slots/Finger.jpg',
	[ItemSlot.ItemSlotFinger2]: 'https://cdn.seventyupgrades.com/item-slots/Finger.jpg',
	[ItemSlot.ItemSlotTrinket1]: 'https://cdn.seventyupgrades.com/item-slots/Trinket.jpg',
	[ItemSlot.ItemSlotTrinket2]: 'https://cdn.seventyupgrades.com/item-slots/Trinket.jpg',
	[ItemSlot.ItemSlotMainHand]: 'https://cdn.seventyupgrades.com/item-slots/MainHand.jpg',
	[ItemSlot.ItemSlotOffHand]: 'https://cdn.seventyupgrades.com/item-slots/OffHand.jpg',
	[ItemSlot.ItemSlotRanged]: 'https://cdn.seventyupgrades.com/item-slots/Ranged.jpg',
};
export function getEmptySlotIconUrl(slot: ItemSlot): string {
	return emptySlotIcons[slot];
}
