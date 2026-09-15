/**
 * По одной типизированной функции на каждый из 15 эндпоинтов контракта
 * (00-CONTEXT §5.1). Сигнатуры используют сгенерированные типы и для параметров,
 * и для результатов, поэтому расхождение со спекой ловится на компиляции.
 *
 * Компоненты вызывают только эти функции и не собирают пути руками.
 */
import { request, requestVoid } from "./client";
import type {
	Campaign,
	CampaignPage,
	CheckinInput,
	CheckinResult,
	CreateCampaignInput,
	CreateDeviceInput,
	CreateFirmwareInput,
	Device,
	DeviceListParams,
	DevicePage,
	FirmwareListParams,
	FirmwarePage,
	FirmwareVersion,
	HealthResponse,
	ReportInput,
	UpdateAttempt,
} from "./types";

/** Общая опция отмены: её передаёт слой данных (`resource.svelte.ts`). */
export interface CallOptions {
	signal?: AbortSignal;
}

/**
 * Недокументированный ответ `POST /devices/{id}/report`: при сбое публикации
 * в Kafka сервер отдаёт `503 {"error":"service unavailable"}`, при этом строка
 * уже записана в Postgres (00-CONTEXT §5.3). В `openapi.yaml` кода нет,
 * поэтому сгенерированные типы о нём не знают и текст задаётся здесь.
 */
const REPORT_503_TEXT = {
	503: {
		message: "Отчёт записан в базу, но событие не ушло в Kafka",
		hint: "Попытка обновления сохранена, однако метрики стадии не изменятся: контроллер не получил событие.",
	},
} as const;

/* 1. Здоровье */

export function checkHealth(options: CallOptions = {}): Promise<HealthResponse> {
	return request<HealthResponse>("GET", "/healthz", options);
}

/**
 * Здоровье rollout-контроллера: Connect-эндпоинт `HealthService.CheckHealth`
 * (POST, пустое тело, ответ `{"status":"OK"}`). Напрямую контроллер браузеру
 * недоступен: путь `/controller/*` проксируют на него Caddy (см. Caddyfile)
 * и dev-сервер Vite, поэтому здесь снова относительный путь своего origin.
 */
export interface ControllerHealthResponse {
	status: string;
}

export function checkControllerHealth(
	options: CallOptions = {},
): Promise<ControllerHealthResponse> {
	return request<ControllerHealthResponse>(
		"POST",
		"/controller/health.v1.HealthService/CheckHealth",
		{ ...options, body: {} },
	);
}

/* 2-6. Устройства */

export function listDevices(
	params: DeviceListParams = {},
	options: CallOptions = {},
): Promise<DevicePage> {
	return request<DevicePage>("GET", "/api/v1/devices", {
		...options,
		query: {
			device_model: params.device_model,
			status: params.status,
			page: params.page,
			limit: params.limit,
		},
	});
}

export function createDevice(input: CreateDeviceInput, options: CallOptions = {}): Promise<Device> {
	return request<Device>("POST", "/api/v1/devices", { ...options, body: input });
}

export function decommissionDevice(id: string, options: CallOptions = {}): Promise<Device> {
	return request<Device>("POST", `/api/v1/devices/${encodeURIComponent(id)}/decommission`, options);
}

export function checkinDevice(
	id: string,
	input: CheckinInput,
	options: CallOptions = {},
): Promise<CheckinResult> {
	return request<CheckinResult>("POST", `/api/v1/devices/${encodeURIComponent(id)}/checkin`, {
		...options,
		body: input,
	});
}

export function reportUpdateAttempt(
	id: string,
	input: ReportInput,
	options: CallOptions = {},
): Promise<UpdateAttempt> {
	return request<UpdateAttempt>("POST", `/api/v1/devices/${encodeURIComponent(id)}/report`, {
		...options,
		body: input,
		statusText: REPORT_503_TEXT,
	});
}

/* 7-8. Прошивки */

export function listFirmwareVersions(
	params: FirmwareListParams = {},
	options: CallOptions = {},
): Promise<FirmwarePage> {
	return request<FirmwarePage>("GET", "/api/v1/firmware", {
		...options,
		query: {
			device_model: params.device_model,
			page: params.page,
			limit: params.limit,
		},
	});
}

export function createFirmwareVersion(
	input: CreateFirmwareInput,
	options: CallOptions = {},
): Promise<FirmwareVersion> {
	return request<FirmwareVersion>("POST", "/api/v1/firmware", { ...options, body: input });
}

/* 9-15. Кампании раскатки */

/**
 * Фильтра по статусу у этого эндпоинта нет (§5.1), есть только `page` и `limit`.
 * Экран «Обзор» поэтому выкачивает несколько страниц и фильтрует на клиенте.
 */
export function listRolloutCampaigns(
	params: { page?: number; limit?: number } = {},
	options: CallOptions = {},
): Promise<CampaignPage> {
	return request<CampaignPage>("GET", "/api/v1/campaigns", {
		...options,
		query: { page: params.page, limit: params.limit },
	});
}

export function createRolloutCampaign(
	input: CreateCampaignInput,
	options: CallOptions = {},
): Promise<Campaign> {
	return request<Campaign>("POST", "/api/v1/campaigns", { ...options, body: input });
}

export function getRolloutCampaign(id: string, options: CallOptions = {}): Promise<Campaign> {
	return request<Campaign>("GET", `/api/v1/campaigns/${encodeURIComponent(id)}`, options);
}

export function startRolloutCampaign(id: string, options: CallOptions = {}): Promise<Campaign> {
	return request<Campaign>("POST", `/api/v1/campaigns/${encodeURIComponent(id)}/start`, options);
}

export function pauseRolloutCampaign(id: string, options: CallOptions = {}): Promise<Campaign> {
	return request<Campaign>("POST", `/api/v1/campaigns/${encodeURIComponent(id)}/pause`, options);
}

export function resumeRolloutCampaign(id: string, options: CallOptions = {}): Promise<Campaign> {
	return request<Campaign>("POST", `/api/v1/campaigns/${encodeURIComponent(id)}/resume`, options);
}

/**
 * Ручной откат асинхронный: сервер отвечает `202` с пустым телом, а статус
 * станет `rolled_back` позже, когда решение применит консьюмер. Тела здесь нет,
 * поэтому `res.json()` не вызывается, а вызывающий код обязан опросить кампанию,
 * а не менять статус оптимистично (§5.3, §7.6).
 */
export function rollbackRolloutCampaign(id: string, options: CallOptions = {}): Promise<void> {
	return requestVoid("POST", `/api/v1/campaigns/${encodeURIComponent(id)}/rollback`, options);
}
