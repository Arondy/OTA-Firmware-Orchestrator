import type { CampaignRow } from "./campaign-row";
import { shortId } from "$lib/format/id";

/**
 * Фильтры списка кампаний. Отдельного поля «модель» нет: поиск и так
 * сопоставляет запрос с моделью, версией и коротким id.
 */
export interface CampaignFilters {
	status: string;
	query: string;
}

export const EMPTY_FILTERS: CampaignFilters = { status: "", query: "" };

export function hasActiveFilters(filters: CampaignFilters): boolean {
	return Boolean(filters.status || filters.query.trim());
}

/**
 * Все фильтры клиентские: у API нет фильтра по статусу кампаний, а серверный
 * фильтр модели есть только у устройств и прошивок (§5.1).
 */
export function filterRows(rows: CampaignRow[], filters: CampaignFilters): CampaignRow[] {
	const query = filters.query.trim().toLowerCase();

	return rows.filter((row) => {
		if (filters.status && row.status !== filters.status) return false;
		if (!query) return true;
		return (
			row.model.toLowerCase().includes(query) ||
			(row.version ?? "").toLowerCase().includes(query) ||
			shortId(row.id).toLowerCase().includes(query)
		);
	});
}

export function paginateRows<T>(rows: T[], page: number, limit: number): T[] {
	const start = (page - 1) * limit;
	return rows.slice(start, start + limit);
}

export interface SortSpec {
	id: string;
	desc: boolean;
}

/** Одиночная сортировка загруженных строк; доступники задаёт экран. */
export function sortRows<T>(
	rows: T[],
	sorting: SortSpec[],
	accessors: Record<string, (row: T) => string | number>,
): T[] {
	const spec = sorting[0];
	if (!spec) return rows;

	const accessor = accessors[spec.id];
	if (!accessor) return rows;

	return [...rows].sort((a, b) => {
		const left = accessor(a);
		const right = accessor(b);
		const cmp = left < right ? -1 : left > right ? 1 : 0;
		return spec.desc ? -cmp : cmp;
	});
}
