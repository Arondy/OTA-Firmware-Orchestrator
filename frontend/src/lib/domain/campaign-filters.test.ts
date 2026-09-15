import { describe, expect, it } from "vitest";
import type { CampaignRow } from "./campaign-row";
import {
	EMPTY_FILTERS,
	filterRows,
	hasActiveFilters,
	paginateRows,
	sortRows,
	type CampaignFilters,
} from "./campaign-filters";

function row(overrides: Partial<CampaignRow> = {}): CampaignRow {
	return {
		id: "3f6d2d8e-4b1c-4a5f-9a7b-8e7d1c2a3b4c",
		model: "demo-sensor-v1",
		version: "2.0.0",
		status: "running",
		createdAt: "2026-09-12T08:00:00.000Z",
		startedAt: "2026-09-12T09:00:00.000Z",
		completedAt: undefined,
		stages: [],
		stats: undefined,
		detailLoaded: false,
		...overrides,
	};
}

const ROWS = [
	row({
		id: "3f6d2d8e-4b1c-4a5f-9a7b-8e7d1c2a3b4c",
		model: "demo-sensor-v1",
		status: "running",
		createdAt: "2026-09-12T08:00:00.000Z",
	}),
	row({
		id: "4f6d2d8e-4b1c-4a5f-9a7b-8e7d1c2a3b4d",
		model: "demo-sensor-v2",
		status: "paused",
		createdAt: "2026-09-12T09:00:00.000Z",
	}),
	row({
		id: "5f6d2d8e-4b1c-4a5f-9a7b-8e7d1c2a3b4e",
		model: "demo-gateway-x1",
		status: "draft",
		version: "3.1.0",
		createdAt: "2026-09-12T10:00:00.000Z",
	}),
];

const [FIRST, SECOND, THIRD] = ROWS;

describe("filterRows", () => {
	it("пустые фильтры пропускают всё", () => {
		expect(filterRows(ROWS, EMPTY_FILTERS)).toHaveLength(3);
	});

	it("статус сравнивается точно", () => {
		const filters: CampaignFilters = { ...EMPTY_FILTERS, status: "paused" };
		expect(filterRows(ROWS, filters)).toEqual([SECOND]);
	});

	it("поиск ищет подстроку модели без учёта регистра", () => {
		const filters: CampaignFilters = { ...EMPTY_FILTERS, query: "GATEWAY" };
		expect(filterRows(ROWS, filters)).toEqual([THIRD]);
	});

	it("поиск матчит модель, версию и короткий id", () => {
		expect(filterRows(ROWS, { ...EMPTY_FILTERS, query: "3.1.0" })).toEqual([THIRD]);
		expect(filterRows(ROWS, { ...EMPTY_FILTERS, query: "3f6d2d8e" })).toEqual([FIRST]);
	});

	it("условия комбинируются", () => {
		const filters: CampaignFilters = { status: "running", query: "sensor" };
		expect(filterRows(ROWS, filters)).toEqual([FIRST]);
	});
});

describe("hasActiveFilters", () => {
	it("пустые - нет, любые заполненные - да", () => {
		expect(hasActiveFilters(EMPTY_FILTERS)).toBe(false);
		expect(hasActiveFilters({ ...EMPTY_FILTERS, status: "draft" })).toBe(true);
		expect(hasActiveFilters({ ...EMPTY_FILTERS, query: "  " })).toBe(false);
	});
});

describe("paginateRows", () => {
	it("режет страницы и не выходит за конец", () => {
		expect(paginateRows(ROWS, 1, 2)).toHaveLength(2);
		expect(paginateRows(ROWS, 2, 2)).toEqual([THIRD]);
		expect(paginateRows(ROWS, 3, 2)).toHaveLength(0);
	});
});

describe("sortRows", () => {
	it("сортирует по доступнику и направлению", () => {
		const accessors = { created: (item: CampaignRow) => item.createdAt };
		const asc = sortRows(ROWS, [{ id: "created", desc: false }], accessors);
		expect(asc).toEqual(ROWS);

		const desc = sortRows(ROWS, [{ id: "created", desc: true }], accessors);
		expect(desc).toEqual([THIRD, SECOND, FIRST]);
	});

	it("без спецификации и без доступника порядок не меняется", () => {
		expect(sortRows(ROWS, [], {})).toEqual(ROWS);
		expect(sortRows(ROWS, [{ id: "unknown", desc: false }], {})).toEqual(ROWS);
	});
});
