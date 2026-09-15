import { describe, expect, it } from "vitest";
import { bucketHint, checkinVerdict } from "./checkin-verdict";
import { bucketOf } from "./bucket";
import type { CheckinCampaignContext, CheckinDeviceContext } from "./checkin-verdict";

const DEVICE_ID = "3f6d2d8e-4b1c-4a5f-9a7b-8e7d1c2a3b4c";
const CAMPAIGN_ID = "9b7f2a10-4c3d-4e5f-8a6b-1c2d3e4f5a6b";
const MODEL = "demo-sensor-v1";

// Закреплено в bucket.test.ts: для этой пары устройства и кампании бакет равен 66.
const KNOWN_BUCKET = 66;

function device(overrides: Partial<CheckinDeviceContext> = {}): CheckinDeviceContext {
	return {
		id: DEVICE_ID,
		device_model: MODEL,
		status: "active",
		current_version: "1.0.0",
		...overrides,
	};
}

function campaign(overrides: Partial<CheckinCampaignContext> = {}): CheckinCampaignContext {
	return {
		id: CAMPAIGN_ID,
		device_model: MODEL,
		status: "running",
		target_version: "1.4.2",
		active_stage: { target_percent: 100 },
		...overrides,
	};
}

describe("checkinVerdict", () => {
	it("бакет совпадает с детерминированным расчётом", () => {
		expect(bucketOf(DEVICE_ID, CAMPAIGN_ID)).toBe(KNOWN_BUCKET);
	});

	it("все условия выполнены - обновление выдаётся", () => {
		const verdict = checkinVerdict(device(), campaign());
		expect(verdict.kind).toBe("available");
		expect(verdict.updateAvailable).toBe(true);
		expect(verdict.bucket).toBe(KNOWN_BUCKET);
	});

	it("проверка 1: выведенное из эксплуатации устройство не получает обновление", () => {
		const verdict = checkinVerdict(device({ status: "decommissioned" }), campaign());
		expect(verdict.kind).toBe("denied");
		expect(verdict.denial).toBe("decommissioned");
		expect(verdict.updateAvailable).toBe(false);
	});

	it("проверка 1 идёт раньше остальных: статус важнее бакета и версии", () => {
		const verdict = checkinVerdict(
			device({ status: "decommissioned", current_version: "1.4.2" }),
			campaign({ active_stage: { target_percent: 1 } }),
		);
		expect(verdict.denial).toBe("decommissioned");
		// Бакет в этом случае не вычисляется: сервер до него не доходит.
		expect(verdict.bucket).toBeUndefined();
	});

	it("проверка 2: кампании нет вовсе", () => {
		const verdict = checkinVerdict(device(), undefined);
		expect(verdict.denial).toBe("no_running_campaign");
	});

	it("проверка 2: кампания не для этой модели", () => {
		const verdict = checkinVerdict(device(), campaign({ device_model: "demo-gateway-x1" }));
		expect(verdict.denial).toBe("no_running_campaign");
	});

	it("проверка 2: кампания не выполняется", () => {
		for (const status of ["draft", "paused", "completed", "rolled_back"] as const) {
			expect(checkinVerdict(device(), campaign({ status })).denial).toBe("no_running_campaign");
		}
	});

	it("проверка 3: текущая версия равна целевой", () => {
		const verdict = checkinVerdict(device({ current_version: "1.4.2" }), campaign());
		expect(verdict.denial).toBe("version_up_to_date");
	});

	it("проверка 3: текущая версия выше целевой", () => {
		const verdict = checkinVerdict(device({ current_version: "2.0.0" }), campaign());
		expect(verdict.denial).toBe("version_up_to_date");
	});

	it("проверка 3 идёт раньше проверки бакета", () => {
		const verdict = checkinVerdict(
			device({ current_version: "1.4.2" }),
			campaign({ active_stage: { target_percent: 1 } }),
		);
		expect(verdict.denial).toBe("version_up_to_date");
		expect(verdict.bucket).toBe(KNOWN_BUCKET);
	});

	it("проверка 4: бакет больше охвата стадии", () => {
		const verdict = checkinVerdict(device(), campaign({ active_stage: { target_percent: 50 } }));
		expect(verdict.denial).toBe("outside_bucket");
		expect(verdict.bucket).toBe(KNOWN_BUCKET);
	});

	it("проверка 4: граница включительная, бакет равный охвату проходит", () => {
		expect(
			checkinVerdict(device(), campaign({ active_stage: { target_percent: KNOWN_BUCKET } })).kind,
		).toBe("available");
		expect(
			checkinVerdict(device(), campaign({ active_stage: { target_percent: KNOWN_BUCKET - 1 } }))
				.denial,
		).toBe("outside_bucket");
	});

	it("не угадывает, когда версию не разобрать", () => {
		const verdict = checkinVerdict(device({ current_version: "latest" }), campaign());
		expect(verdict.kind).toBe("unknown");
		expect(verdict.updateAvailable).toBeUndefined();
		expect(verdict.unknown).toBe("unparseable_version");
		expect(verdict.bucket).toBe(KNOWN_BUCKET);
	});

	it("не угадывает, когда активная стадия неизвестна", () => {
		const verdict = checkinVerdict(device(), campaign({ active_stage: undefined }));
		expect(verdict.kind).toBe("unknown");
		expect(verdict.unknown).toBe("no_active_stage");
	});

	it("pre-release считается более ранней версией, чем релиз", () => {
		const verdict = checkinVerdict(device({ current_version: "1.4.2-rc.1" }), campaign());
		expect(verdict.kind).toBe("available");
	});
});

describe("bucketHint", () => {
	it("поясняет попадание и промах относительно охвата", () => {
		expect(bucketHint(KNOWN_BUCKET, 100)).toContain("обновление выдаётся");
		expect(bucketHint(KNOWN_BUCKET, 50)).toContain("обновление не выдаётся");
	});
});
