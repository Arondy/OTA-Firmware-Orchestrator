/**
 * Производный индекс прошивок (00-CONTEXT §7.4).
 *
 * Зачем он нужен: элемент списка кампаний (`RolloutCampaignListItemResponse`)
 * несёт только `firmware_version_id` и не несёт строку версии. Чтобы показать
 * «demo-sensor-v1 1.4.2», идентификатор приходится разрешать через реестр
 * прошивок. Заодно отсюда берётся список моделей устройства - отдельного
 * эндпоинта с distinct-моделями в контракте нет.
 *
 * Индекс обходит `GET /api/v1/firmware?limit=100` не более чем по 5 страницам
 * и кэшируется на 5 минут. Если упёрлись в потолок страниц, `truncated`
 * становится `true` - интерфейс обязан это признать, а не делать вид, что
 * список моделей полный (§5.4).
 */
import { createResource, type Resource } from "./resource.svelte";
import { listFirmwareVersions } from "$lib/api/endpoints";
import type { FirmwareVersion } from "$lib/api/types";

export const FIRMWARE_INDEX_MAX_PAGES = 5;
export const FIRMWARE_INDEX_PAGE_LIMIT = 100;
export const FIRMWARE_INDEX_TTL_MS = 5 * 60 * 1000;

export interface FirmwareSnapshot {
	versions: FirmwareVersion[];
	/** `true`, если страниц могло быть больше, чем мы имеем право скачать. */
	truncated: boolean;
	/** Сколько страниц реально скачано. */
	pagesLoaded: number;
}

async function fetchSnapshot(signal: AbortSignal): Promise<FirmwareSnapshot> {
	const versions: FirmwareVersion[] = [];

	for (let page = 1; page <= FIRMWARE_INDEX_MAX_PAGES; page += 1) {
		const response = await listFirmwareVersions(
			{ page, limit: FIRMWARE_INDEX_PAGE_LIMIT },
			{ signal },
		);
		const rows = response.firmware_versions;
		versions.push(...rows);

		// Сервер не отдаёт total, поэтому признак конца страницы - неполная страница (§5.2).
		if (rows.length < FIRMWARE_INDEX_PAGE_LIMIT) {
			return { versions, truncated: false, pagesLoaded: page };
		}
		if (page === FIRMWARE_INDEX_MAX_PAGES) {
			return { versions, truncated: true, pagesLoaded: page };
		}
	}

	return { versions, truncated: false, pagesLoaded: 0 };
}

class FirmwareIndex {
	#resource: Resource<FirmwareSnapshot> = createResource<FirmwareSnapshot>(
		"firmware-index",
		fetchSnapshot,
	);

	#byId = $derived.by(() => {
		const map = new Map<string, FirmwareVersion>();
		for (const version of this.#resource.data?.versions ?? []) {
			map.set(version.id, version);
		}
		return map;
	});

	#modelList = $derived.by(() => {
		const models = new Set<string>();
		for (const version of this.#resource.data?.versions ?? []) {
			models.add(version.device_model);
		}
		return [...models].sort((a, b) => a.localeCompare(b, "ru"));
	});

	get versions(): FirmwareVersion[] {
		return this.#resource.data?.versions ?? [];
	}

	/** Индекс неполный: скачаны все разрешённые страницы, а данные ещё есть. */
	get truncated(): boolean {
		return this.#resource.data?.truncated ?? false;
	}

	get pagesLoaded(): number {
		return this.#resource.data?.pagesLoaded ?? 0;
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

	get lastUpdatedAt(): number | undefined {
		return this.#resource.lastUpdatedAt;
	}

	/** Строка версии прошивки по идентификатору, например `1.4.2`. */
	versionLabel(id: string | undefined): string | undefined {
		if (!id) return undefined;
		return this.#byId.get(id)?.fw_version;
	}

	/** Полная запись реестра по идентификатору. */
	get(id: string | undefined): FirmwareVersion | undefined {
		if (!id) return undefined;
		return this.#byId.get(id);
	}

	/**
	 * Список моделей устройства, собранный из реестра прошивок.
	 * Подпись в интерфейсе обязательна: «список собран из реестра прошивок»,
	 * потому что отдельного эндпоинта с моделями в контракте нет.
	 */
	models(): string[] {
		return this.#modelList;
	}

	/**
	 * Загрузить индекс, если он пуст или старше пяти минут. Повторные вызовы
	 * с разных экранов в пределах TTL не порождают новых запросов.
	 */
	async ensureLoaded(): Promise<void> {
		// Первичная загрузка уже идёт: второй запрос отменил бы её без пользы.
		if (this.#resource.pending) return;
		const updatedAt = this.#resource.lastUpdatedAt;
		if (this.#resource.data && updatedAt && Date.now() - updatedAt < FIRMWARE_INDEX_TTL_MS) {
			return;
		}
		await this.#resource.refresh();
	}

	/** Принудительно перечитать реестр (нужно после регистрации прошивки). */
	async refresh(): Promise<void> {
		await this.#resource.refresh();
	}

	destroy(): void {
		this.#resource.destroy();
	}
}

/** Общий индекс на приложение: несколько экранов делят один кэш. */
export const firmwareIndex = new FirmwareIndex();
