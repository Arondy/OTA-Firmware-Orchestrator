import type { Campaign } from "$lib/api/types";
import { ApiError } from "$lib/api/errors";
import { getRolloutCampaign } from "$lib/api/endpoints";
import { createResource, type Resource } from "./resource.svelte";
import { apiAvailability } from "./api-availability.svelte";

const POLL_INTERVAL_MS = 30_000;

/**
 * Агрегатор деталей кампаний: один ресурс на экран вместо опроса каждой строки
 * (§7.2). Обзор держит в опросе не больше 20 кампаний с ограничением в 8
 * одновременных запросов; таблица кампаний догружает стадии лениво по видимости
 * строк и не попадает в опрос вовсе. Интервал общий с обзором - 30 секунд.
 */
class CampaignDetails {
	// Record вместо Map: прокси `$state` надёжно отслеживает добавление ключей,
	// и ячейки таблиц перечитываются после догрузки стадии.
	#details = $state<Record<string, Campaign>>({});
	#inFlight = new Set<string>();
	#polled: string[] = [];
	#once: string[] = [];

	// Доступность API отмечается на каждом подзапросе отдельно (см. #sync),
	// поэтому обёртка сама флаг не трогает: её «успешный» пустой результат
	// при тотальном сбое не должен снимать баннер «API недоступен».
	#resource: Resource<Campaign[]> = createResource("campaign-details", () => this.#sync(), {
		intervalMs: POLL_INTERVAL_MS,
		notifyAvailability: false,
	});

	async #sync(): Promise<Campaign[]> {
		const wanted = [
			...this.#polled,
			...this.#once.filter((id) => this.#details[id] === undefined),
		].filter((id) => !this.#inFlight.has(id));

		const fetched: Campaign[] = [];
		let firstFailure: ApiError | undefined;
		for (const batch of chunk(wanted, 8)) {
			const results = await Promise.all(
				batch.map(async (id) => {
					this.#inFlight.add(id);
					try {
						const campaign = await getRolloutCampaign(id);
						apiAvailability.noteSuccess(Date.now());
						return campaign;
					} catch (error) {
						if (error instanceof ApiError) {
							apiAvailability.noteFailure(error, Date.now());
							firstFailure ??= error;
						}
						return undefined;
					} finally {
						this.#inFlight.delete(id);
					}
				}),
			);
			for (const campaign of results) {
				if (!campaign) continue;
				this.#details[campaign.id] = campaign;
				fetched.push(campaign);
			}
		}

		// Тотальный сбой пробрасывается наружу: обёртка пометит ресурс ошибкой
		// и сохранит опрос, а кэш #details останется на экране. Разрешённый
		// пустой результат («нечего опрашивать») ошибкой не считается.
		if (firstFailure && wanted.length > 0 && fetched.length === 0) {
			throw firstFailure;
		}
		return fetched;
	}

	/** Идентификаторы, которые нужно держать свежими (обзор). */
	setPolled(ids: string[]): void {
		this.#polled = ids;
	}

	/** Одноразовая догрузка (стадии для таблицы); уже загруженные не трогает. */
	ensure(ids: string[]): void {
		const missing = ids.filter((id) => this.#details[id] === undefined && !this.#inFlight.has(id));
		if (missing.length === 0) return;
		this.#once = [...this.#once, ...missing];
		void this.#resource.refresh();
	}

	get(id: string): Campaign | undefined {
		return this.#details[id];
	}

	get error() {
		return this.#resource.error;
	}

	get refreshing() {
		return this.#resource.refreshing;
	}

	async refresh(): Promise<void> {
		await this.#resource.refresh();
	}
}

function chunk<T>(items: T[], size: number): T[][] {
	const batches: T[][] = [];
	for (let index = 0; index < items.length; index += size) {
		batches.push(items.slice(index, index + size));
	}
	return batches;
}

export const campaignDetails = new CampaignDetails();
