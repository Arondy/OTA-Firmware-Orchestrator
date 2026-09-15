import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "$lib/api/errors";
import type { Campaign } from "$lib/api/types";
import { actionErrorHeadline } from "$lib/domain/action-errors";
import { campaignActions } from "./campaign-actions.svelte";
import { toast } from "./toast";

vi.mock("./toast", () => ({
	toast: {
		success: vi.fn(),
		error: vi.fn(),
		info: vi.fn(),
		dismiss: vi.fn(),
	},
}));

const endpoints = vi.hoisted(() => ({
	getRolloutCampaign: vi.fn(),
	rollbackRolloutCampaign: vi.fn(),
	startRolloutCampaign: vi.fn(),
	pauseRolloutCampaign: vi.fn(),
	resumeRolloutCampaign: vi.fn(),
}));

vi.mock("$lib/api/endpoints", () => endpoints);

function campaign(status: Campaign["status"]): Campaign {
	return {
		id: "9b7f2a10-4c3d-4e5f-8a6b-1c2d3e4f5a6b",
		firmware_version_id: "3f6d2d8e-4b1c-4a5f-9a7b-8e7d1c2a3b4c",
		device_model: "demo-sensor-v1",
		status,
		created_at: "2026-09-01T10:00:00Z",
		rollout_stages: [],
	};
}

beforeEach(() => {
	vi.useFakeTimers();
	campaignActions.cancelWatches();
	vi.clearAllMocks();
	endpoints.getRolloutCampaign.mockReset();
	endpoints.rollbackRolloutCampaign.mockReset();
});

describe("откат: 202 и ожидание фактического статуса", () => {
	it("после 202 помечает ожидание и снимает его, когда статус стал «откачена»", async () => {
		const id = campaign("running").id;
		endpoints.rollbackRolloutCampaign.mockResolvedValue(undefined);
		endpoints.getRolloutCampaign
			.mockResolvedValueOnce(campaign("running"))
			.mockResolvedValueOnce(campaign("rolled_back"));

		await campaignActions.rollback(id, "demo-sensor-v1 1.1.0", "выполняется");
		expect(campaignActions.isRollbackInFlight(id)).toBe(true);

		await vi.advanceTimersByTimeAsync(1500);
		expect(campaignActions.isRollbackInFlight(id)).toBe(true);

		await vi.advanceTimersByTimeAsync(1500);
		expect(campaignActions.isRollbackInFlight(id)).toBe(false);
		expect(campaignActions.isRollbackTimedOut(id)).toBe(false);
		expect(toast.success).toHaveBeenCalledWith("Кампания откачена", undefined, `${id}:rollback`);
	});

	it("через 20 с ожидания оставляет настоящий статус и не объявляет сбой", async () => {
		const id = campaign("running").id;
		endpoints.rollbackRolloutCampaign.mockResolvedValue(undefined);
		endpoints.getRolloutCampaign.mockResolvedValue(campaign("running"));

		await campaignActions.rollback(id, "demo-sensor-v1 1.1.0", "выполняется");
		await vi.advanceTimersByTimeAsync(20_000 + 1500);

		expect(campaignActions.isRollbackInFlight(id)).toBe(false);
		expect(campaignActions.isRollbackTimedOut(id)).toBe(true);
		expect(toast.error).not.toHaveBeenCalled();
		expect(toast.success).not.toHaveBeenCalled();
	});

	it("опрос идёт каждые 1500 мс и останавливается вместе с ожиданием", async () => {
		const id = campaign("paused").id;
		endpoints.rollbackRolloutCampaign.mockResolvedValue(undefined);
		endpoints.getRolloutCampaign.mockResolvedValue(campaign("paused"));

		await campaignActions.rollback(id, "demo-sensor-v1 1.1.0", "на паузе");
		await vi.advanceTimersByTimeAsync(4500);
		expect(endpoints.getRolloutCampaign).toHaveBeenCalledTimes(3);

		campaignActions.cancelWatches();
		await vi.advanceTimersByTimeAsync(10_000);
		expect(endpoints.getRolloutCampaign).toHaveBeenCalledTimes(3);
	});
});

describe("отказы действий", () => {
	it("400 на откат сохраняется с переводом по текущему статусу", async () => {
		const id = campaign("completed").id;
		endpoints.rollbackRolloutCampaign.mockRejectedValue(
			new ApiError({
				status: 400,
				method: "POST",
				url: `/api/v1/campaigns/${id}/rollback`,
				serverMessage: "can't rollback non- running/paused rollout campaign",
			}),
		);

		await campaignActions.rollback(id, "demo-sensor-v1 1.1.0", "завершена");

		const failure = campaignActions.failure(id);
		if (!failure) throw new Error("ошибка действия не сохранена");
		expect(failure.action).toBe("rollback");
		expect(actionErrorHeadline("rollback", failure.error, "завершена")).toBe(
			"Откат недоступен для кампании в статусе «завершена»",
		);
		expect(toast.error).toHaveBeenCalled();
		expect(campaignActions.isRollbackInFlight(id)).toBe(false);
	});

	it("409 на старт переводится как конфликт по модели", async () => {
		const id = campaign("draft").id;
		endpoints.startRolloutCampaign.mockRejectedValue(
			new ApiError({
				status: 409,
				method: "POST",
				url: `/api/v1/campaigns/${id}/start`,
				serverMessage: "another campaign for this model is already running",
			}),
		);

		await campaignActions.start(id, "demo-sensor-v1 1.1.0", "черновик");

		const failure = campaignActions.failure(id);
		if (!failure) throw new Error("ошибка действия не сохранена");
		expect(actionErrorHeadline("start", failure.error, "черновик")).toBe(
			"Для этой модели уже выполняется другая кампания",
		);
	});

	it("синхронный старт передаёт ответ кампании подписчикам без лишнего запроса", async () => {
		const id = campaign("draft").id;
		const updated = campaign("running");
		endpoints.startRolloutCampaign.mockResolvedValue(updated);

		let received: Campaign | undefined;
		const off = campaignActions.onChanged((value) => {
			received = value;
		});
		await campaignActions.start(id, "demo-sensor-v1 1.1.0", "черновик");

		expect(received).toBe(updated);
		expect(endpoints.getRolloutCampaign).not.toHaveBeenCalled();
		off();
	});
});
