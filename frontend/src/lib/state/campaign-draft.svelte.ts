/**
 * Черновик новой кампании.
 *
 * Значения формы живут в одном `$state`-объекте и НЕ persist-ятся: устаревший
 * черновик, молча создающий кампанию после возвращения во вкладку, хуже
 * повторного ввода. Шаг живёт в адресной строке маршрута, поэтому здесь только
 * значения и операции над ними.
 *
 * Смена модели сбрасывает выбранную версию: версия чужой модели в форме
 * появиться не может ни через интерфейс, ни через deep link.
 */
import type { CreateCampaignInput } from "$lib/api/types";
import {
	addRow,
	buildStages,
	canaryPresetRows,
	createStageRow,
	duplicateRow,
	moveRow,
	removeRow,
	singlePresetRows,
	type StageDraftRow,
} from "$lib/domain/stage-draft";

class CampaignDraft {
	#model = $state("");
	#firmwareVersionId = $state("");
	rows = $state<StageDraftRow[]>([createStageRow()]);
	/** Deep link `?firmware=` указал на неизвестную или устаревшую запись. */
	firmwareNotice = $state<string | undefined>(undefined);

	get model(): string {
		return this.#model;
	}

	set model(next: string) {
		if (next === this.#model) return;
		this.#model = next;
		this.#firmwareVersionId = "";
		this.firmwareNotice = undefined;
	}

	get firmwareVersionId(): string {
		return this.#firmwareVersionId;
	}

	set firmwareVersionId(next: string) {
		this.#firmwareVersionId = next;
	}

	get step1Complete(): boolean {
		return this.#model.length > 0 && this.#firmwareVersionId.length > 0;
	}

	/** Выбор прошивки парой: deep link, пресет или результат регистрации. */
	setFirmware(model: string, firmwareVersionId: string): void {
		this.#model = model;
		this.#firmwareVersionId = firmwareVersionId;
		this.firmwareNotice = undefined;
	}

	/** Регистрация прошивки из мастера: новая запись сразу выбрана. */
	applyRegistered(model: string, firmwareVersionId: string): void {
		this.setFirmware(model, firmwareVersionId);
	}

	addStage(): void {
		this.rows = addRow(this.rows);
	}

	removeStage(index: number): void {
		this.rows = removeRow(this.rows, index);
	}

	duplicateStage(index: number): void {
		this.rows = duplicateRow(this.rows, index);
	}

	moveStage(index: number, delta: -1 | 1): void {
		this.rows = moveRow(this.rows, index, delta);
	}

	applyCanaryPreset(): void {
		this.rows = canaryPresetRows();
	}

	applySinglePreset(): void {
		this.rows = singlePresetRows();
	}

	/** «Очистить» в шапке мастера: значения сбрасываются, шаг остаётся первым. */
	clear(): void {
		this.#model = "";
		this.#firmwareVersionId = "";
		this.rows = [createStageRow()];
		this.firmwareNotice = undefined;
	}

	payload(): CreateCampaignInput {
		return {
			firmware_version_id: this.#firmwareVersionId,
			rollout_stages: buildStages(this.rows),
		};
	}
}

export const campaignDraft = new CampaignDraft();
