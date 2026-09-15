<script lang="ts">
	import { untrack } from "svelte";
	import XIcon from "phosphor-svelte/lib/XIcon";
	import { CAMPAIGN_STATUS } from "$lib/domain/status";
	import { hasActiveFilters, type CampaignFilters } from "$lib/domain/campaign-filters";
	import Button from "$lib/ui/Button.svelte";
	import Field from "$lib/ui/Field.svelte";
	import Panel from "$lib/ui/Panel.svelte";
	import SearchInput from "$lib/ui/SearchInput.svelte";
	import Select from "$lib/ui/Select.svelte";

	/**
	 * Панель фильтров списка кампаний: статус, единое поле поиска (модель,
	 * версия или короткий id) с popup-подсказками моделей и кнопка «Сбросить».
	 *
	 * Отдельного ввода модели нет: его полностью покрывает поиск, а выбор
	 * точного названия модели даёт тот же popup, что на других экранах.
	 *
	 * Синхронизация двусторонняя: локальные копии полей живут здесь ради
	 * дебаунса, но внешние изменения фильтров (например, «Сбросить фильтры»
	 * в пустом состоянии экрана) возвращаются в поля - иначе устаревшее
	 * локальное значение тут же перезаписывало бы сброс.
	 */
	let {
		filters,
		models,
		onchange,
	}: {
		filters: CampaignFilters;
		models: string[];
		onchange: (filters: CampaignFilters) => void;
	} = $props();

	const DEBOUNCE_MS = 150;

	let statusValue = $state(untrack(() => filters.status));
	let queryInput = $state(untrack(() => filters.query));

	// Последнее значение, которое панель отправила наружу или получила извне:
	// отличие props от него означает внешнее изменение фильтров.
	let applied = $state(untrack(() => ({ status: filters.status, query: filters.query })));

	const STATUS_ITEMS = [
		{ value: "", label: "все" },
		...Object.entries(CAMPAIGN_STATUS).map(([value, meta]) => ({ value, label: meta.label })),
	];

	$effect(() => {
		const next = { status: filters.status, query: filters.query };
		if (next.status === applied.status && next.query === applied.query) return;
		applied = next;
		statusValue = next.status;
		queryInput = next.query;
	});

	$effect(() => {
		const value = statusValue;
		if (value === applied.status) return;
		applied = { ...applied, status: value };
		onchange({ ...filters, status: value });
	});

	$effect(() => {
		const value = queryInput;
		if (value === applied.query) return;
		const timer = setTimeout(() => {
			applied = { ...applied, query: value };
			onchange({ ...filters, query: value });
		}, DEBOUNCE_MS);
		return () => clearTimeout(timer);
	});

	function reset(): void {
		statusValue = "";
		queryInput = "";
		applied = { status: "", query: "" };
		onchange({ status: "", query: "" });
	}
</script>

<Panel dense>
	<div class="flex flex-wrap items-end gap-3">
		<Field label="Статус" class="w-44">
			<Select
				id="campaign-status"
				bind:value={statusValue}
				items={STATUS_ITEMS}
				ariaLabel="Статус кампании"
			/>
		</Field>

		<Field label="Поиск" for="campaign-search" class="min-w-52 flex-1">
			<SearchInput
				id="campaign-search"
				bind:value={queryInput}
				suggestions={models}
				placeholder="модель, версия или короткий id"
			/>
		</Field>

		{#if hasActiveFilters(filters)}
			<Button variant="secondary" icon={XIcon} onclick={reset}>Сбросить</Button>
		{/if}
	</div>
</Panel>
