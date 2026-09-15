import { describe, expect, it } from "vitest";
import {
	isOnTargetVersion,
	lastSeenMarker,
	LAST_SEEN_STALE_MS,
	TARGET_CHIP_TITLE,
} from "./device-row";
import { DEVICE_ONLINE_WINDOW_MS } from "./status";

const NOW = Date.parse("2026-09-12T12:00:00.000Z");
const iso = (offsetMs: number) => new Date(NOW - offsetMs).toISOString();

describe("маркер последней отметки", () => {
	it("свежая отметка меньше пяти минут - fresh", () => {
		expect(lastSeenMarker(iso(DEVICE_ONLINE_WINDOW_MS - 1000), NOW)).toBe("fresh");
		expect(lastSeenMarker(iso(0), NOW)).toBe("fresh");
	});

	it("от пяти минут до часа - recent, дальше часа - stale", () => {
		expect(lastSeenMarker(iso(DEVICE_ONLINE_WINDOW_MS + 1000), NOW)).toBe("recent");
		expect(lastSeenMarker(iso(LAST_SEEN_STALE_MS - 1000), NOW)).toBe("recent");
		expect(lastSeenMarker(iso(LAST_SEEN_STALE_MS + 1000), NOW)).toBe("stale");
	});

	it("без отметки и с неразобранной датой маркера нет: строка скажет «нет отметки»", () => {
		expect(lastSeenMarker(undefined, NOW)).toBeUndefined();
		expect(lastSeenMarker("не дата", NOW)).toBeUndefined();
	});
});

describe("чип «целевая»", () => {
	const targets = new Map([["demo-sensor-v1", "2.0.0"]]);

	it("ставка на версию выполняющейся кампании своей модели", () => {
		expect(isOnTargetVersion("demo-sensor-v1", "2.0.0", targets)).toBe(true);
		expect(isOnTargetVersion("demo-sensor-v1", "1.9.0", targets)).toBe(false);
	});

	it("модель без выполняющейся кампании чип не получает", () => {
		expect(isOnTargetVersion("demo-camera-z1", "2.0.0", targets)).toBe(false);
	});

	it("подсказка чипа формулирует причину", () => {
		expect(TARGET_CHIP_TITLE).toBe("устройство уже на версии раскатки");
	});
});
