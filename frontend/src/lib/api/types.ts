/**
 * Человекочитаемые алиасы поверх сгенерированного `schema.d.ts`.
 *
 * Единственный источник правды - `ota-orchestrator/api/openapi.yaml`; файл
 * `schema.d.ts` создаётся командой `bun run gen:api` и руками не правится.
 * Если имя схемы в спеке изменится, здесь сломается компиляция - так и задумано.
 */
import type { components, operations } from "./schema";

/* Ответы сервера */

export type Device = components["schemas"]["DeviceResponse"];
export type FirmwareVersion = components["schemas"]["FirmwareVersionResponse"];
export type CampaignListItem = components["schemas"]["RolloutCampaignListItemResponse"];
export type Campaign = components["schemas"]["RolloutCampaignResponse"];
export type Stage = components["schemas"]["RolloutStageResponse"];
export type Stats = components["schemas"]["RolloutCampaignStats"];
export type UpdateAttempt = components["schemas"]["ReportDeviceResponse"];
export type CheckinResult = components["schemas"]["CheckinDeviceResponse"];
export type HealthResponse =
	operations["checkHealth"]["responses"][200]["content"]["application/json"];

/* Перечисления */

export type CampaignStatus = components["schemas"]["RolloutCampaignsStatus"];
export type StageStatus = components["schemas"]["RolloutStagesStatus"];
export type DeviceStatus = components["schemas"]["DeviceStatus"];
export type AttemptResult = components["schemas"]["UpdateAttemptsResult"];

/* Тела запросов (DTO). Сервер декодирует их с DisallowUnknownFields,
 * поэтому отправляем ровно эти поля и никаких лишних (00-CONTEXT §5.3). */

export type CreateDeviceInput = components["schemas"]["CreateDeviceRequest"];
export type CheckinInput = components["schemas"]["CheckinDeviceRequest"];
export type ReportInput = components["schemas"]["ReportDeviceRequest"];
export type CreateFirmwareInput = components["schemas"]["CreateFirmwareVersionRequest"];
export type CreateStageInput = components["schemas"]["RolloutStageRequest"];
export type CreateCampaignInput = components["schemas"]["CreateRolloutCampaignRequest"];

/* Списковые обёртки: у них нет поля total, только массив (00-CONTEXT §5.2). */

export type DevicePage = components["schemas"]["ListDevicesResponse"];
export type FirmwarePage = components["schemas"]["ListFirmwareVersionsResponse"];
export type CampaignPage = components["schemas"]["ListRolloutCampaignsResponse"];

/* Тело ошибки: `{ error }`, `{ error, fields }` или `{ error, params }`. */

export type ErrorBody = components["schemas"]["ErrorResponse"];
export type ValidationErrorBody = components["schemas"]["ValidationErrorResponse"];

/**
 * Параметры списка. `page` от 1, `limit` не больше серверного
 * DB_PAGINATION_LIMIT (100 в .env.example); превышение даёт 400.
 */
export interface ListParams {
	page?: number;
	limit?: number;
}

export interface DeviceListParams extends ListParams {
	device_model?: string;
	status?: DeviceStatus;
}

export interface FirmwareListParams extends ListParams {
	device_model?: string;
}
