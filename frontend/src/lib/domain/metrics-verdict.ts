/**
 * Вывод о стадии по метрикам контроллера. Три взаимно исключающих исхода.
 */
import type { Tone } from "./status";
import { plural } from "$lib/format/number";

export interface MetricsVerdict {
	text: string;
	tone: Tone;
}

const FORMS: [string, string, string] = ["наблюдение", "наблюдения", "наблюдений"];

export function metricsVerdict(params: {
	sampleSize: number;
	minSampleSize: number;
	successRate: number;
	threshold: number;
}): MetricsVerdict {
	const { sampleSize, minSampleSize, successRate, threshold } = params;

	if (sampleSize < minSampleSize) {
		return {
			tone: "neutral",
			text: `Набор выборки: решение будет принято после ${minSampleSize} ${plural(minSampleSize, FORMS)}.`,
		};
	}

	if (successRate >= threshold) {
		return {
			tone: "success",
			text: "Процент успеха выше порога. Стадия может быть продвинута автоматически.",
		};
	}

	return {
		tone: "danger",
		text: "Процент успеха ниже порога. Возможен автоматический откат.",
	};
}
