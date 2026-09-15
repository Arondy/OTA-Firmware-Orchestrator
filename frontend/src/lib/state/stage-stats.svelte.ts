/**
 * Снимки метрик стадий за сеанс просмотра.
 *
 * API отдаёт `stats` только для активной стадии кампании: сколько устройств
 * попало в выборку и какой процент успеха набрала прошедшая стадия, из ответа
 * не узнать. Пока экран кампании открыт, он опрашивается каждые 3 секунды, и
 * каждый снимок метрик активной стадии сохраняется здесь - так «История
 * переходов» может показать фактические выборку и процент успеха для стадий,
 * которые успели смениться при операторе.
 *
 * После перезагрузки страницы накопленное теряется: для уже завершённых стадий
 * история показывает требуемые показатели (минимальную выборку и порог), а не
 * выдуманные наблюдения (§10.4: никаких фактов, которых нет в данных).
 */
import type { Campaign } from "$lib/api/types";

export interface StageStatsSnapshot {
	sampleSize: number;
	successRate: number;
}

class StageStatsHistory {
	// campaignId -> stageId -> снимок. Record, а не Map: прокси `$state`
	// отслеживает добавление ключей, и таймлайн перечитывается после записи.
	#snapshots = $state<Record<string, Record<string, StageStatsSnapshot>>>({});

	/** Записать метрики активной стадии из ответа кампании, если они есть. */
	record(campaign: Campaign): void {
		const stats = campaign.stats;
		if (!stats) return;
		const byCampaign = this.#snapshots[campaign.id] ?? {};
		const previous = byCampaign[stats.active_stage_id];
		if (
			previous &&
			previous.sampleSize === stats.sample_size &&
			previous.successRate === stats.success_rate
		) {
			return;
		}
		this.#snapshots[campaign.id] = {
			...byCampaign,
			[stats.active_stage_id]: {
				sampleSize: stats.sample_size,
				successRate: stats.success_rate,
			},
		};
	}

	/** Снимок метрик стадии, если он наблюдался в этом сеансе. */
	get(campaignId: string, stageId: string): StageStatsSnapshot | undefined {
		return this.#snapshots[campaignId]?.[stageId];
	}

	/** Все снимки кампании: карта stageId -> метрики. */
	forCampaign(campaignId: string): Record<string, StageStatsSnapshot> {
		return this.#snapshots[campaignId] ?? {};
	}
}

export const stageStats = new StageStatsHistory();
