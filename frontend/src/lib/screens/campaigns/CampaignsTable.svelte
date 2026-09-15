<script lang="ts" generics="T extends CampaignRow">
	import { goto } from "$app/navigation";
	import ArrowCounterClockwiseIcon from "phosphor-svelte/lib/ArrowCounterClockwiseIcon";
	import ArrowSquareOutIcon from "phosphor-svelte/lib/ArrowSquareOutIcon";
	import PauseIcon from "phosphor-svelte/lib/PauseIcon";
	import PlayIcon from "phosphor-svelte/lib/PlayIcon";
	import RocketLaunchIcon from "phosphor-svelte/lib/RocketLaunchIcon";
	import type { SortingState } from "@tanstack/svelte-table";
	import type { CampaignRow } from "$lib/domain/campaign-row";
	import { activeStageOf, progressOf } from "$lib/domain/campaign-row";
	import { allowedActions, type CampaignAction } from "$lib/domain/transitions";
	import { CAMPAIGN_STATUS } from "$lib/domain/status";
	import { ROLLBACK_CONFIRM } from "$lib/domain/transitions";
	import type { SortSpec } from "$lib/domain/campaign-filters";
	import { campaignActions } from "$lib/state/campaign-actions.svelte";
	import { campaignDetails } from "$lib/state/campaign-details.svelte";
	import ConfirmDialog from "$lib/ui/ConfirmDialog.svelte";
	import DataTable from "$lib/ui/DataTable.svelte";
	import type { Column } from "$lib/ui/table-types";
	import IconButton from "$lib/ui/IconButton.svelte";
	import MonoId from "$lib/ui/MonoId.svelte";
	import RelativeTime from "$lib/ui/RelativeTime.svelte";
	import Skeleton from "$lib/ui/Skeleton.svelte";
	import StatusChip from "$lib/ui/StatusChip.svelte";
	import Tooltip from "$lib/ui/Tooltip.svelte";
	import StagePipeline from "$lib/viz/StagePipeline.svelte";

	let {
		rows,
		sorting,
		onSortingChange,
		skeletonRows = 25,
	}: {
		rows: T[];
		skeletonRows?: number;
		sorting: SortSpec[];
		onSortingChange: (sorting: SortSpec[]) => void;
	} = $props();

	let container: HTMLDivElement | undefined = $state(undefined);
	let confirmRow = $state<CampaignRow | undefined>(undefined);

	// Ширины колонок зафиксированы (px) и растягиваются драгом разделителей:
	// раскладка не зависит от содержимого строк и активных фильтров.
	// Кампания шире остальных: в ней две строки (модель и идентификатор).
	// Действия узкие: максимум три иконки; Создана шире: префикс
	// «завершена» + относительное время не должны сжиматься.
	// Цвет иконки повторяет смысл действия: зелёный - запуск и возобновление,
	// жёлтый - пауза, красный - откат. Подпись всё равно обязательна: цвет
	// здесь только ускоряет узнавание, а не заменяет название.
	const columns: Column<T>[] = [
		{ id: "campaign", header: "Кампания", sortValue: undefined, width: 260 },
		{ id: "version", header: "Версия", width: 128 },
		{ id: "status", header: "Статус", width: 144 },
		{ id: "stages", header: "Стадии", width: 190 },
		{
			id: "progress",
			header: "Прогресс",
			align: "numeric",
			width: 96,
			sortValue: (row) => {
				const active = activeStageOf(row);
				return active ? active.order_index + 1 : -1;
			},
		},
		{
			id: "created",
			header: "Создана",
			align: "right",
			width: 200,
			sortValue: (row) => row.createdAt,
		},
		{ id: "actions", header: "Действия", align: "right", width: 112 },
	];

	// Стадии догружаются деталью кампании по видимости строки, а не все сразу.
	// Эффект пересоздаёт наблюдатель при каждой смене набора строк: до загрузки
	// списка в таблице нет ни одного [data-row-id].
	$effect(() => {
		const trackedIds = rows.map((row) => row.id).join(",");
		if (!container || trackedIds.length === 0) return;

		const observer = new IntersectionObserver(
			(entries) => {
				for (const entry of entries) {
					if (!entry.isIntersecting) continue;
					const id = (entry.target as HTMLElement).dataset.rowId;
					if (id) campaignDetails.ensure([id]);
					observer.unobserve(entry.target);
				}
			},
			{ rootMargin: "200px" },
		);

		for (const element of container.querySelectorAll<HTMLElement>("[data-row-id]")) {
			observer.observe(element);
		}
		return () => observer.disconnect();
	});

	function label(row: CampaignRow): string {
		return row.version ? `${row.model} ${row.version}` : row.model;
	}

	function open(row: CampaignRow): void {
		void goto(`/campaigns/${row.id}`);
	}

	function actionPending(row: CampaignRow, action: CampaignAction): boolean {
		return (
			campaignActions.isPending(row.id, action) ||
			(action === "rollback" && campaignActions.isRollbackInFlight(row.id))
		);
	}

	function toSortingState(): SortingState {
		return sorting.map((spec) => ({ id: spec.id, desc: spec.desc }));
	}

	function fromSortingState(state: SortingState): SortSpec[] {
		return state.map((spec) => ({ id: spec.id, desc: spec.desc }));
	}
