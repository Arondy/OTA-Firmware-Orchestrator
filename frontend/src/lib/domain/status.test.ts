import { describe, expect, it } from "vitest";
import type { AttemptResult, CampaignStatus, DeviceStatus, StageStatus } from "$lib/api/types";
import {
	CAMPAIGN_STATUS,
	DEVICE_ONLINE_WINDOW_MS,
	DEVICE_STATUS,
	ATTEMPT_RESULT,
	NO_LAST_SEEN_LABEL,
	STAGE_STATUS,
	TONE_CLASSES,
	deviceLiveness,
} from "./status";

const NOW = Date.parse("2026-09-12T12:00:00.000Z");

describe("карты статусов", () => {
	it("покрывают каждое значение перечисления кампании", () => {
		const expected: CampaignStatus[] = ["draft", "running", "paused", "completed", "rolled_back"];
		expect(Object.keys(CAMPAIGN_STATUS).sort()).toEqual([...expected].sort());
	});

	it("покрывают каждое значение перечисления стадии", () => {
		const expected: StageStatus[] = ["pending", "active", "passed", "failed"];
		expect(Object.keys(STAGE_STATUS).sort()).toEqual([...expected].sort());
	});

	it("покрывают каждое значение результата попытки", () => {
		const expected: AttemptResult[] = ["success", "failure", "timeout"];
		expect(Object.keys(ATTEMPT_RESULT).sort()).toEqual([...expected].sort());
	});

	it("покрывают каждое значение статуса устройства", () => {
		const expected: DeviceStatus[] = ["active", "decommissioned"];
		expect(Object.keys(DEVICE_STATUS).sort()).toEqual([...expected].sort());
	});
});

