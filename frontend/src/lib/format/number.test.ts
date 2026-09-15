import { describe, expect, it } from "vitest";
import { int, percent, ratio } from "./number";

const THIN = "\u202f";

describe("int", () => {
	it("разделяет разряды тонким неразрывным пробелом", () => {
		expect(int(5000)).toBe(`5${THIN}000`);
		expect(int(1234567)).toBe(`1${THIN}234${THIN}567`);
	});

	it("не добавляет разделитель к коротким числам", () => {
		expect(int(100)).toBe("100");
		expect(int(0)).toBe("0");
	});

	it("отбрасывает дробную часть", () => {
		expect(int(12.7)).toBe("12");
	});

	it("не выдумывает значение для NaN и Infinity", () => {
		expect(int(Number.NaN)).toBe("не число");
		expect(int(Number.POSITIVE_INFINITY)).toBe("не число");
	});
});

describe("percent", () => {
	it("целая доля показывается без знаков после запятой", () => {
		expect(percent(0.95)).toBe("95%");
		expect(percent(1)).toBe("100%");
		expect(percent(0)).toBe("0%");
	});

	it("нецелая доля показывается с одним знаком", () => {
		expect(percent(0.9849)).toBe("98,5%");
		expect(percent(0.985)).toBe("98,5%");
	});

	it("округление до целого не скрывает, что значение не равно 100%", () => {
		expect(percent(0.9999)).toBe("100,0%");
	});

	it("артефакты двоичной арифметики не ломают правило целого", () => {
		// В JS 0.29 * 100 === 28.999999999999996, а 0.07 * 100 === 7.000000000000001.
		expect(Number.isInteger(0.29 * 100)).toBe(false);
		expect(Number.isInteger(0.07 * 100)).toBe(false);
		expect(percent(0.29)).toBe("29%");
		expect(percent(0.07)).toBe("7%");
		expect(percent(0.57)).toBe("57%");
	});

	it("явное число знаков переопределяет правило", () => {
		expect(percent(0.95, 2)).toBe("95,00%");
		expect(percent(0.9849, 0)).toBe("98%");
	});

	it("дробная часть отделяется запятой", () => {
		expect(percent(0.1234, 2)).toBe("12,34%");
	});
});

describe("ratio", () => {
	it("считает долю", () => {
		expect(ratio(12, 20)).toBeCloseTo(0.6, 10);
	});

	it("при нулевом знаменателе возвращает undefined, а не 0 и не NaN", () => {
		expect(ratio(5, 0)).toBeUndefined();
		expect(ratio(0, 0)).toBeUndefined();
	});

	it("не считает от нечисел", () => {
		expect(ratio(Number.NaN, 10)).toBeUndefined();
	});
});
