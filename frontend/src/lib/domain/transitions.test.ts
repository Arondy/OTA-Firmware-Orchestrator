import { describe, expect, it } from "vitest";
import type { CampaignStatus } from "$lib/api/types";
import { ACTION_META, allowedActions, isLive, isTerminal } from "./transitions";

const ALL_STATUSES: CampaignStatus[] = ["draft", "running", "paused", "completed", "rolled_back"];

describe("allowedActions", () => {
	it("черновик можно только запустить", () => {
		expect(allowedActions("draft")).toEqual(["start"]);
	});

	it("выполняющуюся кампанию можно поставить на паузу и откатить", () => {
		expect(allowedActions("running")).toEqual(["pause", "rollback"]);
	});

	it("кампанию на паузе можно возобновить и откатить", () => {
		expect(allowedActions("paused")).toEqual(["resume", "rollback"]);
	});

	it("терминальные статусы не дают действий", () => {
		expect(allowedActions("completed")).toEqual([]);
		expect(allowedActions("rolled_back")).toEqual([]);
	});

	it("каждый статус перечисления покрыт", () => {
		for (const status of ALL_STATUSES) {
			expect(Array.isArray(allowedActions(status))).toBe(true);
		}
	});
});

describe("isTerminal / isLive", () => {
	it("терминальные статусы", () => {
		expect(isTerminal("completed")).toBe(true);
		expect(isTerminal("rolled_back")).toBe(true);
		expect(isTerminal("draft")).toBe(false);
		expect(isTerminal("running")).toBe(false);
		expect(isTerminal("paused")).toBe(false);
	});

	it("опрашивать нужно только выполняющуюся кампанию и кампанию на паузе", () => {
		expect(isLive("running")).toBe(true);
		expect(isLive("paused")).toBe(true);
		expect(isLive("draft")).toBe(false);
		expect(isLive("completed")).toBe(false);
		expect(isLive("rolled_back")).toBe(false);
	});

	it("терминальный статус никогда не «живой»", () => {
		for (const status of ALL_STATUSES) {
			if (isTerminal(status)) expect(isLive(status)).toBe(false);
		}
	});
});

describe("ACTION_META", () => {
	it("откат помечен как опасное и асинхронное действие", () => {
		expect(ACTION_META.rollback.confirm).toBe(true);
		expect(ACTION_META.rollback.async).toBe(true);
		expect(ACTION_META.rollback.tone).toBe("danger");
	});

	it("остальные действия синхронные и без подтверждения", () => {
		for (const action of ["start", "pause", "resume"] as const) {
			expect(ACTION_META[action].async).toBeUndefined();
			expect(ACTION_META[action].confirm).toBeUndefined();
		}
	});

	it("подписи взяты из словаря и стоят в повелительном наклонении", () => {
		expect(ACTION_META.start.label).toBe("Запустить");
		expect(ACTION_META.pause.label).toBe("Поставить на паузу");
		expect(ACTION_META.resume.label).toBe("Возобновить");
		expect(ACTION_META.rollback.label).toBe("Откатить");
	});

	it("у каждого действия есть подпись на время запроса и пояснение", () => {
		for (const action of ["start", "pause", "resume", "rollback"] as const) {
			expect(ACTION_META[action].pendingLabel.length).toBeGreaterThan(0);
			expect(ACTION_META[action].description.length).toBeGreaterThan(0);
		}
	});

	it("подпись ожидания отката говорит, что статус изменится позже", () => {
		expect(ACTION_META.rollback.pendingLabel).toBe("Откат запрошен");
	});
});
