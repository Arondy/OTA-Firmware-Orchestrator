import { describe, expect, it } from "vitest";
import type { Stage } from "$lib/api/types";
import {
	attentionReason,
	attentionText,
	progressOf,
	sortActiveRows,
	topModels,
	type CampaignRow,
} from "./campaign-row";

function stage(overrides: Partial<Stage> = {}): Stage {
	return {
		id: "stage-1",
		campaign_id: "campaign-1",
		order_index: 1,
		target_percent: 50,
		min_sample_size: 20,
		success_threshold: 0.95,
		status: "active",
		...overrides,
	};
}

function row(overrides: Partial<CampaignRow> = {}): CampaignRow {
	return {
		id: "campaign-1",
		model: "demo-sensor-v1",
		version: "2.0.0",
		status: "running",
		createdAt: "2026-09-12T08:00:00.000Z",
		startedAt: "2026-09-12T09:00:00.000Z",
		completedAt: undefined,
		stages: [
			stage({ id: "stage-0", order_index: 0, status: "passed" }),
			stage(),
			stage({ id: "stage-2", order_index: 2, status: "pending" }),
		],
		stats: { active_stage_id: "stage-1", success_rate: 0.98, sample_size: 34 },
		detailLoaded: true,
		...overrides,
	};
}

describe("attentionReason", () => {
	it("ниже порога при набранной выборке", () => {
		expect(
			attentionReason(
				row({ stats: { active_stage_id: "s", success_rate: 0.91, sample_size: 34 } }),
			),
		).toBe("below-threshold");
	});

	it("при нулевой выборке причина - тонкая выборка, а не «ниже порога»", () => {
		expect(
			attentionReason(row({ stats: { active_stage_id: "s", success_rate: 0, sample_size: 0 } })),
		).toBe("thin-sample");
	});

	it("тонкая выборка, когда порог ещё не нарушен", () => {
		expect(
			attentionReason(
				row({ stats: { active_stage_id: "s", success_rate: 0.99, sample_size: 12 } }),
			),
		).toBe("thin-sample");
	});

	it("пауза без проблем с метриками", () => {
		expect(attentionReason(row({ status: "paused" }))).toBe("paused");
	});

	it("running без метрик - контроллер молчит", () => {
		expect(attentionReason(row({ stats: undefined }))).toBe("no-metrics");
	});

	it("здоровая раскатка не требует внимания", () => {
		expect(attentionReason(row())).toBeUndefined();
	});

	it("терминальные статусы не требуют внимания", () => {
		expect(attentionReason(row({ status: "completed", stats: undefined }))).toBeUndefined();
		expect(attentionReason(row({ status: "rolled_back", stats: undefined }))).toBeUndefined();
	});

	it("пока деталь не загружена, причина неизвестна, а не «нет метрик»", () => {
		expect(attentionReason(row({ stats: undefined, detailLoaded: false }))).toBeUndefined();
	});
});

describe("attentionText", () => {
	it("формат «ниже порога» из спеки", () => {
		const text = attentionText(
			row({ stats: { active_stage_id: "s", success_rate: 0.913, sample_size: 34 } }),
			"below-threshold",
		);
		expect(text).toBe(
			"demo-sensor-v1 2.0.0: процент успеха 91,3% ниже порога 95% на стадии 2 (50%).",
		);
	});

	it("формат тонкой выборки из спеки", () => {
		const text = attentionText(
			row({ stats: { active_stage_id: "s", success_rate: 0.99, sample_size: 12 } }),
			"thin-sample",
		);
		expect(text).toBe("demo-sensor-v1 2.0.0: 12/20 наблюдений");
	});
});

describe("sortActiveRows", () => {
	it("требующие внимания выше здоровых, running выше paused", () => {
		const healthy = row({ id: "a", startedAt: "2026-09-12T10:00:00.000Z" });
		const sick = row({
			id: "b",
			startedAt: "2026-09-12T07:00:00.000Z",
			stats: { active_stage_id: "s", success_rate: 0.5, sample_size: 30 },
		});
		const paused = row({ id: "c", status: "paused", startedAt: "2026-09-12T11:00:00.000Z" });

		expect(sortActiveRows([healthy, paused, sick]).map((item) => item.id)).toEqual(["b", "a", "c"]);
	});

	it("при равной тяжести - сначала более свежие", () => {
		const older = row({ id: "old", startedAt: "2026-09-12T06:00:00.000Z" });
		const newer = row({ id: "new", startedAt: "2026-09-12T09:00:00.000Z" });
		expect(sortActiveRows([older, newer]).map((item) => item.id)).toEqual(["new", "old"]);
	});
});

describe("progressOf", () => {
	it("номер активной стадии плюс один и всего стадий", () => {
		expect(progressOf(row()).label).toBe("2/3");
	});

	it("без стадий - прочерк с причиной", () => {
		const result = progressOf(row({ stages: [] }));
		expect(result.label).toBe("нет");
		expect(result.title).toBeUndefined();
	});

	it("завершённая кампания показывает пройденные стадии", () => {
		const result = progressOf(
			row({
				status: "completed",
				stages: [stage({ status: "passed" }), stage({ status: "passed" })],
			}),
		);
		expect(result.label).toBe("2/2");
		expect(result.title).toBeUndefined();
	});

	it("откаченная кампания указывает стадию остановки", () => {
		const result = progressOf(
			row({
				status: "rolled_back",
				stages: [stage({ status: "passed" }), stage({ status: "failed" })],
			}),
		);
		expect(result.label).toBe("2/2");
		expect(result.title).toContain("остановилась на стадии 2");
	});

	it("черновик без активной стадии объясняет причину", () => {
		const result = progressOf(row({ status: "draft", stages: [stage({ status: "pending" })] }));
		expect(result.label).toBe("нет");
		expect(result.title).toContain("Кампания не запущена");
	});
});

describe("topModels", () => {
	it("топ по числу кампаний, при равенстве - по алфавиту", () => {
		const rows = [
			row({ id: "1", model: "b-model" }),
			row({ id: "2", model: "a-model" }),
			row({ id: "3", model: "a-model" }),
		];
		expect(topModels(rows, 2)).toEqual([
			{ model: "a-model", count: 2 },
			{ model: "b-model", count: 1 },
		]);
	});
});
