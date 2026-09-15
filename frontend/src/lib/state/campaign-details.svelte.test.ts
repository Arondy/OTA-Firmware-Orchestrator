import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "$lib/api/errors";
import type { Campaign } from "$lib/api/types";
import { apiAvailability } from "./api-availability.svelte";
import { campaignDetails } from "./campaign-details.svelte";

/**
 * Агрегатор деталей обязан пробрасывать тотальный сбой наружу и не отмечать
 * доступность API сам (notifyAvailability: false): иначе его «успешный» пустой
 * список снимал бы баннер «API недоступен» на каждом такте опроса при мёртвом
 * сервере.
 */

const endpoints = vi.hoisted(() => ({
	getRolloutCampaign: vi.fn(),
}));

vi.mock("$lib/api/endpoints", () => endpoints);

function campaignOf(id: string): Campaign {
	return {
		id,
		firmware_version_id: "3f6d2d8e-4b1c-4a5f-9a7b-8e7d1c2a3b4c",
		device_model: "demo-sensor-v1",
		status: "completed",
		created_at: "2026-09-01T10:00:00Z",
		rollout_stages: [],
	};
}

function networkError(): ApiError {
	return new ApiError({ status: 0, method: "GET", url: "/api/v1/test" });
}

/** Идентификаторы уникальны между тестами: одиночка живёт весь файл. */
const EMPTY = "11111111-1111-4111-8111-111111111101";
const FAILING_A = "11111111-1111-4111-8111-111111111102";
const FAILING_B = "11111111-1111-4111-8111-111111111103";
const STORED = "11111111-1111-4111-8111-111111111104";
const PART_OK = "11111111-1111-4111-8111-111111111105";
const PART_FAIL = "11111111-1111-4111-8111-111111111106";
const OK_ONLY = "11111111-1111-4111-8111-111111111107";

beforeEach(() => {
	vi.useFakeTimers();
	apiAvailability.reset();
	endpoints.getRolloutCampaign.mockReset();
});

afterEach(() => {
	vi.useRealTimers();
});

describe("campaignDetails: пустой опрос", () => {
	it("«нечего опрашивать» ошибкой не считается и запросов не делает", async () => {
		await campaignDetails.refresh();

		expect(campaignDetails.error).toBeUndefined();
		expect(endpoints.getRolloutCampaign).not.toHaveBeenCalled();
	});
});

describe("campaignDetails: тотальный сбой", () => {
	it("помечает ресурс ошибкой, а не «успешным» пустым списком", async () => {
		endpoints.getRolloutCampaign.mockRejectedValue(networkError());
		campaignDetails.ensure([FAILING_A]);

		await vi.advanceTimersByTimeAsync(0);

		expect(campaignDetails.error).toBeInstanceOf(ApiError);
	});

	it("не снимает баннер «API недоступен» после собственных неудач", async () => {
		endpoints.getRolloutCampaign.mockRejectedValue(networkError());
		campaignDetails.ensure([FAILING_B]);

		await vi.advanceTimersByTimeAsync(0);

		// Подзапросы уже отметили сбой связи: обёртка не должна перекрывать его
		// «успехом» пустого списка (источник мерцания баннера).
		expect(apiAvailability.down).toBe(true);
	});

	it("сохраняет ранее загруженные детали на экране", async () => {
		endpoints.getRolloutCampaign.mockImplementation(async (id: string) => {
			if (id === STORED) return campaignOf(STORED);
			throw networkError();
		});
		campaignDetails.ensure([STORED]);
		await vi.advanceTimersByTimeAsync(0);
		expect(campaignDetails.get(STORED)).toBeDefined();

		// Следующий такт падает целиком: кэш остаётся, ошибка видна.
		await campaignDetails.refresh();

		expect(campaignDetails.get(STORED)).toBeDefined();
		expect(campaignDetails.error).toBeInstanceOf(ApiError);
	});
});

describe("campaignDetails: частичный успех", () => {
	it("ошибка одного подзапроса не роняет весь ресурс", async () => {
		endpoints.getRolloutCampaign.mockImplementation(async (id: string) => {
			if (id === PART_FAIL) throw networkError();
			return campaignOf(id);
		});
		campaignDetails.ensure([PART_OK, PART_FAIL]);

		await vi.advanceTimersByTimeAsync(0);

		expect(campaignDetails.error).toBeUndefined();
		expect(campaignDetails.get(PART_OK)).toBeDefined();
		expect(campaignDetails.get(PART_FAIL)).toBeUndefined();
	});

	it("успешные подзапросы сами отмечают доступность API", async () => {
		endpoints.getRolloutCampaign.mockResolvedValue(campaignOf(OK_ONLY));
		campaignDetails.ensure([OK_ONLY]);

		await vi.advanceTimersByTimeAsync(0);

		expect(apiAvailability.down).toBe(false);
		expect(apiAvailability.lastSuccessAt).toBeTypeOf("number");
	});
});

describe("campaignDetails: по умолчанию опрашивает только обзор", () => {
	it("не повторяет запрос уже загруженного идентификатора", async () => {
		endpoints.getRolloutCampaign.mockImplementation(async (id: string) => campaignOf(id));
		// Одиночка живёт весь файл: в очереди могли остаться идентификаторы
		// прошлых тестов, поэтому считаем вызовы только по своему.
		campaignDetails.ensure([EMPTY]);
		await vi.advanceTimersByTimeAsync(0);
		const callsFor = () =>
			endpoints.getRolloutCampaign.mock.calls.filter((args) => args[0] === EMPTY).length;
		expect(callsFor()).toBe(1);

		campaignDetails.ensure([EMPTY]);
		await vi.advanceTimersByTimeAsync(0);
		expect(callsFor()).toBe(1);
	});
});
