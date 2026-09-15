<script lang="ts" generics="T extends object">
	import type { Snippet } from "svelte";
	import {
		createTable,
		stockFeatures,
		type ColumnDef,
		type SortingState,
		type Updater,
	} from "@tanstack/svelte-table";

	type Features = typeof stockFeatures;
	import ArrowsDownUpIcon from "phosphor-svelte/lib/ArrowsDownUpIcon";
	import ArrowUpIcon from "phosphor-svelte/lib/ArrowUpIcon";
	import ArrowDownIcon from "phosphor-svelte/lib/ArrowDownIcon";
	import type { ApiError } from "$lib/api/errors";
	import type { Column } from "./table-types";
	import ErrorState from "./ErrorState.svelte";
	import Icon from "./Icon.svelte";
	import Skeleton from "./Skeleton.svelte";
	import { cn } from "./cn";

	let {
		columns,
		rows,
		rowKey,
		cell,
		caption,
		sorting,
		onSortingChange,
		loading = false,
		error,
		emptySnippet,
		onRowClick,
		onRetry,
		dense = false,
		tableId,
		skeletonRows = 6,
		class: className,
	}: {
		columns: Column<T>[];
		rows: T[];
		rowKey: (row: T) => string;
		/** Сниппет ячейки: получает идентификатор колонки и строку. */
		cell: Snippet<[string, T]>;
		caption: string;
		/** Управляемая сортировка: экран сортирует до пагинации. Без неё - внутренняя. */
		sorting?: SortingState;
		onSortingChange?: (sorting: SortingState) => void;
		loading?: boolean;
		error?: ApiError;
		emptySnippet?: Snippet;
		onRowClick?: (row: T) => void;
		onRetry?: () => void;
		dense?: boolean;
		/**
		 * Ключ таблицы для фиксированных ширин колонок: с ним раскладка
		 * `table-fixed`, ширины из `Column.width` не пляшут при фильтрации
		 * и загрузке шрифта, а пользователь может тянуть разделители колонок.
		 * Изменённые ширины сохраняются в localStorage.
		 */
		tableId?: string;
		/** Скелет повторяет размер страницы: замена строк не двигает макет. */
		skeletonRows?: number;
		class?: string;
	} = $props();

	let internalSorting = $state<SortingState>([]);
	const currentSorting = $derived(sorting ?? internalSorting);

	/* --- Ширины колонок (tableId) --- */

	const MIN_COL_WIDTH = 64;
	const DEFAULT_COL_WIDTH = 160;
	const STORAGE_PREFIX = "ota-col-widths:";

	const resizable = $derived(tableId !== undefined);
	/**
	 * Ширины задают все колонки, включая последнюю: на узком экране сумма
	 * ширин растягивает таблицу и обёртка даёт горизонтальную прокрутку,
	 * а не сплющивает последнюю колонку до нуля. На широком экране остаток
	 * распределяется между колонками пропорционально.
	 */
	const sizedColumns = $derived(resizable ? columns : []);

	function loadWidths(): Record<string, number> {
		const declared: Record<string, number> = {};
		for (const column of columns) {
			declared[column.id] = column.width ?? DEFAULT_COL_WIDTH;
		}
		if (!tableId) return declared;
		try {
			const raw = localStorage.getItem(STORAGE_PREFIX + tableId);
			if (!raw) return declared;
			const parsed: unknown = JSON.parse(raw);
			if (typeof parsed !== "object" || parsed === null) return declared;
			for (const [key, value] of Object.entries(parsed as Record<string, unknown>)) {
				if (typeof value === "number" && Number.isFinite(value) && key in declared) {
					declared[key] = Math.max(MIN_COL_WIDTH, Math.round(value));
				}
			}
		} catch {
			// Повреждённое хранилище не должно ронять таблицу.
		}
		return declared;
	}

	let widths = $state<Record<string, number>>(loadWidths());

	function persistWidths(): void {
		if (!tableId) return;
		try {
			localStorage.setItem(STORAGE_PREFIX + tableId, JSON.stringify(widths));
		} catch {
			// Хранилище недоступно (приватный режим): ширины живут до перезагрузки.
		}
	}

	function setWidth(id: string, next: number): void {
		widths = { ...widths, [id]: Math.max(MIN_COL_WIDTH, Math.round(next)) };
	}

	let drag: { id: string; startX: number; startWidth: number } | undefined;
	let draggingId = $state<string | undefined>(undefined);

	function onGripPointerDown(event: PointerEvent, id: string): void {
		event.preventDefault();
		(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId);
		drag = { id, startX: event.clientX, startWidth: widths[id] ?? DEFAULT_COL_WIDTH };
		draggingId = id;
	}

	function onGripPointerMove(event: PointerEvent): void {
		if (!drag) return;
		setWidth(drag.id, drag.startWidth + (event.clientX - drag.startX));
	}

	function onGripPointerUp(): void {
		if (!drag) return;
		drag = undefined;
		draggingId = undefined;
		persistWidths();
	}

	function onGripKeydown(event: KeyboardEvent, id: string): void {
		const step = event.shiftKey ? 32 : 8;
		if (event.key === "ArrowLeft") {
			event.preventDefault();
			setWidth(id, (widths[id] ?? DEFAULT_COL_WIDTH) - step);
			persistWidths();
		} else if (event.key === "ArrowRight") {
			event.preventDefault();
			setWidth(id, (widths[id] ?? DEFAULT_COL_WIDTH) + step);
			persistWidths();
		}
	}

	/* --- Сортировка --- */

	// TanStack держит состояние и модели строк; ячейки рисуем своими сниппетами.
	const defs = $derived(
		columns.map((column): ColumnDef<Features, T, unknown> => ({
			id: column.id,
			header: column.header,
			accessorFn: column.sortValue ?? (() => undefined),
			enableSorting: column.sortValue !== undefined,
		})),
	);

	const table = createTable<Features, T>({
		features: stockFeatures,
		get data() {
			return rows;
		},
		get columns() {
			return defs;
		},
		get state() {
			return { sorting: currentSorting };
		},
		onSortingChange: (updater: Updater<SortingState>) => {
			const next = typeof updater === "function" ? updater(currentSorting) : updater;
			// Одиночная сортировка: вторая колонка заменяет первую, а не добавляется.
			const sliced = next.slice(-1);
			if (onSortingChange) onSortingChange(sliced);
			else internalSorting = sliced;
		},
	});

	const sortedRows = $derived(table.getRowModel().rows.map((row) => row.original));

	function toggleSort(column: Column<T>): void {
		const current = currentSorting[0];
		let next: SortingState;
		if (current?.id !== column.id) next = [{ id: column.id, desc: false }];
		else if (!current.desc) next = [{ id: column.id, desc: true }];
		else next = [];

		if (onSortingChange) onSortingChange(next);
		else internalSorting = next;
	}

	function sortGlyph(column: Column<T>) {
		const current = currentSorting[0];
		if (current?.id !== column.id) return ArrowsDownUpIcon;
		return current.desc ? ArrowDownIcon : ArrowUpIcon;
	}

	const ALIGN = {
		left: "text-left",
		right: "text-right",
		numeric: "text-right tabular-nums",
	} as const;

	/**
	 * Клик по интерактивному элементу внутри строки (копирование id, кнопки
	 * действий, ссылки) не должен активировать строку: иначе копирование
	 * идентификатора уводило бы оператора в карточку кампании.
	 */
	const INTERACTIVE_SELECTOR =
		'button, a, input, select, textarea, label, [role="button"], [role="option"], [role="menuitem"]';

	function isInteractive(target: EventTarget | null): boolean {
		return target instanceof Element && target.closest(INTERACTIVE_SELECTOR) !== null;
	}
