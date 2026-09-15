import { listRolloutCampaigns } from "$lib/api/endpoints";
import type { CampaignListItem } from "$lib/api/types";
import { createResource, type Resource } from "./resource.svelte";

const PAGE_LIMIT = 100;
const MAX_PAGES = 5;

/**
 * Коллекция кампаний: обход до пяти страниц по 100 записей с остановкой на
 * первой неполной, потому что сервер не отдаёт общее число (§5.2). Список не
 * опрашивается: обновление ручное и после мутаций (§7.2).
 */
class CampaignCollection {
	#loadedCount = $state(0);
	#walking = $state(false);

	#resource: Resource<CampaignListItem[]> = createResource("campaigns-collection", () =>
		this.#walk(),
	);

	async #walk(): Promise<CampaignListItem[]> {
		const rows: CampaignListItem[] = [];
		this.#walking = true;
		this.#loadedCount = 0;

		try {
			for (let page = 1; page <= MAX_PAGES; page += 1) {
				const response = await listRolloutCampaigns({ page, limit: PAGE_LIMIT });
				rows.push(...response.rollout_campaigns);
				this.#loadedCount = rows.length;
				if (response.rollout_campaigns.length < PAGE_LIMIT) break;
			}
			return rows;
		} finally {
			this.#walking = false;
		}
	}

	get rows(): CampaignListItem[] {
		return this.#resource.data ?? [];
	}

	/** Сколько загружено на текущий момент: видно только пока идёт обход. */
	get loadedCount(): number {
		return this.#loadedCount;
	}

	get walking(): boolean {
		return this.#walking;
	}

	get pending(): boolean {
		return this.#resource.pending;
	}

	get refreshing(): boolean {
		return this.#resource.refreshing;
	}

	get error() {
		return this.#resource.error;
	}

	get lastUpdatedAt() {
		return this.#resource.lastUpdatedAt;
	}

	async refresh(): Promise<void> {
		await this.#resource.refresh();
	}
}

export const campaignCollection = new CampaignCollection();
