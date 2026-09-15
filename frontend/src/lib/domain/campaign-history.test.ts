import { describe, expect, it } from "vitest";
import type { Campaign } from "$lib/api/types";
import { buildTimeline, stageNumber } from "./campaign-history";

function base(status: Campaign["status"]): Campaign {
	return {
		id: "9b7f2a10-4c3d-4e5f-8a6b-1c2d3e4f5a6b",
		firmware_version_id: "3f6d2d8e-4b1c-4a5f-9a7b-8e7d1c2a3b4c",
		device_model: "demo-sensor-v1",
		status,
		created_at: "2026-09-01T10:00:00Z",
		rollout_stages: [
			{
				id: "s1",
				campaign_id: "c",
				order_index: 0,
				target_percent: 10,
				min_sample_size: 20,
				success_threshold: 0.95,
				status: "passed",
				entered_at: "2026-09-01T11:00:00Z",
			},
			{
				id: "s2",
				campaign_id: "c",
				order_index: 1,
				target_percent: 100,
				min_sample_size: 20,
				success_threshold: 0.95,
				status: "active",
				entered_at: "2026-09-01T12:00:00Z",
			},
		],
	};
}

describe("buildTimeline", () => {
	it("черновик: единственная запись о создании, без выдуманных событий", () => {
		const draft = base("draft");
		for (const stage of draft.rollout_stages) stage.entered_at = undefined;
		const entries = buildTimeline(draft);
		expect(entries).toHaveLength(1);
		expect(entries[0].title).toBe("Кампания создана");
	});

	it("порядок по времени: создание, старт, переходы стадий", () => {
		const campaign = base("running");
		campaign.started_at = "2026-09-01T10:30:00Z";
		const entries = buildTimeline(campaign);
		expect(entries.map((entry) => entry.key)).toEqual([
			"created",
			"started",
			"stage-s1",
			"stage-s2",
		]);
		expect(entries[1].title).toBe("Старт раскатки");
		expect(entries[1].note).toBe("стадия 1 (10%): мин. выборка 20, порог успеха 95%");
		expect(entries[2].title).toBe("Стадия 1 перешла в статус «пройдена»");
		expect(entries[2].note).toBe("мин. выборка 20, порог успеха 95%");
	});

	it("откат без completed_at: запись есть, времени нет, причина названа", () => {
		const entries = buildTimeline(base("rolled_back"));
		const last = entries.at(-1);
		if (!last) throw new Error("история пуста");
		expect(last.title).toBe("Кампания откачена");
		expect(last.at).toBeUndefined();
		expect(last.note).toBe("точное время применения решения API не возвращает");
	});

	it("переход стадии со снимком метрик: выборка и процент успеха второй строкой", () => {
		const campaign = base("running");
		campaign.started_at = "2026-09-01T10:30:00Z";
		const entries = buildTimeline(campaign, {
			s1: { sampleSize: 42, successRate: 0.976 },
		});
		const stageEntry = entries.find((entry) => entry.key === "stage-s1");
		expect(stageEntry?.note).toBe("выборка 42, процент успеха 97,6%");
	});

	it("метрики активной стадии берутся из ответа API без снимков", () => {
		const campaign = base("running");
		campaign.stats = {
			active_stage_id: "s2",
			sample_size: 7,
			success_rate: 6 / 7,
		};
		const entries = buildTimeline(campaign);
		const stageEntry = entries.find((entry) => entry.key === "stage-s2");
		expect(stageEntry?.note).toContain("выборка 7");
	});

	it("завершённая кампания: последняя запись в тоне успеха", () => {
		const campaign = base("completed");
		campaign.completed_at = "2026-09-02T10:00:00Z";
		const last = buildTimeline(campaign).at(-1);
		if (!last) throw new Error("история пуста");
		expect(last.title).toBe("Раскатка завершена");
		expect(last.tone).toBe("success");
	});
});

it("номер стадии для оператора начинается с единицы", () => {
	expect(stageNumber(0)).toBe(1);
	expect(stageNumber(2)).toBe(3);
});
