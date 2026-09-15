/**
 * Черновик стадий будущей кампании.
 *
 * Вся логика формы чистая и тестируемая: строки черновика, перевод процентов
 * в доли и обратно, перенумерация `order_index`, жёсткие ошибки полей
 * (зеркало серверных правил из openapi.yaml) и предупреждение об убывании
 * охвата, которое серверу не мешает, но оператору важно (§4.6).
 *
 * `order_index` в черновике не хранится: порядок задаётся позицией строки в
 * массиве, поэтому удаление и перестановка строк не могут рассинхронизировать
 * номера. Тесты проверяют именно это.
 */
import type { CreateStageInput } from "$lib/api/types";

/** Пределы строк повторяющегося редактора: зеркало `minItems`/`maxItems` схемы. */
export const STAGE_ROW_MIN = 1;
export const STAGE_ROW_MAX = 20;

export interface StageDraftRow {
	/** Служебный ключ строки для `{#each}`; в запрос не попадает. */
	key: string;
	/** Охват стадии, 1..100 целыми процентами. */
	targetPercent: number | undefined;
	/** Минимальный размер выборки, целое >= 1. */
	minSampleSize: number | undefined;
	/** Порог успеха в процентах, 0 < x <= 100, шаг 0.5; отправляется долей. */
	thresholdPercent: number | undefined;
}

export type StageField = "targetPercent" | "minSampleSize" | "thresholdPercent";

let rowCounter = 0;

/** Новая пустая строка редактора. Ключ монотонный: повторных значений не бывает. */
export function createStageRow(): StageDraftRow {
	rowCounter += 1;
	return {
		key: `stage-row-${rowCounter}`,
		targetPercent: undefined,
		minSampleSize: undefined,
		thresholdPercent: undefined,
	};
}

/**
 * Проценты в долю контракта: 95 -> 0.95, 99.5 -> 0.995.
 * Округление через целое защищает от хвостов двоичной арифметики:
 * сервер не принимает значения вроде 1.0000001.
 */
export function percentToFraction(percent: number): number {
	return Math.round(percent * 100) / 10_000;
}

/** Доля в проценты интерфейса: 0.95 -> 95, 0.995 -> 99.5. */
export function fractionToPercent(fraction: number): number {
	return Math.round(fraction * 10_000) / 100;
}

function isInteger(value: number | undefined): value is number {
	return value !== undefined && Number.isFinite(value) && Number.isInteger(value);
}

/**
 * Жёсткие ошибки строки: зеркало правил схемы запроса создания кампании.
 * Сообщения сознательно без номера стадии и имени поля: ошибка рисуется под
 * своим вводом внутри карточки стадии, контекст виден и без префикса.
 */
export function validateStageRow(
	row: StageDraftRow,
	index: number,
): Partial<Record<StageField, string>> {
	void index;
	const errors: Partial<Record<StageField, string>> = {};

	if (row.targetPercent === undefined || Number.isNaN(row.targetPercent)) {
		errors.targetPercent = "Укажите число от 1 до 100";
	} else if (!isInteger(row.targetPercent)) {
		errors.targetPercent = "Укажите целое число от 1 до 100";
	} else if (row.targetPercent < 1 || row.targetPercent > 100) {
		errors.targetPercent = "Укажите число от 1 до 100";
	}

	if (row.minSampleSize === undefined || Number.isNaN(row.minSampleSize)) {
		errors.minSampleSize = "Укажите число не меньше 1";
	} else if (!isInteger(row.minSampleSize)) {
		errors.minSampleSize = "Укажите целое число не меньше 1";
	} else if (row.minSampleSize < 1) {
		errors.minSampleSize = "Укажите число не меньше 1";
	}

	if (row.thresholdPercent === undefined || Number.isNaN(row.thresholdPercent)) {
		errors.thresholdPercent = "Укажите число больше 0 и не больше 100";
	} else if (row.thresholdPercent <= 0 || row.thresholdPercent > 100) {
		errors.thresholdPercent = "Укажите число больше 0 и не больше 100";
	}

	return errors;
}

/** Ошибки всех строк; пустой объект на строку означает, что строка валидна. */
export function validateStageRows(rows: StageDraftRow[]): Partial<Record<StageField, string>>[] {
	return rows.map((row, index) => validateStageRow(row, index));
}

export function isRowsValid(rows: StageDraftRow[]): boolean {
	if (rows.length < STAGE_ROW_MIN || rows.length > STAGE_ROW_MAX) return false;
	return validateStageRows(rows).every((errors) => Object.keys(errors).length === 0);
}

/**
 * Предупреждение об убывании охвата: API такое принимает, но устройства,
 * получившие обновление на ранней стадии, выпадут из более узкой следующей.
 * Это предупреждение, а не ошибка: блокируют только жёсткие правила схемы.
 */
export function coverageWarnings(rows: StageDraftRow[]): string[] {
	const warnings: string[] = [];

	for (let index = 1; index < rows.length; index += 1) {
		const current = rows[index].targetPercent;
		const previous = rows[index - 1].targetPercent;
		if (current === undefined || previous === undefined) continue;
		if (current >= previous) continue;
		warnings.push(
			`Охват стадии ${index + 1} (${current}%) меньше охвата стадии ${index} (${previous}%): устройства, получившие обновление на первой стадии, выпадут из второй.`,
		);
	}

	return warnings;
}

/** Тело запроса стадий: номера по позициям, порог - долей. */
export function buildStages(rows: StageDraftRow[]): CreateStageInput[] {
	return rows.map((row, index) => ({
		order_index: index,
		target_percent: row.targetPercent ?? 0,
		min_sample_size: row.minSampleSize ?? 1,
		success_threshold: percentToFraction(row.thresholdPercent ?? 0),
	}));
}

/* Операции над строками: все возвращают новый массив, порядок и есть номер. */

export function addRow(rows: StageDraftRow[]): StageDraftRow[] {
	if (rows.length >= STAGE_ROW_MAX) return rows;
	return [...rows, createStageRow()];
}

export function removeRow(rows: StageDraftRow[], index: number): StageDraftRow[] {
	if (rows.length <= STAGE_ROW_MIN) return rows;
	return rows.filter((_, position) => position !== index);
}

export function duplicateRow(rows: StageDraftRow[], index: number): StageDraftRow[] {
	if (rows.length >= STAGE_ROW_MAX) return rows;
	const source = rows[index];
	const copy: StageDraftRow = { ...source, key: createStageRow().key };
	return [...rows.slice(0, index + 1), copy, ...rows.slice(index + 1)];
}

export function moveRow(rows: StageDraftRow[], index: number, delta: -1 | 1): StageDraftRow[] {
	const target = index + delta;
	if (target < 0 || target >= rows.length) return rows;
	const next = [...rows];
	[next[index], next[target]] = [next[target], next[index]];
	return next;
}

/* Пресеты: одно нажатие, дальше строки полностью редактируемые. */

function presetRow(
	targetPercent: number,
	minSampleSize: number,
	thresholdPercent: number,
): StageDraftRow {
	return { key: createStageRow().key, targetPercent, minSampleSize, thresholdPercent };
}

/** Канареечная схема 10 → 50 → 100 с выборками 5/20/50 и порогом 95%. */
export function canaryPresetRows(): StageDraftRow[] {
	return [presetRow(10, 5, 95), presetRow(50, 20, 95), presetRow(100, 50, 95)];
}

/** Одна стадия на всю аудиторию: порог и выборка те же, что у канареечной схемы. */
export function singlePresetRows(): StageDraftRow[] {
	return [presetRow(100, 20, 95)];
}
