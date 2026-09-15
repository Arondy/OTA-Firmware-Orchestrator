<script lang="ts">
	import BinaryIcon from "phosphor-svelte/lib/BinaryIcon";
	import XIcon from "phosphor-svelte/lib/XIcon";
	import { listFirmwareVersions } from "$lib/api/endpoints";
	import type { FirmwareVersion } from "$lib/api/types";
	import { tryCompareSemver } from "$lib/logic/semver";
	import { shortId } from "$lib/format/id";
	import {
		createPagedResource,
		type Filters,
		type PagedResource,
	} from "$lib/state/paged-resource.svelte";
	import { firmwareIndex } from "$lib/state/firmware-index.svelte";
	import ActionButton from "$lib/ui/ActionButton.svelte";
	import Button from "$lib/ui/Button.svelte";
	import EmptyState from "$lib/ui/EmptyState.svelte";
	import Field from "$lib/ui/Field.svelte";
	import Pagination from "$lib/ui/Pagination.svelte";
	import SearchInput from "$lib/ui/SearchInput.svelte";
	import SearchSelect from "$lib/ui/SearchSelect.svelte";
	import FirmwareRegisterModal from "$lib/screens/firmware/FirmwareRegisterModal.svelte";
	import FirmwareTable from "$lib/screens/firmware/FirmwareTable.svelte";

	/**
	 * Реестр прошивок: серверный фильтр модели и пагинация без общего числа,
	 * поиск и сортировка - по загруженной странице и честно подписаны.
	 * Состояние, которым стоит делиться, живёт в адресной строке (§7.5):
	 * `?model=&page=&limit=` читает и пишет сам постаничный ресурс.
	 */
	interface FirmwareFilters extends Filters {
		model: string;
	}

	const resource: PagedResource<FirmwareVersion, FirmwareFilters> = createPagedResource({
		key: "firmware-registry",
		fetch: ({ page, limit, filters, signal }) =>
			listFirmwareVersions(
				{
					page,
					limit,
					// «Все модели» не отправляет параметр вовсе: пустая строка
					// в query превратилась бы в `device_model=` (§5.1).
					device_model: filters.model || undefined,
				},
				{ signal },
			).then((response) => response.firmware_versions),
		initialFilters: { model: "" },
		limit: 25,
	});

	let query = $state("");
	let modelFilter = $state(resource.filters.model ?? "");
	let registerOpen = $state(false);

	// Выбор модели и сброс фильтров сходятся в одну точку: URL пишет ресурс.
	$effect(() => {
		const current = resource.filters.model ?? "";
		if (modelFilter === current) return;
		void resource.setFilters({ model: modelFilter });
	});
	$effect(() => {
		const current = resource.filters.model ?? "";
		if (current !== modelFilter) modelFilter = current;
	});

	$effect(() => {
		void firmwareIndex.ensureLoaded();
	});

	// Единый элемент выбора модели: тот же SearchSelect, что на «Устройствах».
	// Варианта «Все модели» в списке нет - пустое значение и есть «все»,
	// плейсхолдер показывает его, а сбрасывает кнопка «Сбросить».
	const modelOptions = $derived.by(() => {
		const models = new Set(firmwareIndex.models());
		for (const row of resource.rows) models.add(row.device_model);
		if (modelFilter) models.add(modelFilter);
		return [...models]
			.sort((a, b) => a.localeCompare(b, "ru"))
			.map((model) => ({ value: model, label: model }));
	});

	const filtersActive = $derived(modelFilter !== "" || query.trim() !== "");

	const searched = $derived.by(() => {
		const needle = query.trim().toLowerCase();
		if (!needle) return resource.rows;
		return resource.rows.filter(
			(row) =>
				row.fw_version.toLowerCase().includes(needle) ||
				shortId(row.id).toLowerCase().includes(needle) ||
				row.fw_checksum.toLowerCase().includes(needle),
		);
	});

	/** «Последняя» - наибольший semver внутри модели среди загруженных строк. */
	const latestIds = $derived.by(() => {
		const best = new Map<string, FirmwareVersion>();
		for (const row of resource.rows) {
			const current = best.get(row.device_model);
			if (!current) {
				best.set(row.device_model, row);
				continue;
			}
			const compared = tryCompareSemver(row.fw_version, current.fw_version);
			if (compared !== undefined && compared > 0) best.set(row.device_model, row);
		}
		return new Set([...best.values()].map((row) => row.id));
	});

	const registryEmpty = $derived(
		!resource.pending && resource.rows.length === 0 && !resource.filters.model && !resource.error,
	);
</script>

<svelte:head>
	<title>Прошивки - OTA Firmware Orchestrator</title>
</svelte:head>

<div class="grid gap-4">
	<div class="flex flex-wrap items-end justify-between gap-3">
		<h1 class="text-page font-semibold tracking-[-0.01em] text-fg-primary">Прошивки</h1>
		<ActionButton onclick={() => (registerOpen = true)}>Зарегистрировать прошивку</ActionButton>
	</div>

	<div
		class="flex flex-wrap items-end gap-3 rounded-panel border border-border-subtle bg-bg-surface p-3"
	>
		<Field label="Модель" class="w-full sm:w-64">
			<SearchSelect
				id="firmware-model-filter"
				bind:value={modelFilter}
				items={modelOptions}
				ariaLabel="Модель устройства"
				placeholder="Все модели"
			/>
		</Field>
		<Field label="Поиск" for="firmware-search" class="min-w-52 flex-1">
			<SearchInput id="firmware-search" bind:value={query} placeholder="версия или короткий id" />
		</Field>
		{#if filtersActive}
			<Button
				variant="secondary"
				icon={XIcon}
				onclick={() => {
					query = "";
					modelFilter = "";
					void resource.resetFilters();
				}}
			>
				Сбросить
			</Button>
		{/if}
	</div>

	{#if registryEmpty}
		<EmptyState
			icon={BinaryIcon}
			title="Прошивок пока нет"
			text="Зарегистрируйте версию: модель, semver, sha256 бинарника и ссылка на файл."
		>
			{#snippet action()}
				<ActionButton onclick={() => (registerOpen = true)}>Зарегистрировать прошивку</ActionButton>
			{/snippet}
		</EmptyState>
	{:else}
		<FirmwareTable
			rows={searched}
			{latestIds}
			loading={resource.pending}
			skeletonRows={resource.limit}
		/>
		{#if !resource.pending && searched.length === 0}
			<EmptyState icon={BinaryIcon} title="Ничего не найдено">
				{#snippet action()}
					<Button
						variant="secondary"
						onclick={() => {
							query = "";
							void resource.resetFilters();
						}}
					>
						Сбросить фильтры
					</Button>
				{/snippet}
			</EmptyState>
		{/if}
	{/if}

	{#if !registryEmpty}
		<Pagination
			page={resource.page}
			limit={resource.limit}
			hasMore={resource.hasMore}
			onPage={(page) => void resource.setPage(page)}
			onLimit={(limit) => void resource.setLimit(limit)}
		/>
	{/if}
</div>

<FirmwareRegisterModal
	open={registerOpen}
	onOpenChange={(open) => (registerOpen = open)}
	onCreated={() => void resource.refresh()}
/>
