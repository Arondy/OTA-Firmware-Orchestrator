<script lang="ts" generics="T extends { id: string }">
	import DataTable from "./DataTable.svelte";
	import type { Column } from "./table-types";

	/**
	 * Обёртка только для юнит-тестов: сниппет ячейки нельзя задать из TS-кода,
	 * его объявляет этот компонент. В прод-код не импортируется.
	 */
	let {
		columns,
		rows,
		onRowClick,
	}: {
		columns: Column<T>[];
		rows: T[];
		onRowClick?: (row: T) => void;
	} = $props();
</script>

<DataTable {columns} {rows} {onRowClick} rowKey={(row) => row.id} caption="Тестовая таблица">
	{#snippet cell(id, row)}
		<button type="button" data-cell-id={id}>{row.id}</button>
	{/snippet}
</DataTable>
