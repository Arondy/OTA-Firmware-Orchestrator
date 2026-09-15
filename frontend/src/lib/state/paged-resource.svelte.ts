/**
 * Постраничный ресурс (00-CONTEXT §7.2, §7.5).
 *
 * Обёртка над `createResource`, которая добавляет то, что разрешает контракт:
 * номер страницы, размер страницы, фильтры точного совпадения и признак
 * «есть ли ещё страницы». Сервер не отдаёт `total`, поэтому `hasMore` считается
 * единственно честным способом - `rows.length === limit`, а пейджер умеет только
 * «Назад» и «Дальше». Общего числа страниц здесь не появляется ни в каком виде.
 *
 * Состояние таблицы, которым стоит делиться, живёт в адресной строке
 * (`?model=&status=&page=&limit=`) и пишется через `goto(..., { replaceState: true,
 * keepFocus: true, noScroll: true })`, чтобы не ломать фокус и не дёргать скролл.
 */
import { goto } from "$app/navigation";
import { page as appPage } from "$app/state";
import { createResource, type Resource } from "./resource.svelte";
import type { ApiError } from "$lib/api/errors";

/** Серверный потолок размера страницы (DB_PAGINATION_LIMIT в .env.example). */
export const SERVER_MAX_LIMIT = 100;
export const DEFAULT_LIMIT = 100;
export const MIN_LIMIT = 1;

export type Filters = Record<string, string | undefined>;

export interface PagedFetchArgs<TFilters extends Filters> {
	page: number;
	limit: number;
	filters: TFilters;
	signal: AbortSignal;
}

export interface PagedResourceOptions<TItem, TFilters extends Filters> {
	key: string;
	fetch: (args: PagedFetchArgs<TFilters>) => Promise<TItem[]>;
	initialFilters: TFilters;
	/** Размер страницы по умолчанию; усекается до `SERVER_MAX_LIMIT`. */
	limit?: number;
	/** Держать состояние в адресной строке. По умолчанию включено. */
	syncUrl?: boolean;
	/**
	 * Имена query-параметров. Ключи - имена полей фильтров плюс служебные
	 * `page` и `limit`; значения - имена параметров в URL. По умолчанию имя
	 * параметра совпадает с именем поля.
	 */
	urlKeys?: Record<string, string>;
}

export interface PagedResource<TItem, TFilters extends Filters> {
	readonly rows: TItem[];
	readonly page: number;
	readonly limit: number;
	readonly filters: TFilters;
	/** `rows.length === limit`: единственно доступная оценка без `total` (§5.2). */
	readonly hasMore: boolean;
	readonly canGoBack: boolean;
	readonly pending: boolean;
	readonly refreshing: boolean;
	readonly error: ApiError | undefined;
	readonly lastUpdatedAt: number | undefined;
	/** Сколько фильтров задано: для подписи «сбросить фильтры». */
	readonly activeFilterCount: number;
	next(): Promise<void>;
	prev(): Promise<void>;
	setPage(page: number): Promise<void>;
	setLimit(limit: number): Promise<void>;
	setFilters(patch: Partial<TFilters>): Promise<void>;
	resetFilters(): Promise<void>;
	/** Точечная правка загруженных строк без полного перезапроса (§7.6). */
	update(updater: (rows: TItem[]) => TItem[]): void;
	refresh(): Promise<void>;
	destroy(): void;
}

function clampPage(value: number): number {
	if (!Number.isFinite(value)) return 1;
	return Math.max(1, Math.trunc(value));
}

function clampLimit(value: number): number {
	if (!Number.isFinite(value)) return DEFAULT_LIMIT;
	return Math.min(SERVER_MAX_LIMIT, Math.max(MIN_LIMIT, Math.trunc(value)));
}

function paramKey(urlKeys: Record<string, string> | undefined, name: string): string {
	return urlKeys?.[name] ?? name;
}

function parsePositiveInt(raw: string | null): number | undefined {
	if (raw === null || raw.trim().length === 0) return undefined;
	const value = Number.parseInt(raw, 10);
	return Number.isFinite(value) ? value : undefined;
}