</script>

{#if error}
	<ErrorState {error} onretry={onRetry} />
{:else}
	<!-- contain:paint обрезает переполнение широкой table-fixed таблицы:
	    	без него Chromium протаскивает часть прокручиваемой области на страницу
	    	и на 768-1280px появляется горизонтальная прокрутка всего документа. -->
	<div class={cn("overflow-x-auto [contain:paint]", className)}>
		<table class={cn("w-full border-collapse text-table", resizable && "table-fixed")}>
			<caption class="sr-only">{caption}</caption>
			{#if resizable}
				<colgroup>
					{#each sizedColumns as column (column.id)}
						<col style="width: {widths[column.id] ?? DEFAULT_COL_WIDTH}px" />
					{/each}
				</colgroup>
			{/if}
			<thead class="sticky top-0 z-10 bg-bg-inset">
				<tr class="border-b border-border-subtle">
					{#each columns as column, index (column.id)}
						<th
							scope="col"
							class={cn(
								"px-3 py-2 text-micro font-medium uppercase text-fg-muted",
								resizable && "relative",
								ALIGN[column.align ?? "left"],
							)}
						>
							{#if column.sortValue}
								<button
									type="button"
									class="inline-flex items-center gap-1 uppercase hover:text-fg-secondary"
									onclick={() => toggleSort(column)}
								>
									{column.header}
									<Icon glyph={sortGlyph(column)} size={16} />
								</button>
							{:else}
								{column.header}
							{/if}
							{#if resizable && index < columns.length - 1}
								<!-- Разделитель колонок: драг указателем, стрелки с клавиатуры.
								     Кнопка, а не span с ролью: интерактивный элемент без
								     предупреждений доступности. -->
								<button
									type="button"
									aria-label={`Ширина колонки «${column.header}»: перетащите или измените стрелками`}
									class={cn(
										"absolute inset-y-0 -right-1 z-20 w-2 cursor-col-resize touch-none select-none",
										"after:absolute after:inset-y-1 after:left-1/2 after:w-px after:-translate-x-1/2 after:transition-colors",
										// Линия видна всегда: без неё непонятно, где граница,
										// которую можно тянуть; при наведении и драге - ярче.
										draggingId === column.id
											? "after:bg-accent"
											: "after:bg-border-strong/60 hover:after:bg-border-strong focus-visible:after:bg-border-strong",
										"focus-visible:outline-2 focus-visible:outline-[var(--ring)]",
									)}
									onpointerdown={(event) => onGripPointerDown(event, column.id)}
									onpointermove={onGripPointerMove}
									onpointerup={onGripPointerUp}
									onpointercancel={onGripPointerUp}
									onkeydown={(event) => onGripKeydown(event, column.id)}
								></button>
							{/if}
						</th>
					{/each}
				</tr>
			</thead>

			<tbody>
				{#if loading}
					{#each Array.from({ length: skeletonRows }) as _, index (index)}
						<tr class="border-b border-border-subtle">
							{#each columns as column (column.id)}
								<td class={cn("px-3", dense ? "h-9" : "h-10", ALIGN[column.align ?? "left"])}>
									<Skeleton shape="text" rows={1} class="w-3/4" />
								</td>
							{/each}
						</tr>
					{/each}
				{:else if sortedRows.length === 0}
					<tr>
						<td colspan={columns.length}>
							{#if emptySnippet}{@render emptySnippet()}{/if}
						</td>
					</tr>
				{:else}
					{#each sortedRows as row (rowKey(row))}
						<tr
							data-row-id={rowKey(row)}
							class={cn(
								"row-vis border-b border-border-subtle transition-colors",
								onRowClick && "cursor-pointer hover:bg-bg-raised",
							)}
							tabindex={onRowClick ? 0 : undefined}
							role={onRowClick ? "link" : undefined}
							onclick={(event) => {
								if (!onRowClick || isInteractive(event.target)) return;
								onRowClick(row);
							}}
							onkeydown={(event) => {
								if (!onRowClick) return;
								// Enter/Space активируют строку только когда фокус на самой
								// строке. Событие от вложенной кнопки (меню действий, копирование)
								// всплывает до строки: без проверки нажатие Enter в меню
								// уводило бы с экрана вместо открытия меню.
								if (event.target !== event.currentTarget) return;
								if (event.key === "Enter" || event.key === " ") {
									event.preventDefault();
									onRowClick(row);
								}
							}}
						>
							{#each columns as column (column.id)}
								<td
									class={cn(
										"px-3 text-fg-primary",
										dense ? "h-9" : "h-10",
										ALIGN[column.align ?? "left"],
									)}
								>
									{@render cell(column.id, row)}
								</td>
							{/each}
						</tr>
					{/each}
				{/if}
			</tbody>
		</table>
	</div>
{/if}
