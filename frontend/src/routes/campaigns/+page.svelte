<script lang="ts">
	import { goto } from "$app/navigation";
	import { page as appPage } from "$app/state";
	import { untrack } from "svelte";
	import RocketIcon from "phosphor-svelte/lib/RocketIcon";
	import {
		EMPTY_FILTERS,
		filterRows,
		paginateRows,
		sortRows,
		type CampaignFilters,
		type SortSpec,
	} from "$lib/domain/campaign-filters";
	import { toCampaignRow, type CampaignRow } from "$lib/domain/campaign-row";
	import { campaignActions } from "$lib/state/campaign-actions.svelte";
	import { campaignCollection } from "$lib/state/campaign-collection.svelte";
	import { campaignDetails } from "$lib/state/campaign-details.svelte";
	import { firmwareIndex } from "$lib/state/firmware-index.svelte";
	import { clock } from "$lib/state/clock.svelte";
	import { updatedAgoLabel } from "$lib/format/datetime";
	import { int } from "$lib/format/number";
	import CampaignsTable from "$lib/screens/campaigns/CampaignsTable.svelte";
	import FiltersBar from "$lib/screens/campaigns/FiltersBar.svelte";
	import ActionButton from "$lib/ui/ActionButton.svelte";
	import Button from "$lib/ui/Button.svelte";
	import EmptyState from "$lib/ui/EmptyState.svelte";
	import ErrorState from "$lib/ui/ErrorState.svelte";
	import InlineBanner from "$lib/ui/InlineBanner.svelte";
	import Pagination from "$lib/ui/Pagination.svelte";
	import RefreshControl from "$lib/ui/RefreshControl.svelte";
	import Skeleton from "$lib/ui/Skeleton.svelte";

	const DEFAULT_LIMIT = 25;

	function readUrl(): { filters: CampaignFilters; page: number; limit: number } {
		const params = appPage.url.searchParams;
		const page = Number.parseInt(params.get("page") ?? "1", 10);
		const limit = Number.parseInt(params.get("limit") ?? String(DEFAULT_LIMIT), 10);

		return {
			filters: {
				status: params.get("status") ?? "",
				query: params.get("q") ?? "",
			},
			page: Number.isFinite(page) && page > 0 ? page : 1,
			limit: [25, 50, 100].includes(limit) ? limit : DEFAULT_LIMIT,
		};
	}

	const initial = untrack(readUrl);
	let filters = $state(initial.filters);
	let page = $state(initial.page);
	let limit = $state(initial.limit);
	let sorting = $state<SortSpec[]>([]);

	$effect(() => {
		void firmwareIndex.ensureLoaded();
	});

	$effect(() => {
		const params = new URLSearchParams();
		if (filters.status) params.set("status", filters.status);
		if (filters.query) params.set("q", filters.query);
		if (page !== 1) params.set("page", String(page));
		if (limit !== DEFAULT_LIMIT) params.set("limit", String(limit));

		const search = params.toString();
		const target = `${appPage.url.pathname}${search ? `?${search}` : ""}`;
		if (target === appPage.url.pathname + appPage.url.search) return;
		void goto(target, { replaceState: true, keepFocus: true, noScroll: true });
	});

	const rows = $derived.by<CampaignRow[]>(() =>
		campaignCollection.rows.map((item) =>
			toCampaignRow(
				item,
				campaignDetails.get(item.id),
				firmwareIndex.versionLabel(item.firmware_version_id),
			),
		),
	);

	const filtered = $derived(filterRows(rows, filters));

	const sorted = $derived(
		sortRows(filtered, sorting, {
			created: (row) => row.createdAt,
			progress: (row) => row.stages.find((stage) => stage.status === "active")?.order_index ?? -1,
		}),
	);

	const paged = $derived(paginateRows(sorted, page, limit));
	const hasMore = $derived(paged.length === limit && page * limit < sorted.length);

	function setFilters(next: CampaignFilters): void {
		filters = next;
		page = 1;
	}

	async function refreshAll(): Promise<void> {
		await campaignCollection.refresh();
	}

	$effect(() =>
		campaignActions.onChanged(async () => {
			await campaignCollection.refresh();
		}),
	);

	const updatedAt = $derived(updatedAgoLabel(campaignCollection.lastUpdatedAt, clock.now));
</script>

<svelte:head>
	<title>Кампании - OTA Firmware Orchestrator</title>
</svelte:head>

<div class="flex flex-wrap items-end justify-between gap-3">
	<div>
		<h1 class="text-page font-semibold tracking-[-0.01em] text-fg-primary">Кампании</h1>
		<p class="mt-1 text-dense text-fg-muted">
			{#if campaignCollection.walking}
				загружено {int(campaignCollection.loadedCount)} кампаний
			{:else}
				показано {int(paged.length)} из {int(filtered.length)}
			{/if}
		</p>
	</div>
	<div class="flex items-center gap-2">
		<RefreshControl refreshing={campaignCollection.refreshing} refresh={refreshAll} {updatedAt} />
		<ActionButton href="/campaigns/new">Новая кампания</ActionButton>
	</div>
</div>

<div class="mt-4">
	<FiltersBar {filters} models={firmwareIndex.models()} onchange={setFilters} />
</div>

{#if campaignCollection.error && rows.length > 0}
	<div class="mt-4">
		<InlineBanner tone="danger">
			{campaignCollection.error.headline}
			{#snippet action()}
				<Button variant="secondary" size="sm" onclick={() => campaignCollection.refresh()}>
					Повторить
				</Button>
			{/snippet}
		</InlineBanner>
	</div>
{/if}

<div class="mt-4">
	{#if campaignCollection.pending}
		<div class="grid gap-1">
			{#each Array.from({ length: 8 }) as _, index (index)}
				<Skeleton shape="row" />
			{/each}
		</div>
	{:else if campaignCollection.error && rows.length === 0}
		<ErrorState error={campaignCollection.error} onretry={() => campaignCollection.refresh()} />
	{:else if rows.length === 0}
		<EmptyState
			icon={RocketIcon}
			title="Кампаний пока нет"
			text="Создайте первую: выберите прошивку и задайте стадии раскатки."
		>
			{#snippet action()}
				<ActionButton href="/campaigns/new">Новая кампания</ActionButton>
			{/snippet}
		</EmptyState>
	{:else if filtered.length === 0}
		<EmptyState icon={RocketIcon} title="Ничего не найдено">
			{#snippet action()}
				<Button variant="secondary" onclick={() => setFilters(EMPTY_FILTERS)}>
					Сбросить фильтры
				</Button>
			{/snippet}
		</EmptyState>
	{:else}
		<CampaignsTable
			rows={paged}
			{sorting}
			skeletonRows={limit}
			onSortingChange={(next) => (sorting = next)}
		/>

		<div class="mt-3">
			<Pagination
				{page}
				{limit}
				{hasMore}
				onPage={(next) => (page = next)}
				onLimit={(next) => {
					limit = next;
					page = 1;
				}}
			/>
		</div>
	{/if}
</div>
