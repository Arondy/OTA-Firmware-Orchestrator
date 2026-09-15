import { describe, expect, it } from "vitest";
import { mulberry32, pickBatchResult } from "./batch-prng";

describe("mulberry32", () => {
	it("один seed воспроизводит последовательность полностью", () => {
		const first = Array.from({ length: 50 }, mulberry32(20260913));
		const second = Array.from({ length: 50 }, mulberry32(20260913));
		expect(first).toEqual(second);
	});

	it("разные seed дают разные последовательности", () => {
		const first = Array.from({ length: 20 }, mulberry32(1));
		const second = Array.from({ length: 20 }, mulberry32(2));
		expect(first).not.toEqual(second);
	});

	it("значения лежат в [0, 1)", () => {
		const rng = mulberry32(7);
		for (let index = 0; index < 500; index += 1) {
			const value = rng();
			expect(value).toBeGreaterThanOrEqual(0);
			expect(value).toBeLessThan(1);
		}
	});
});

describe("pickBatchResult", () => {
	it("доля 100 даёт только успех, доля 0 только ошибку", () => {
		const always = mulberry32(11);
		for (let index = 0; index < 100; index += 1) {
			expect(pickBatchResult(always, 100)).toBe("success");
		}
		const never = mulberry32(11);
		for (let index = 0; index < 100; index += 1) {
			expect(pickBatchResult(never, 0)).toBe("failure");
		}
	});

	it("исходы воспроизводимы по seed: прогон с тем же seed даёт ту же смесь", () => {
		const run = (seed: number) => {
			const rng = mulberry32(seed);
			return Array.from({ length: 40 }, () => pickBatchResult(rng, 70));
		};
		expect(run(42)).toEqual(run(42));
		expect(run(42).filter((result) => result === "success").length).toBeGreaterThan(20);
	});
});
