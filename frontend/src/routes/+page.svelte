<script lang="ts">
	import RocketIcon from "phosphor-svelte/lib/RocketIcon";
	import { listRolloutCampaigns } from "$lib/api/endpoints";
	import {
		attentionReason,
		hasRecentChange,
		sortActiveRows,
		toCampaignRow,
		type AttentionReason,
		type CampaignRow,
	} from "$lib/domain/campaign-row";
	import { isLive } from "$lib/domain/transitions";
	import { campaignActions } from "$lib/state/campaign-actions.svelte";
	import { campaignDetails } from "$lib/state/campaign-details.svelte";
	import { firmwareIndex } from "$lib/state/firmware-index.svelte";
	import { createResource } from "$lib/state/resource.svelte";
	import { clock } from "$lib/state/clock.svelte";
	import { updatedAgoLabel } from "$lib/format/datetime";
	import { int } from "$lib/format/number";
	import ActiveRolloutRow from "$lib/screens/overview/ActiveRolloutRow.svelte";
	import AttentionPanel from "$lib/screens/overview/AttentionPanel.svelte";
	import DistributionPanel from "$lib/screens/overview/DistributionPanel.svelte";
	import SummaryRail from "$lib/screens/overview/SummaryRail.svelte";
	import ActionButton from "$lib/ui/ActionButton.svelte";
	import Button from "$lib/ui/Button.svelte";
	import EmptyState from "$lib/ui/EmptyState.svelte";
	import Panel from "$lib/ui/Panel.svelte";
	import RefreshControl from "$lib/ui/RefreshControl.svelte";
	import Skeleton from "$lib/ui/Skeleton.svelte";

	// Обновление раз в 30 секунд: чаще обзор дёргать смысла нет, а поллинг
	// заметно нагружает оркестратор на больших списках кампаний.
	const OVERVIEW_POLL_MS = 30_000;
	const DETAIL_CAP = 20;

	const list = createResource(
		"overview-campaigns",
		(signal) => listRolloutCampaigns({ page: 1, limit: 100 }, { signal }),
		{
			intervalMs: OVERVIEW_POLL_MS,
			// Связка «список -> агрегатор» живёт в callback, а не в $effect: refresh()
			// читает реактивное состояние ресурса, и эффект с таким вызовом
			// перезапускал бы сам себя бесконечно.
			onSettled: (settled) => {
				if (!settled.ok || !settled.data) return;
				const ids = settled.data.rollout_campaigns
					.filter((item) => item.status === "running" || item.status === "paused")
					.slice(0, DETAIL_CAP)
					.map((item) => item.id);
				campaignDetails.setPolled(ids);
				void campaignDetails.refresh();
			},
		},
	);

	const rows = $derived.by<CampaignRow[]>(() =>
		(list.data?.rollout_campaigns ?? []).map((item) =>
			toCampaignRow(
				item,
				campaignDetails.get(item.id),
				firmwareIndex.versionLabel(item.firmware_version_id),
			),
		),
	);

	const activeRows = $derived(sortActiveRows(rows.filter((row) => isLive(row.status))));
	// Рендерятся только первые DETAIL_CAP строк: ровно те, что живут в опросе
	// деталей, - подпись «показаны первые 20 из N» обязана быть правдой.
	const shownRows = $derived(activeRows.slice(0, DETAIL_CAP));
	const attention = $derived.by<{ row: CampaignRow; reason: AttentionReason }[]>(() => {
		const items: { row: CampaignRow; reason: AttentionReason }[] = [];
		const now = clock.now;
		for (const row of activeRows) {
			const reason = attentionReason(row);
			if (!reason) continue;
			if (!hasRecentChange(row, now)) continue;
			items.push({ row, reason });
		}
		return items;
	});

	async function refreshAll(): Promise<void> {
		await list.refresh();
		await campaignDetails.refresh();
	}

	$effect(() =>
		campaignActions.onChanged(async () => {
			await list.refresh();
			await campaignDetails.refresh();
		}),
	);

	const updatedAt = $derived(updatedAgoLabel(list.lastUpdatedAt, clock.now));
</script>

<svelte:head>
	<title>Обзор - OTA Firmware Orchestrator</title>
</svelte:head>

<div class="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between sm:gap-3">
	<h1 class="text-page font-semibold tracking-[-0.01em] text-fg-primary">Обзор</h1>
	<div class="flex items-center gap-2">
		<RefreshControl
			refreshing={list.refreshing || campaignDetails.refreshing}
			refresh={refreshAll}
			{updatedAt}
		/>
		<ActionButton href="/campaigns/new">Новая кампания</ActionButton>
	</div>
</div>

<!-- items-start: колонки не растягиваются по высоте друг друга.
     Левая колонка: активные раскатки как главная. Правая рельса:
     внимание + распределение + прошивки/модели. -->
<div class="mt-4 grid gap-4 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)] lg:items-start">
	<div class="grid min-w-0 content-start gap-4">
		<Panel title="Активные раскатки">
			{#if list.pending}
				<ul class="grid">
					{#each Array.from({ length: 3 }) as _, index (index)}
						<li
							class="flex h-[135px] flex-col justify-center gap-2 border-b border-border-subtle last:border-b-0 md:h-[83px]"
						>
							<Skeleton class="h-5 w-2/3" />
							<Skeleton class="h-4 w-1/3" />
						</li>
					{/each}
				</ul>
			{:else if activeRows.length === 0}
				<EmptyState
					icon={RocketIcon}
					title="Активных раскаток нет"
					text="Запустите кампанию из черновика или создайте новую."
				>
					{#snippet action()}
						<Button variant="secondary" href="/campaigns">К кампаниям</Button>
					{/snippet}
				</EmptyState>
			{:else}
				{#if activeRows.length > DETAIL_CAP}
					<p class="flex flex-wrap items-center gap-x-1 pb-2 text-dense text-fg-muted">
						показаны первые {int(DETAIL_CAP)} из {int(activeRows.length)}
						<a href="/campaigns" class="underline underline-offset-4 hover:text-accent-text">
							все кампании
						</a>
					</p>
				{/if}
				<!-- overflow-anchor:none - при смене статуса строки пересортировываются,
				     и браузерная привязка прокрутки уводила страницу вверх. -->
				<ul class="grid [overflow-anchor:none]">
					{#each shownRows as row (row.id)}
						<ActiveRolloutRow {row} />
					{/each}
				</ul>
			{/if}
		</Panel>
	</div>

	<div class="grid min-w-0 content-start gap-4">
		{#if list.pending}
			<Panel title="Требуют внимания">
				<div class="grid gap-2">
					<Skeleton class="h-9 w-full" />
					<Skeleton class="h-9 w-full" />
					<Skeleton class="h-9 w-2/3" />
				</div>
			</Panel>
		{:else}
			<AttentionPanel items={attention} />
		{/if}

		{#if list.pending}
			<Panel title="Распределение кампаний">
				<Skeleton class="h-24 w-full" />
			</Panel>
		{:else}
			<DistributionPanel {rows} />
		{/if}

		<SummaryRail
			{rows}
			firmware={firmwareIndex.versions}
			firmwareCount={firmwareIndex.versions.length}
		/>
	</div>
</div>
