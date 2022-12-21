import { RaidSimRequest, RaidSimResult } from '../core/proto/api.js';
import { SimResult, SimResultFilter } from '../core/proto_utils/sim_result.js';
import { Component } from '../core/components/component.js';
import { EventID, TypedEvent } from '../core/typed_event.js';

import { ColorSettings } from './color_settings.js';

export interface SimResultData {
	eventID: EventID,
	result: SimResult,
	filter: SimResultFilter,
};

export interface ResultComponentConfig {
	parent: HTMLElement,
	rootCssClass?: string,
	cssScheme?: String | null,
	resultsEmitter: TypedEvent<SimResultData | null>,
	colorSettings: ColorSettings,
};

export abstract class ResultComponent extends Component {
	private readonly colorSettings: ColorSettings;

	private lastSimResult: SimResultData | null;

	constructor(config: ResultComponentConfig) {
		super(config.parent, config.rootCssClass || '');
		this.colorSettings = config.colorSettings;
		this.lastSimResult = null;

		config.resultsEmitter.on((eventID, resultData) => {
			if (!resultData)
				return;

			this.lastSimResult = resultData;
			this.onSimResult(resultData);
		});
	}

	getLastSimResult(): SimResultData {
		if (this.lastSimResult) {
			return this.lastSimResult;
		} else {
			throw new Error('No last sim result!');
		}
	}

	abstract onSimResult(resultData: SimResultData): void;
}
