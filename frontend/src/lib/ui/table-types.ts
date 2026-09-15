export interface Column<TRow> {
	id: string;
	header: string;
	align?: "left" | "right" | "numeric";
	/** Значение для клиентской сортировки; без него колонка не сортируется. */
	sortValue?: (row: TRow) => string | number;
	/**
	 * Начальная ширина колонки в px для таблиц с `tableId`: раскладка
	 * `table-fixed` не зависит от подмены шрифта и не пляшет при фильтрации,
	 * а пользователь может изменить ширину драгом разделителя.
	 */
	width?: number;
}
