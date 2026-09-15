/**
 * История переходов кампании.
 *
 * Собирается только из полей, которые реально отдаёт
 * `GET /api/v1/campaigns/{id}`: моменты создания, старта, завершения и
 * `entered_at` стадий. Событий, которых API не поддерживает (решения
 * оценщика, попытки обновлений), здесь нет и не может быть.
 */
import type { Campaign } from "$lib/api/types";
import { int, percent } from "$lib/format/number";
import type { Tone } from "./status";
import { STAGE_STATUS } from "./status";

/**
 * Снимок метрик стадии, наблюдавшийся клиентом (см. `state/stage-stats`):
 * API отдаёт `stats` только для активной стадии, поэтому фактические выборка
 * и процент успеха прошедших стадий известны, только если оператор видел их
 * переход вживую.
 */
export interface StageStatsSnapshot {
	sampleSize: number;
	successRate: number;
}

export interface TimelineEntry {
	/** Уникален в пределах истории: ключ сортировки и ключ рендера. */
	key: string;
	title: string;
	tone: Tone;
	/** Момент события в ISO; `undefined`, когда API его не отдаёт. */
	at: string | undefined;
	/** Пояснение вторым рядом, когда без него запись неоднозначна. */
	note?: string;
}

/** Номер стадии для оператора: внутренний `order_index` начинается с нуля. */
export function stageNumber(orderIndex: number): number {
	return orderIndex + 1;
}

export function buildTimeline(
	campaign: Campaign,
	statsByStage: Record<string, StageStatsSnapshot> = {},
): TimelineEntry[] {
	const entries: TimelineEntry[] = [];
	const stages = [...campaign.rollout_stages].sort((a, b) => a.order_index - b.order_index);

	/** Требовательные показатели стадии: то, что API отдаёт всегда. */
	const requirements = (stage: Campaign["rollout_stages"][number]): string =>
		`мин. выборка ${int(stage.min_sample_size)}, порог успеха ${percent(stage.success_threshold)}`;

	const statsOf = (stageId: string): StageStatsSnapshot | undefined => {
		const recorded = statsByStage[stageId];
		if (recorded) return recorded;
		const live = campaign.stats;
		if (live && live.active_stage_id === stageId) {
			return { sampleSize: live.sample_size, successRate: live.success_rate };
		}
		return undefined;
	};

	entries.push({
		key: "created",
		title: "Кампания создана",
		tone: "neutral",
		at: campaign.created_at,
	});

	if (campaign.started_at) {
		const first = stages[0];
		entries.push({
			key: "started",
			title: "Старт раскатки",
			tone: "accent",
			at: campaign.started_at,
			note:
				first === undefined
					? undefined
					: `стадия ${stageNumber(first.order_index)} (${first.target_percent}%): ${requirements(first)}`,
		});
	}

	for (const stage of stages) {
		if (!stage.entered_at) continue;
		const snapshot = statsOf(stage.id);
		entries.push({
			key: `stage-${stage.id}`,
			title: `Стадия ${stageNumber(stage.order_index)} перешла в статус «${STAGE_STATUS[stage.status].label}»`,
			tone: STAGE_STATUS[stage.status].tone,
			at: stage.entered_at,
			// Фактические метрики - если стадия наблюдалась живой; иначе её
			// требуемые показатели (выдуманных наблюдений быть не должно).
			note: snapshot
				? `выборка ${int(snapshot.sampleSize)}, процент успеха ${percent(snapshot.successRate)}`
				: requirements(stage),
		});
	}

	if (campaign.completed_at || campaign.status === "completed") {
		if (campaign.status === "rolled_back") {
			entries.push({
				key: "finished",
				title: "Кампания откачена",
				tone: "danger",
				at: campaign.completed_at,
				note:
					campaign.completed_at === undefined
						? "точное время применения решения API не возвращает"
						: undefined,
			});
		} else if (campaign.completed_at) {
			entries.push({
				key: "finished",
				title: "Раскатка завершена",
				tone: "success",
				at: campaign.completed_at,
			});
		}
	} else if (campaign.status === "rolled_back") {
		// Откачена, но `completed_at` не пришёл: событие показываем без времени.
		entries.push({
			key: "finished",
			title: "Кампания откачена",
			tone: "danger",
			at: undefined,
			note: "точное время применения решения API не возвращает",
		});
	}

	return entries
		.filter((entry) => entry.at !== undefined || entry.key === "finished")
		.sort((a, b) => {
			if (a.at === undefined || b.at === undefined) return 0;
			return a.at.localeCompare(b.at);
		});
}
