import { describe, expect, it } from "vitest";
import type { CreateCampaignInput } from "$lib/api/types";
import {
	addRow,
	buildStages,
	canaryPresetRows,
	coverageWarnings,
	createStageRow,
	duplicateRow,
	fractionToPercent,
	isRowsValid,
	moveRow,
	percentToFraction,
	removeRow,
	singlePresetRows,
	STAGE_ROW_MAX,
	validateStageRow,
	validateStageRows,
	type StageDraftRow,
} from "./stage-draft";

function row(target: number, sample: number, threshold: number): StageDraftRow {
	return {
		key: `k-${target}`,
		targetPercent: target,
		minSampleSize: sample,
		thresholdPercent: threshold,
	};
}

describe("проценты и доли", () => {
	it("переводит проценты в долю контракта без хвостов двоичной арифметики", () => {
		expect(percentToFraction(95)).toBe(0.95);
		expect(percentToFraction(100)).toBe(1);
		expect(percentToFraction(99.5)).toBe(0.995);
		expect(percentToFraction(0.5)).toBe(0.005);
		// 1.0000001 в теле запроса недопустим.
		expect(percentToFraction(100)).not.toBeGreaterThan(1);
	});

	it("обратный перевод возвращает проценты интерфейса", () => {
		expect(fractionToPercent(0.95)).toBe(95);
		expect(fractionToPercent(1)).toBe(100);
		expect(fractionToPercent(0.995)).toBe(99.5);
		for (const value of [1, 10, 50, 95, 99.5, 100]) {
			expect(fractionToPercent(percentToFraction(value))).toBe(value);
		}
	});
});

describe("жёсткая валидация строк", () => {
	it("пустые поля получают сообщение с ограничением, без префиксов", () => {
		const errors = validateStageRows([createStageRow()])[0];
		expect(errors.targetPercent).toBe("Укажите число от 1 до 100");
		expect(errors.minSampleSize).toBe("Укажите число не меньше 1");
		expect(errors.thresholdPercent).toBe("Укажите число больше 0 и не больше 100");
	});

	it("target_percent=0 и 101 отклоняются, целое требование ловит дробь", () => {
		const errors = validateStageRows([row(0, 1, 95), row(101, 1, 95), row(10.5, 1, 95)]);
		expect(errors[0].targetPercent).toBe("Укажите число от 1 до 100");
		expect(errors[1].targetPercent).toBe("Укажите число от 1 до 100");
		expect(errors[2].targetPercent).toBe("Укажите целое число от 1 до 100");
	});

	it("min_sample_size=0 отклоняется", () => {
		expect(validateStageRows([row(10, 0, 95)])[0].minSampleSize).toBe("Укажите число не меньше 1");
	});

	it("порог 0 и 100.5 отклоняются, граница 100 допустима", () => {
		const errors = validateStageRows([row(10, 1, 0), row(10, 1, 100.5), row(10, 1, 100)]);
		expect(errors[0].thresholdPercent).toBe("Укажите число больше 0 и не больше 100");
		expect(errors[1].thresholdPercent).toBe("Укажите число больше 0 и не больше 100");
		expect(errors[2].thresholdPercent).toBeUndefined();
	});

	it("isRowsValid учитывает пределы числа строк", () => {
		expect(isRowsValid([])).toBe(false);
		expect(isRowsValid([row(10, 5, 95)])).toBe(true);
		const many = Array.from({ length: STAGE_ROW_MAX + 1 }, (_, index) => ({
			...row(100, 1, 95),
			key: `m-${index}`,
		}));
		expect(isRowsValid(many)).toBe(false);
	});
});

describe("перенумерация order_index", () => {
	it("номера задаются позицией: удаление середины не оставляет дыр", () => {
		const rows = [row(10, 5, 95), row(50, 20, 95), row(100, 50, 95)];
		const afterRemove = removeRow(rows, 1);
		expect(buildStages(afterRemove).map((stage) => stage.order_index)).toEqual([0, 1]);
	});

	it("перестановка и дублирование сохраняют непрерывность номеров", () => {
		const rows = [row(10, 5, 95), row(50, 20, 95), row(100, 50, 95)];
		const moved = moveRow(rows, 2, -1);
		expect(buildStages(moved).map((stage) => stage.order_index)).toEqual([0, 1, 2]);
		expect(buildStages(moved).map((stage) => stage.target_percent)).toEqual([10, 100, 50]);

		const duplicated = duplicateRow(rows, 0);
		expect(duplicated).toHaveLength(4);
		expect(buildStages(duplicated).map((stage) => stage.order_index)).toEqual([0, 1, 2, 3]);
	});

	it("удаление единственной строки и добавление сверх предела запрещены", () => {
		const single = [row(100, 20, 95)];
		expect(removeRow(single, 0)).toHaveLength(1);
		let rows = single;
		while (rows.length < STAGE_ROW_MAX) rows = addRow(rows);
		expect(addRow(rows)).toHaveLength(STAGE_ROW_MAX);
	});
});

describe("предупреждение об убывании охвата", () => {
	it("убывание охвата даёт предупреждение, а не ошибку", () => {
		const rows = [row(50, 20, 95), row(10, 5, 95)];
		expect(isRowsValid(rows)).toBe(true);
		const warnings = coverageWarnings(rows);
		expect(warnings).toHaveLength(1);
		expect(warnings[0]).toContain("Охват стадии 2 (10%) меньше охвата стадии 1 (50%)");
	});

	it("неубывающая схема предупреждений не даёт", () => {
		expect(coverageWarnings(canaryPresetRows())).toEqual([]);
		expect(coverageWarnings(singlePresetRows())).toEqual([]);
	});
});

describe("тело запроса", () => {
	it("содержит ровно ключи схемы и долю порога", () => {
		const payload: CreateCampaignInput = {
			firmware_version_id: "fw",
			rollout_stages: buildStages(canaryPresetRows()),
		};
		expect(Object.keys(payload).sort()).toEqual(["firmware_version_id", "rollout_stages"]);
		expect(payload.rollout_stages[0]).toEqual({
			order_index: 0,
			target_percent: 10,
			min_sample_size: 5,
			success_threshold: 0.95,
		});
	});

	it("пресеты дают полностью заполненные строки", () => {
		for (const presetRow of canaryPresetRows()) {
			expect(validateStageRow(presetRow, 0)).toEqual({});
		}
	});
});
