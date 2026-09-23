import type { Campaign, CampaignListItem, CampaignStatus, Stage, Stats } from "$lib/api/types";
import { int, percent } from "$lib/format/number";

export interface CampaignRow {
	id: string;
	model: string;
	version: string | undefined;
	status: CampaignStatus;
	createdAt: string;
	startedAt: string | undefined;
	completedAt: string | undefined;
	stages: Stage[];
	stats: Stats | undefined;
	/** Деталь кампании уже загружена: до этого метрики считать неизвестными. */
	detailLoaded: boolean;
}

export type AttentionReason = "below-threshold" | "thin-sample" | "paused" | "no-metrics";

export function toCampaignRow(
	item: CampaignListItem,
	detail: Campaign | undefined,
	version: string | undefined,
): CampaignRow {
	return {
		id: item.id,
		model: item.device_model,
		version,
		status: item.status,
		createdAt: item.created_at,
		startedAt: item.started_at,
		completedAt: item.completed_at,
		stages: detail?.rollout_stages ?? [],
		stats: detail?.stats,
		detailLoaded: detail !== undefined,
	};
}

export function activeStageOf(row: CampaignRow): Stage | undefined {
	return row.stages.find((stage) => stage.status === "active");
}

/**
 * Причина внимания по приоритету спеки: ниже порога, затем тонкая выборка,
 * затем пауза, затем отсутствие метрик. Одна строка - одна причина.
 */
export function attentionReason(row: CampaignRow): AttentionReason | undefined {
	if (row.status !== "running" && row.status !== "paused") return undefined;
	if (!row.detailLoaded) return undefined;

	const stage = activeStageOf(row);

	// При нулевой выборке процент успеха не измерение, поэтому сначала тонкая выборка.
	if (row.stats && stage && row.stats.sample_size < stage.min_sample_size) {
		return "thin-sample";
	}
	if (row.stats && stage && row.stats.success_rate < stage.success_threshold) {
		return "below-threshold";
	}
	if (row.status === "paused") return "paused";
	if (row.status === "running" && !row.stats) return "no-metrics";
	return undefined;
}

function label(row: CampaignRow): string {
	return row.version ? `${row.model} ${row.version}` : row.model;
}

export function attentionText(row: CampaignRow, reason: AttentionReason): string {
	const stage = activeStageOf(row);
	const stageName = stage
		? `${stage.order_index + 1} (${stage.target_percent}%)`
		: "активной стадии";

	if (reason === "below-threshold" && row.stats && stage) {
		return `${label(row)}: процент успеха ${percent(row.stats.success_rate)} ниже порога ${percent(stage.success_threshold)} на стадии ${stageName}.`;
	}
	if (reason === "thin-sample" && row.stats && stage) {
		return `${label(row)}: ${int(row.stats.sample_size)}/${int(stage.min_sample_size)} наблюдений`;
	}
	if (reason === "paused") return `${label(row)}: на паузе.`;
	return `${label(row)}: контроллер не отдаёт метрики.`;
}

/** running сначала, затем paused; внутри - по тяжести причины и свежести запуска. */
export function sortActiveRows(rows: CampaignRow[]): CampaignRow[] {
	const rank = (row: CampaignRow): number => {
		const reason = attentionReason(row);
		const base = row.status === "running" ? 0 : 100;
		if (reason === "below-threshold") return base + 0;
		if (reason === "thin-sample") return base + 1;
		return base + 2;
	};

	return [...rows].sort((a, b) => {
		const byRank = rank(a) - rank(b);
		if (byRank !== 0) return byRank;
		const aTime = a.startedAt ? Date.parse(a.startedAt) : 0;
		const bTime = b.startedAt ? Date.parse(b.startedAt) : 0;
		return bTime - aTime;
	});
}

/** Позиция активной стадии: `2 / 3`; без активной стадии - прочерк с причиной. */
export function progressOf(row: CampaignRow): { label: string; title?: string } {
	const total = row.stages.length;
	const active = activeStageOf(row);

	if (total === 0) return { label: "нет" };
	if (!active) {
		if (row.status === "completed") {
			return { label: `${total}/${total}` };
		}
		const failed = row.stages.find((stage) => stage.status === "failed");
		if (failed) {
			return {
				label: `${failed.order_index + 1}/${total}`,
				title: `Раскатка остановилась на стадии ${failed.order_index + 1} из ${total}`,
			};
		}
		const title = row.status === "draft" ? "Кампания не запущена" : "Активной стадии нет";
		return { label: "нет", title };
	}
	return { label: `${active.order_index + 1}/${total}` };
}

export function topModels(rows: CampaignRow[], count: number): { model: string; count: number }[] {
	const counts = new Map<string, number>();
	for (const row of rows) counts.set(row.model, (counts.get(row.model) ?? 0) + 1);

	return [...counts.entries()]
		.map(([model, value]) => ({ model, count: value }))
		.sort((a, b) => b.count - a.count || a.model.localeCompare(b.model, "ru"))
		.slice(0, count);
}

const RECENT_CHANGE_MS = 24 * 60 * 60 * 1000;

/**
 * Изменение за последний день: создание, старт, завершение или вход
 * в любую стадию. Блок внимания показывает только недавно менявшиеся
 * кампании, чтобы старые паузы не висели в нём вечно.
 */
export function hasRecentChange(row: CampaignRow, now: number = Date.now()): boolean {
	const candidates: (string | undefined)[] = [
		row.createdAt,
		row.startedAt,
		row.completedAt,
		...row.stages.map((stage) => stage.entered_at),
	];
	for (const iso of candidates) {
		if (!iso) continue;
		const time = Date.parse(iso);
		if (Number.isNaN(time)) continue;
		if (now - time <= RECENT_CHANGE_MS) return true;
	}
	return false;
}
