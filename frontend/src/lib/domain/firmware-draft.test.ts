import { describe, expect, it } from "vitest";
import {
	buildFirmwareBody,
	isAbsoluteBinaryUrl,
	normalizeChecksum,
	validateFirmwareDraft,
	type FirmwareDraft,
} from "./firmware-draft";

const VALID: FirmwareDraft = {
	deviceModel: "demo-sensor-v1",
	fwVersion: "1.2.3",
	fwChecksum: "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
	binaryUrl: "https://firmware.internal/demo-sensor-v1/1.2.3.bin",
};

describe("нормализация контрольной суммы", () => {
	it("регистр не важен: перед отправкой сумма нижним регистром", () => {
		expect(
			normalizeChecksum("9F86D081884C7D659A2FEAA0C55AD015A3BF4F1B2B0B822CD15D6C15B0F00A08"),
		).toBe(VALID.fwChecksum);
		expect(normalizeChecksum("  abc  ")).toBe("abc");
	});
});

describe("абсолютность ссылки", () => {
	it("http и https приняты, относительные и прочие схемы отклонены", () => {
		expect(isAbsoluteBinaryUrl("https://a/b.bin")).toBe(true);
		expect(isAbsoluteBinaryUrl("http://a/b.bin")).toBe(true);
		expect(isAbsoluteBinaryUrl("/firmware/1.bin")).toBe(false);
		expect(isAbsoluteBinaryUrl("ftp://a/b.bin")).toBe(false);
		expect(isAbsoluteBinaryUrl("не ссылка")).toBe(false);
	});
});

describe("валидация черновика", () => {
	it("корректный черновик без ошибок", () => {
		expect(validateFirmwareDraft(VALID)).toEqual({});
	});

	it("модель короче 2 и длиннее 64 отклоняется", () => {
		expect(validateFirmwareDraft({ ...VALID, deviceModel: "a" }).deviceModel).toBeDefined();
		expect(
			validateFirmwareDraft({ ...VALID, deviceModel: "x".repeat(65) }).deviceModel,
		).toBeDefined();
	});

	it("semver проверяется тем же выражением, что примет сервер", () => {
		expect(validateFirmwareDraft({ ...VALID, fwVersion: "1.0" }).fwVersion).toBeDefined();
		expect(validateFirmwareDraft({ ...VALID, fwVersion: "v1.0.0" }).fwVersion).toBeDefined();
		expect(validateFirmwareDraft({ ...VALID, fwVersion: "01.0.0" }).fwVersion).toBeDefined();
		expect(validateFirmwareDraft({ ...VALID, fwVersion: "1.2.3-rc.1" }).fwVersion).toBeUndefined();
		expect(
			validateFirmwareDraft({ ...VALID, fwVersion: "1.2.3+build.5" }).fwVersion,
		).toBeUndefined();
	});

	it("сумма длиной не 64 или не hex отклоняется", () => {
		expect(validateFirmwareDraft({ ...VALID, fwChecksum: "abc" }).fwChecksum).toBeDefined();
		expect(
			validateFirmwareDraft({ ...VALID, fwChecksum: "z".repeat(64) }).fwChecksum,
		).toBeDefined();
	});
});

describe("тело запроса", () => {
	it("ровно четыре ключа схемы, сумма нормализована", () => {
		const body = buildFirmwareBody({ ...VALID, fwChecksum: VALID.fwChecksum.toUpperCase() });
		expect(Object.keys(body).sort()).toEqual([
			"binary_url",
			"device_model",
			"fw_checksum",
			"fw_version",
		]);
		expect(body.fw_checksum).toBe(VALID.fwChecksum);
	});
});
