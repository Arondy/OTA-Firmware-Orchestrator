import { describe, expect, it, vi } from "vitest";
import { mount, unmount } from "svelte";
import DataTableHarness from "./DataTableHarness.svelte";
import type { Column } from "./table-types";

/**
 * Enter и Space на вложенной кнопке строки (меню «Действия с кампанией») не
 * должны активировать саму строку: обработчик обязан различать фокус на строке
 * и на её вложенном элементе, иначе оператор уходит с экрана вместо открытия меню.
 */

interface Row {
	id: string;
}

const columns: Column<Row>[] = [{ id: "name", header: "Имя" }];

function press(target: Element, key: string): void {
	target.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true }));
}

function mountRows(onRowClick: (row: Row) => void): () => void {
	const component = mount(DataTableHarness, {
		target: document.body,
		props: { columns, rows: [{ id: "row-1" }], onRowClick },
	});
	return () => unmount(component);
}

describe("DataTable: клавиатура строки", () => {
	it("Enter на самой строке открывает кампанию", () => {
		const onRowClick = vi.fn();
		const dispose = mountRows(onRowClick);

		const row = document.querySelector<HTMLElement>("[data-row-id='row-1']");
		expect(row).not.toBeNull();
		row?.focus();
		press(row as HTMLElement, "Enter");

		expect(onRowClick).toHaveBeenCalledTimes(1);
		expect(onRowClick).toHaveBeenCalledWith({ id: "row-1" });
		dispose();
	});

	it("Enter на вложенной кнопке не уводит с экрана", () => {
		const onRowClick = vi.fn();
		const dispose = mountRows(onRowClick);

		const button = document.querySelector<HTMLButtonElement>("[data-row-id='row-1'] button");
		expect(button).not.toBeNull();
		press(button as HTMLButtonElement, "Enter");

		expect(onRowClick).not.toHaveBeenCalled();
		dispose();
	});

	it("Space на вложенной кнопке тоже не активирует строку", () => {
		const onRowClick = vi.fn();
		const dispose = mountRows(onRowClick);

		const button = document.querySelector<HTMLButtonElement>("[data-row-id='row-1'] button");
		press(button as HTMLButtonElement, " ");

		expect(onRowClick).not.toHaveBeenCalled();
		dispose();
	});
});

/**
 * Клик по кнопке копирования id внутри строки не должен активировать саму
 * строку: иначе оператор уходит в карточку кампании вместо копирования.
 */
describe("DataTable: клик по строке", () => {
	it("клик по ячейке активирует строку", () => {
		const onRowClick = vi.fn();
		const dispose = mountRows(onRowClick);

		const row = document.querySelector<HTMLElement>("[data-row-id='row-1']");
		row?.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true }));

		expect(onRowClick).toHaveBeenCalledTimes(1);
		dispose();
	});

	it("клик по вложенной кнопке не активирует строку", () => {
		const onRowClick = vi.fn();
		const dispose = mountRows(onRowClick);

		const button = document.querySelector<HTMLButtonElement>("[data-row-id='row-1'] button");
		expect(button).not.toBeNull();
		button?.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true }));

		expect(onRowClick).not.toHaveBeenCalled();
		dispose();
	});
});