function readFromUrl<TFilters extends Filters>(
	options: PagedResourceOptions<unknown, TFilters>,
): { page: number; limit: number; filters: TFilters } {
	const fallbackLimit = clampLimit(options.limit ?? DEFAULT_LIMIT);
	const base = { page: 1, limit: fallbackLimit, filters: { ...options.initialFilters } };

	if (typeof window === "undefined") return base;

	const params = appPage.url.searchParams;
	// Обход идёт по расширенной записи: точечное присваивание в обобщённый индекс
	// TypeScript доказать не может, а одно приведение на границе остаётся честным.
	const filters: Record<string, string | undefined> = { ...base.filters };
	for (const name of Object.keys(base.filters)) {
		const raw = params.get(paramKey(options.urlKeys, name));
		filters[name] = raw === null || raw.length === 0 ? undefined : raw;
	}

	return {
		page: clampPage(parsePositiveInt(params.get(paramKey(options.urlKeys, "page"))) ?? 1),
		limit: clampLimit(
			parsePositiveInt(params.get(paramKey(options.urlKeys, "limit"))) ?? fallbackLimit,
		),
		filters: filters as TFilters,
	};
}

export function createPagedResource<TItem, TFilters extends Filters>(
	options: PagedResourceOptions<TItem, TFilters>,
): PagedResource<TItem, TFilters> {
	const syncUrl = options.syncUrl ?? true;
	const initial = syncUrl
		? readFromUrl<TFilters>(options as PagedResourceOptions<unknown, TFilters>)
		: {
				page: 1,
				limit: clampLimit(options.limit ?? DEFAULT_LIMIT),
				filters: { ...options.initialFilters },
			};

	let currentPage = $state(initial.page);
	let currentLimit = $state(initial.limit);
	let currentFilters = $state<TFilters>(initial.filters);

	const resource: Resource<TItem[]> = createResource<TItem[]>(options.key, (signal) =>
		options.fetch({
			page: currentPage,
			limit: currentLimit,
			filters: { ...currentFilters },
			signal,
		}),
	);

	async function syncUrlState(): Promise<void> {
		if (!syncUrl || typeof window === "undefined") return;

		const params = new URLSearchParams();
		// В адресную строку пишутся только значения, отличающиеся от умолчаний.
		if (currentPage > 1) params.set(paramKey(options.urlKeys, "page"), String(currentPage));
		if (currentLimit !== DEFAULT_LIMIT) {
			params.set(paramKey(options.urlKeys, "limit"), String(currentLimit));
		}
		for (const [name, value] of Object.entries(currentFilters)) {
			if (typeof value === "string" && value.length > 0) {
				params.set(paramKey(options.urlKeys, name), value);
			}
		}

		const search = params.toString();
		await goto(`${appPage.url.pathname}${search.length > 0 ? `?${search}` : ""}`, {
			replaceState: true,
			keepFocus: true,
			noScroll: true,
		});
	}

	async function applyChange(): Promise<void> {
		await syncUrlState();
		await resource.refresh();
	}

	return {
		get rows(): TItem[] {
			return resource.data ?? [];
		},
		get page(): number {
			return currentPage;
		},
		get limit(): number {
			return currentLimit;
		},
		get filters(): TFilters {
			return { ...currentFilters };
		},
		get hasMore(): boolean {
			// Честная оценка без total: полная страница означает, что за ней может быть ещё.
			return (resource.data?.length ?? 0) === currentLimit;
		},
		get canGoBack(): boolean {
			return currentPage > 1;
		},
		get pending(): boolean {
			return resource.pending;
		},
		get refreshing(): boolean {
			return resource.refreshing;
		},
		get error(): ApiError | undefined {
			return resource.error;
		},
		get lastUpdatedAt(): number | undefined {
			return resource.lastUpdatedAt;
		},
		get activeFilterCount(): number {
			return Object.values(currentFilters).filter(
				(value) => typeof value === "string" && value.length > 0,
			).length;
		},

		async next(): Promise<void> {
			if (!this.hasMore) return;
			currentPage += 1;
			await applyChange();
		},
		async prev(): Promise<void> {
			if (currentPage <= 1) return;
			currentPage -= 1;
			await applyChange();
		},
		async setPage(page: number): Promise<void> {
			const next = clampPage(page);
			if (next === currentPage) return;
			currentPage = next;
			await applyChange();
		},
		async setLimit(limit: number): Promise<void> {
			const next = clampLimit(limit);
			if (next === currentLimit) return;
			currentLimit = next;
			// Размер страницы изменился - номер страницы теряет смысл.
			currentPage = 1;
			await applyChange();
		},
		async setFilters(patch: Partial<TFilters>): Promise<void> {
			currentFilters = { ...currentFilters, ...patch };
			currentPage = 1;
			await applyChange();
		},
		update(updater: (rows: TItem[]) => TItem[]): void {
			resource.write(updater(resource.data ?? []));
		},
		async resetFilters(): Promise<void> {
			currentFilters = { ...options.initialFilters };
			currentPage = 1;
			await applyChange();
		},
		async refresh(): Promise<void> {
			await resource.refresh();
		},
		destroy(): void {
			resource.destroy();
		},
	};
}
