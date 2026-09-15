process.env.TZ = "UTC";

import { describe, expect, it } from "vitest";
import {
	UNPARSEABLE_DATE_LABEL,
	absoluteDateTime,
	absoluteDateTimeTitle,
	absoluteTime,
	relativeTime,
} from "./datetime";

const NOW = Date.parse("2026-09-12T12:00:00.000Z");

function iso(offsetMs: number): string {
	return new Date(NOW - offsetMs).toISOString();
}

const SECOND = 1000;
const MINUTE = 60 * SECOND;
const HOUR = 60 * MINUTE;

describe("relativeTime", () => {
	it("совсем свежее значение - «только что»", () => {
		expect(relativeTime(iso(0), NOW)).toBe("только что");
		expect(relativeTime(iso(9 * SECOND), NOW)).toBe("только что");
	});

	it("секунды", () => {
		expect(relativeTime(iso(12 * SECOND), NOW)).toBe("12 с назад");
		expect(relativeTime(iso(59 * SECOND), NOW)).toBe("59 с назад");
	});

	it("минуты", () => {
		expect(relativeTime(iso(MINUTE), NOW)).toBe("1 мин назад");
		expect(relativeTime(iso(5 * MINUTE), NOW)).toBe("5 мин назад");
		expect(relativeTime(iso(59 * MINUTE), NOW)).toBe("59 мин назад");
	});

	it("часы в пределах тех же суток", () => {
		expect(relativeTime(iso(2 * HOUR), NOW)).toBe("2 ч назад");
	});

	it("предыдущий календарный день - «вчера»", () => {
		expect(relativeTime("2026-09-11T22:00:00.000Z", NOW)).toBe("вчера");
	});

	it("старые значения переходят на абсолютный формат", () => {
		expect(relativeTime("2026-09-01T08:00:00.000Z", NOW)).toBe("1 сен 2026, 08:00");
	});

	it("небольшой перекос часов в будущее не показывает отрицательное время", () => {
		expect(relativeTime(iso(-30 * SECOND), NOW)).toBe("только что");
	});

	it("большое значение из будущего показывается как есть, а не «только что»", () => {
		expect(relativeTime(iso(-2 * HOUR), NOW)).toBe("12 сен 2026, 14:00");
	});

	it("неразобранная дата не маскируется под время", () => {
		expect(relativeTime("не дата", NOW)).toBe(UNPARSEABLE_DATE_LABEL);
		expect(relativeTime(undefined, NOW)).toBe(UNPARSEABLE_DATE_LABEL);
	});
});

describe("абсолютные форматы", () => {
	it("absoluteTime даёт часы и минуты", () => {
		expect(absoluteTime("2026-09-12T14:32:00.000Z")).toBe("14:32");
	});

	it("absoluteDateTime соответствует формату из словаря", () => {
		expect(absoluteDateTime("2026-09-12T14:32:00.000Z")).toBe("12 сен 2026, 14:32");
	});

	it("в title есть и локальное время, и исходный момент UTC", () => {
		const title = absoluteDateTimeTitle("2026-09-12T14:32:00.000Z");
		expect(title).toContain("12 сен 2026, 14:32");
		expect(title).toContain("2026-09-12T14:32:00.000Z");
	});

	it("неразобранная дата подписана честно", () => {
		expect(absoluteDateTime("мусор")).toBe(UNPARSEABLE_DATE_LABEL);
		expect(absoluteTime(undefined)).toBe(UNPARSEABLE_DATE_LABEL);
	});

	it("месяцы сокращаются по словарю, без точки", () => {
		const months = [
			"янв",
			"фев",
			"мар",
			"апр",
			"май",
			"июн",
			"июл",
			"авг",
			"сен",
			"окт",
			"ноя",
			"дек",
		];
		months.forEach((name, index) => {
			const value = `2026-${String(index + 1).padStart(2, "0")}-05T10:00:00.000Z`;
			expect(absoluteDateTime(value)).toBe(`5 ${name} 2026, 10:00`);
		});
	});
});
