/**
 * Черновик регистрации прошивки.
 *
 * Правила зеркалируют схему `CreateFirmwareVersionRequest` из openapi.yaml:
 * модель 2..64, версия - тем же регулярным выражением semver, что принимает
 * сервер (`logic/semver.ts`), сумма - 64 шестнадцатеричных символа
 * (регистр не важен, перед отправкой нормируется к нижнему), ссылка -
 * абсолютный адрес.
 */
import type { CreateFirmwareInput } from "$lib/api/types";
import { isSemver } from "$lib/logic/semver";

export interface FirmwareDraft {
	deviceModel: string;
	fwVersion: string;
	fwChecksum: string;
	binaryUrl: string;
}

export type FirmwareField = keyof FirmwareDraft;

export const EMPTY_FIRMWARE_DRAFT: FirmwareDraft = {
	deviceModel: "",
	fwVersion: "",
	fwChecksum: "",
	binaryUrl: "",
};

const CHECKSUM_PATTERN = /^[0-9a-fA-F]{64}$/;

/** sha256 принимается в любом регистре, а отправляется нижним регистром. */
export function normalizeChecksum(value: string): string {
	return value.trim().toLowerCase();
}

/** Абсолютный адрес прямой ссылки на бинарник: только http и https. */
export function isAbsoluteBinaryUrl(value: string): boolean {
	try {
		const url = new URL(value.trim());
		return url.protocol === "http:" || url.protocol === "https:";
	} catch {
		return false;
	}
}

export function validateFirmwareDraft(
	draft: FirmwareDraft,
): Partial<Record<FirmwareField, string>> {
	const errors: Partial<Record<FirmwareField, string>> = {};
	const model = draft.deviceModel.trim();
	const version = draft.fwVersion.trim();
	const checksum = normalizeChecksum(draft.fwChecksum);
	const binaryUrl = draft.binaryUrl.trim();

	if (model.length < 2 || model.length > 64) {
		errors.deviceModel = "От 2 до 64 символов";
	}
	if (!isSemver(version)) {
		errors.fwVersion = "Semver, например 1.1.0";
	}
	if (!CHECKSUM_PATTERN.test(checksum)) {
		errors.fwChecksum = "64 шестнадцатеричных символа";
	}
	if (!isAbsoluteBinaryUrl(binaryUrl)) {
		errors.binaryUrl = "Абсолютный адрес, например https://storage/firmware.bin";
	}

	return errors;
}

/** Тело запроса: ровно четыре ключа схемы, лишних полей сервер не принимает (§5.3). */
export function buildFirmwareBody(draft: FirmwareDraft): CreateFirmwareInput {
	return {
		device_model: draft.deviceModel.trim(),
		fw_version: draft.fwVersion.trim(),
		fw_checksum: normalizeChecksum(draft.fwChecksum),
		binary_url: draft.binaryUrl.trim(),
	};
}