</script>

{#snippet rowActions(row: T)}
	<!-- Быстрые действия иконками: кнопка на каждое доступное действие,
	     подсказка с названием появляется при наведении после задержки.
	     stopPropagation страховка: клик по кнопкам не активирует строку. -->
	<div
		class="flex items-center justify-end gap-0.5"
		role="none"
		onclick={(event) => event.stopPropagation()}
	>
		<Tooltip label="Открыть кампанию">
			<span class="inline-flex">
				<IconButton
					glyph={ArrowSquareOutIcon}
					ariaLabel="Открыть кампанию"
					size="sm"
					onclick={() => open(row)}
				/>
			</span>
		</Tooltip>
		{#if allowedActions(row.status).includes("pause")}
			<Tooltip label="Поставить на паузу">
				<span class="inline-flex">
					<IconButton
						glyph={PauseIcon}
						ariaLabel="Поставить на паузу"
						size="sm"
						variant="warning"
						loading={actionPending(row, "pause")}
						onclick={() =>
							campaignActions.pause(row.id, label(row), CAMPAIGN_STATUS[row.status].label)}
					/>
				</span>
			</Tooltip>
		{/if}
		{#if allowedActions(row.status).includes("resume")}
			<Tooltip label="Возобновить">
				<span class="inline-flex">
					<IconButton
						glyph={PlayIcon}
						ariaLabel="Возобновить"
						size="sm"
						variant="success"
						loading={actionPending(row, "resume")}
						onclick={() =>
							campaignActions.resume(row.id, label(row), CAMPAIGN_STATUS[row.status].label)}
					/>
				</span>
			</Tooltip>
		{/if}
		{#if allowedActions(row.status).includes("start")}
			<Tooltip label="Запустить">
				<span class="inline-flex">
					<IconButton
						glyph={RocketLaunchIcon}
						ariaLabel="Запустить"
						size="sm"
						variant="success"
						loading={actionPending(row, "start")}
						onclick={() =>
							campaignActions.start(row.id, label(row), CAMPAIGN_STATUS[row.status].label)}
					/>
				</span>
			</Tooltip>
		{/if}
		{#if allowedActions(row.status).includes("rollback")}
			<Tooltip label="Откатить">
				<span class="inline-flex">
					<IconButton
						glyph={ArrowCounterClockwiseIcon}
						ariaLabel="Откатить"
						size="sm"
						variant="danger"
						loading={actionPending(row, "rollback")}
						onclick={() => (confirmRow = row)}
					/>
				</span>
			</Tooltip>
		{/if}
	</div>
{/snippet}

<div bind:this={container}>
	<!-- Узкий экран: карточки вместо таблицы (§6). -->
	<ol class="grid gap-2 md:hidden">
		{#each rows as row (row.id)}
			{@const progress = progressOf(row)}
			<li
				data-row-id={row.id}
				class="grid gap-1.5 rounded-panel border border-border-subtle bg-bg-surface p-3"
			>
				<div class="flex flex-wrap items-center gap-2">
					<span class="text-table font-medium text-fg-primary">{row.model}</span>
					{#if row.version}
						<span class="font-mono text-table text-fg-secondary">{row.version}</span>
					{/if}
					<StatusChip meta={CAMPAIGN_STATUS[row.status]} raw={row.status} size="sm" />
				</div>
				<div class="flex flex-wrap items-center justify-between gap-2 text-dense text-fg-secondary">
					<MonoId id={row.id} copyKey={`campaign-card-${row.id}`} />
					<span class="tabular-nums" title={progress.title}>{progress.label}</span>
				</div>
				{#if campaignDetails.get(row.id)}
					<StagePipeline stages={row.stages} mode="compact" />
				{:else}
					<Skeleton shape="chip" class="h-1.5 w-full" />
				{/if}
				<div class="flex items-center justify-between gap-2">
					{#if row.completedAt}
						<span class="text-dense text-fg-secondary"
							>завершена <RelativeTime iso={row.completedAt} /></span
						>
					{:else}
						<span class="text-dense text-fg-secondary"
							>создана <RelativeTime iso={row.createdAt} /></span
						>
					{/if}
					{@render rowActions(row)}
				</div>
			</li>
		{:else}
			<li class="text-fg-muted">Кампаний нет</li>
		{/each}
	</ol>

	<div class="hidden min-w-0 md:block">
		<DataTable
			tableId="campaigns"
			{columns}
			{rows}
			{skeletonRows}
			rowKey={(row) => row.id}
			sorting={toSortingState()}
			onSortingChange={(state) => onSortingChange(fromSortingState(state))}
			onRowClick={open}
			caption="Кампании раскатки: модель, версия прошивки, статус, стадии, прогресс и даты"
		>
			{#snippet cell(id: string, row: T)}
				{#if id === "campaign"}
					<div class="grid gap-0.5">
						<span class="text-ui font-medium text-fg-primary">{row.model}</span>
						<MonoId id={row.id} copyKey={`row-${row.id}`} />
					</div>
				{:else if id === "version"}
					{#if row.version}
						<span class="font-mono text-table text-fg-secondary">{row.version}</span>
					{:else}
						<span class="text-table text-fg-muted">нет данных</span>
					{/if}
				{:else if id === "status"}
					<StatusChip meta={CAMPAIGN_STATUS[row.status]} raw={row.status} size="sm" />
				{:else if id === "stages"}
					{#if campaignDetails.get(row.id)}
						<StagePipeline stages={row.stages} mode="compact" class="w-40" />
					{:else}
						<Skeleton shape="chip" class="h-5 w-24" />
					{/if}
				{:else if id === "progress"}
					{@const progress = progressOf(row)}
					<span class="tabular-nums" title={progress.title}>{progress.label}</span>
				{:else if id === "created"}
					{#if row.completedAt}
						<span class="text-table text-fg-secondary">
							завершена <RelativeTime iso={row.completedAt} />
						</span>
					{:else}
						<RelativeTime iso={row.createdAt} />
					{/if}
				{:else}
					{@render rowActions(row)}
				{/if}
			{/snippet}
		</DataTable>
	</div>
</div>

{#if confirmRow}
	<ConfirmDialog
		open={confirmRow !== undefined}
		onOpenChange={(open) => {
			if (!open) confirmRow = undefined;
		}}
		title={ROLLBACK_CONFIRM.title}
		body={ROLLBACK_CONFIRM.body}
		confirmLabel={ROLLBACK_CONFIRM.confirm}
		tone="danger"
		busy={campaignActions.isRollbackInFlight(confirmRow.id)}
		onconfirm={() => {
			const row = confirmRow;
			if (!row) return;
			confirmRow = undefined;
			void campaignActions.rollback(row.id, label(row), CAMPAIGN_STATUS[row.status].label);
		}}
	/>
{/if}
