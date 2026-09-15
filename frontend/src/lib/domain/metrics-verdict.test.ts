import { describe, expect, it } from "vitest";
import { metricsVerdict } from "./metrics-verdict";

describe("metricsVerdict", () => {
	it("выборка не набрана: решение отложено, тон нейтральный", () => {
		const verdict = metricsVerdict({
			sampleSize: 12,
			minSampleSize: 20,
			successRate: 0.4,
			threshold: 0.95,
		});
		expect(verdict.tone).toBe("neutral");
		expect(verdict.text).toBe("Набор выборки: решение будет принято после 20 наблюдений.");
	});

	it("процент выше порога: стадия может быть продвинута", () => {
		const verdict = metricsVerdict({
			sampleSize: 40,
			minSampleSize: 20,
			successRate: 0.97,
			threshold: 0.95,
		});
		expect(verdict.tone).toBe("success");
		expect(verdict.text).toBe(
			"Процент успеха выше порога. Стадия может быть продвинута автоматически.",
		);
	});

	it("процент ниже порога: возможен автоматический откат", () => {
		const verdict = metricsVerdict({
			sampleSize: 40,
			minSampleSize: 20,
			successRate: 0.94,
			threshold: 0.95,
		});
		expect(verdict.tone).toBe("danger");
		expect(verdict.text).toBe("Процент успеха ниже порога. Возможен автоматический откат.");
	});

	it("граница порога относится к успеху", () => {
		const verdict = metricsVerdict({
			sampleSize: 20,
			minSampleSize: 20,
			successRate: 0.95,
			threshold: 0.95,
		});
		expect(verdict.tone).toBe("success");
	});
});
