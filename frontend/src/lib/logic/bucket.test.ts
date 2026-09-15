import { describe, expect, it } from "vitest";
import { bucketOf, fnv1a32, isInBucket, isUuid, uuidToBytes } from "./bucket";

/**
 * Золотые значения посчитаны независимой реализацией FNV-1a на Python поверх
 * тех же 32 байт и совпали побайтово. Они закрепляют соответствие серверному
 * `UpdateService.calculateBucket`.
 */
const DEVICE_ID = "3f6d2d8e-4b1c-4a5f-9a7b-8e7d1c2a3b4c";
const CAMPAIGN_ID = "9b7f2a10-4c3d-4e5f-8a6b-1c2d3e4f5a6b";

const UUIDV7_DEVICE = "01990f2a-7c6b-7e11-9a3f-2b8c4d5e6f70";
const UUIDV7_CAMPAIGN = "01990f2a-9d1e-7a44-8b02-4c6f1a2b3c4d";

const NIL_UUID = "00000000-0000-0000-0000-000000000000";

describe("uuidToBytes", () => {
	it("раскладывает канонический UUID на 16 байт в порядке записи", () => {
		const bytes = uuidToBytes(DEVICE_ID);
		expect(bytes).toHaveLength(16);
		expect([...bytes].map((byte) => byte.toString(16).padStart(2, "0")).join("")).toBe(
			"3f6d2d8e4b1c4a5f9a7b8e7d1c2a3b4c",
		);
	});

	it("принимает запись без дефисов и даёт те же байты", () => {
		expect(uuidToBytes(DEVICE_ID.replaceAll("-", ""))).toEqual(uuidToBytes(DEVICE_ID));
	});

	it("бросает исключение на строке неверной длины", () => {
		expect(() => uuidToBytes("3f6d2d8e")).toThrow(/32 шестнадцатеричных/);
	});

	it("бросает исключение на не-шестнадцатеричных символах", () => {
		expect(() => uuidToBytes("zzzzzzzz-4b1c-4a5f-9a7b-8e7d1c2a3b4c")).toThrow(
			/не шестнадцатеричный UUID/,
		);
	});
});

describe("fnv1a32", () => {
	it("на пустом входе возвращает исходное смещение FNV-1a", () => {
		expect(fnv1a32(new Uint8Array(0))).toBe(0x811c9dc5);
	});

	it("возвращает беззнаковое 32-битное значение", () => {
		const value = fnv1a32(uuidToBytes(DEVICE_ID));
		expect(value).toBeGreaterThanOrEqual(0);
		expect(value).toBeLessThanOrEqual(0xffffffff);
	});

	it("совпадает с эталонным значением для пары UUID из комментария", () => {
		const bytes = new Uint8Array(32);
		bytes.set(uuidToBytes(DEVICE_ID), 0);
		bytes.set(uuidToBytes(CAMPAIGN_ID), 16);
		expect(fnv1a32(bytes)).toBe(2617095165);
	});
});

describe("bucketOf", () => {
	it("золотое значение: пара из комментария даёт бакет 66", () => {
		expect(bucketOf(DEVICE_ID, CAMPAIGN_ID)).toBe(66);
	});

	it("золотое значение: пара uuidv7 даёт бакет 29", () => {
		expect(bucketOf(UUIDV7_DEVICE, UUIDV7_CAMPAIGN)).toBe(29);
	});

	it("золотое значение: два нулевых UUID дают бакет 26", () => {
		expect(bucketOf(NIL_UUID, NIL_UUID)).toBe(26);
	});

	it("детерминирован: повторный вызов даёт то же число", () => {
		expect(bucketOf(DEVICE_ID, CAMPAIGN_ID)).toBe(bucketOf(DEVICE_ID, CAMPAIGN_ID));
	});

	it("не зависит от формата записи UUID", () => {
		expect(bucketOf(DEVICE_ID.replaceAll("-", ""), CAMPAIGN_ID)).toBe(
			bucketOf(DEVICE_ID, CAMPAIGN_ID),
		);
	});

	it("чувствителен к перестановке аргументов", () => {
		expect(bucketOf(CAMPAIGN_ID, DEVICE_ID)).not.toBe(bucketOf(DEVICE_ID, CAMPAIGN_ID));
	});

	it("всегда попадает в диапазон 1..100", () => {
		// Проверка распределения на синтетических идентификаторах, а не на выдуманных числах.
		for (let index = 0; index < 500; index += 1) {
			const hex = index.toString(16).padStart(32, "0");
			const bucket = bucketOf(hex, CAMPAIGN_ID);
			expect(bucket).toBeGreaterThanOrEqual(1);
			expect(bucket).toBeLessThanOrEqual(100);
		}
	});
});

describe("isInBucket", () => {
	it("бакет равен охвату - обновление выдаётся", () => {
		expect(isInBucket(50, 50)).toBe(true);
	});

	it("бакет больше охвата - обновление не выдаётся", () => {
		expect(isInBucket(51, 50)).toBe(false);
	});

	it("охват 100% пропускает любой бакет", () => {
		expect(isInBucket(100, 100)).toBe(true);
	});
});

describe("isUuid", () => {
	it("принимает каноническую запись", () => {
		expect(isUuid(DEVICE_ID)).toBe(true);
	});

	it("отклоняет произвольную строку", () => {
		expect(isUuid("demo-sensor-v1")).toBe(false);
	});
});