describe("семантика цвета (§8.2)", () => {
	it("янтарь - у всего, что идёт прямо сейчас", () => {
		expect(CAMPAIGN_STATUS.running.tone).toBe("progress");
		expect(STAGE_STATUS.active.tone).toBe("progress");
	});

	it("успешный исход - success", () => {
		expect(CAMPAIGN_STATUS.completed.tone).toBe("success");
		expect(STAGE_STATUS.passed.tone).toBe("success");
		expect(ATTEMPT_RESULT.success.tone).toBe("success");
	});

	it("пауза - held, отказ и ошибка - danger", () => {
		expect(CAMPAIGN_STATUS.paused.tone).toBe("held");
		expect(CAMPAIGN_STATUS.rolled_back.tone).toBe("danger");
		expect(STAGE_STATUS.failed.tone).toBe("danger");
		expect(ATTEMPT_RESULT.failure.tone).toBe("danger");
		expect(ATTEMPT_RESULT.timeout.tone).toBe("danger");
	});

	it("инертное и неизвестное - neutral", () => {
		expect(CAMPAIGN_STATUS.draft.tone).toBe("neutral");
		expect(STAGE_STATUS.pending.tone).toBe("neutral");
		expect(DEVICE_STATUS.decommissioned.tone).toBe("neutral");
	});

	it("пульсируют только состояния, которые меняются без участия оператора", () => {
		expect(CAMPAIGN_STATUS.running.live).toBe(true);
		expect(STAGE_STATUS.active.live).toBe(true);
		expect(CAMPAIGN_STATUS.paused.live).toBeUndefined();
		expect(CAMPAIGN_STATUS.completed.live).toBeUndefined();
	});

	it("подписи взяты из словаря §9", () => {
		expect(CAMPAIGN_STATUS.draft.label).toBe("черновик");
		expect(CAMPAIGN_STATUS.running.label).toBe("выполняется");
		expect(CAMPAIGN_STATUS.paused.label).toBe("на паузе");
		expect(CAMPAIGN_STATUS.completed.label).toBe("завершена");
		expect(CAMPAIGN_STATUS.rolled_back.label).toBe("откачена");
		expect(STAGE_STATUS.pending.label).toBe("ожидает");
		expect(STAGE_STATUS.active.label).toBe("активна");
		expect(STAGE_STATUS.passed.label).toBe("пройдена");
		expect(STAGE_STATUS.failed.label).toBe("провалена");
		expect(ATTEMPT_RESULT.success.label).toBe("успех");
		expect(ATTEMPT_RESULT.failure.label).toBe("ошибка");
		expect(ATTEMPT_RESULT.timeout.label).toBe("таймаут");
		expect(DEVICE_STATUS.active.label).toBe("активно");
		expect(DEVICE_STATUS.decommissioned.label).toBe("выведено из эксплуатации");
	});

	it("у каждого статуса есть иконка: цвет никогда не единственный носитель смысла", () => {
		const metas = [
			...Object.values(CAMPAIGN_STATUS),
			...Object.values(STAGE_STATUS),
			...Object.values(ATTEMPT_RESULT),
			...Object.values(DEVICE_STATUS),
		];
		for (const meta of metas) {
			expect(typeof meta.icon).toBe("function");
			expect(meta.label.length).toBeGreaterThan(0);
		}
	});

	it("классы утилит заданы полными строками для каждого тона", () => {
		for (const tone of ["accent", "progress", "success", "held", "danger", "neutral"] as const) {
			const classes = TONE_CLASSES[tone];
			expect(Object.keys(classes).sort()).toEqual(["dot", "icon", "metric"]);
			expect(classes.dot).toMatch(/^bg-/);
			expect(classes.icon).toMatch(/^text-/);
			expect(classes.metric).toMatch(/^text-/);
		}
	});

	it("статус не несёт цветной подложки: тон только на точке и показателе", () => {
		for (const classes of Object.values(TONE_CLASSES)) {
			for (const value of Object.values(classes)) {
				expect(value).not.toMatch(/\/12|\/35|tint/);
			}
		}
	});

	it("в классах нет hex-литералов: только токены", () => {
		for (const classes of Object.values(TONE_CLASSES)) {
			for (const value of Object.values(classes)) {
				expect(value).not.toMatch(/#/);
			}
		}
	});
});

describe("deviceLiveness", () => {
	it("выведенное из эксплуатации устройство - neutral", () => {
		const liveness = deviceLiveness("decommissioned", "2026-09-12T11:59:00.000Z", NOW);
		expect(liveness.tone).toBe("neutral");
		expect(liveness.label).toBe("выведено из эксплуатации");
	});

	it("статус важнее свежести отметки", () => {
		const liveness = deviceLiveness("decommissioned", new Date(NOW).toISOString(), NOW);
		expect(liveness.label).toBe("выведено из эксплуатации");
	});

	it("отсутствие last_seen - «нет отметки», а не «никогда»", () => {
		const liveness = deviceLiveness("active", undefined, NOW);
		expect(liveness.tone).toBe("neutral");
		expect(liveness.label).toBe(NO_LAST_SEEN_LABEL);
	});

	it("в подсказке про отсутствие отметки объяснены оба случая и TTL Redis", () => {
		const liveness = deviceLiveness("active", undefined, NOW);
		expect(liveness.title).toContain("24 часа");
		expect(liveness.title).toContain("в последние сутки");
		expect(liveness.title).toContain("никогда не отмечалось");
		expect(liveness.title).not.toBe(NO_LAST_SEEN_LABEL);
	});

	it("свежая отметка в пределах окна - success", () => {
		const fresh = new Date(NOW - DEVICE_ONLINE_WINDOW_MS + 1000).toISOString();
		expect(deviceLiveness("active", fresh, NOW).tone).toBe("success");
		expect(deviceLiveness("active", fresh, NOW).label).toBe("на связи");
	});

	it("отметка старше окна - neutral, но время не теряется", () => {
		const stale = new Date(NOW - DEVICE_ONLINE_WINDOW_MS - 1000).toISOString();
		const liveness = deviceLiveness("active", stale, NOW);
		expect(liveness.tone).toBe("neutral");
		expect(liveness.label).toBe("было на связи");
		expect(liveness.title).toContain(stale);
	});

	it("неразобранная дата не выдаётся за время", () => {
		const liveness = deviceLiveness("active", "не дата", NOW);
		expect(liveness.tone).toBe("neutral");
		expect(liveness.label).toBe("отметка не разобрана");
		expect(liveness.title).toContain("не удалось разобрать");
	});
});
