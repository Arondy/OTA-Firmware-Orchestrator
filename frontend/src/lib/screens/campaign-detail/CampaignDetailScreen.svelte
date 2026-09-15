<script lang="ts">
	import FileDashedIcon from "phosphor-svelte/lib/FileDashedIcon";
	import FileXIcon from "phosphor-svelte/lib/FileXIcon";
	import { untrack } from "svelte";
	import { getRolloutCampaign } from "$lib/api/endpoints";
	import type { Campaign } from "$lib/api/types";
	import { campaignActions } from "$lib/state/campaign-actions.svelte";
	import { createResource, type Resource } from "$lib/state/resource.svelte";
	import { firmwareIndex } from "$lib/state/firmware-index.svelte";
	import { campaignCollection } from "$lib/state/campaign-collection.svelte";
	import { stageStats } from "$lib/state/stage-stats.svelte";
	import { CAMPAIGN_STATUS } from "$lib/domain/status";
	import {
		allowedActions,
		actionMeta,
		isLive,
		ROLLBACK_CONFIRM,
		type CampaignAction,
	} from "$lib/domain/transitions";
	import { actionErrorHeadline } from "$lib/domain/action-errors";
	import Button from "$lib/ui/Button.svelte";
	import ConfirmDialog from "$lib/ui/ConfirmDialog.svelte";
	import EmptyState from "$lib/ui/EmptyState.svelte";
	import ErrorState from "$lib/ui/ErrorState.svelte";
	import InlineBanner from "$lib/ui/InlineBanner.svelte";
	import JsonView from "$lib/ui/JsonView.svelte";
	import Panel from "$lib/ui/Panel.svelte";
	import Skeleton from "$lib/ui/Skeleton.svelte";
	import StagePipeline from "$lib/viz/StagePipeline.svelte";
	import DetailHeader from "./DetailHeader.svelte";
	import IdentityStrip from "./IdentityStrip.svelte";
	import StageTable from "./StageTable.svelte";
	import MetricsPanel from "./MetricsPanel.svelte";
	import HistoryTimeline from "./HistoryTimeline.svelte";

	/**
	 * Экран кампании: единственный ресурс с поллингом 3 с, пока статус живой
	 * и вкладка видна; терминальные статусы не опрашиваются вовсе. Откат
	 * переживает три фазы: подтверждение, ожидание 202, фактический статус.
	 */
	let { id }: { id: string } = $props();

	// Экран пересоздаётся ключом по id в маршруте, поэтому идентификатор
	// читается один раз: ресурс и таймеры принадлежат одной кампании.
	const campaignId = untrack(() => id);

	const UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
	const invalidId = !UUID_PATTERN.test(campaignId);

	const resource: Resource<Campaign> = createResource<Campaign>(
		`campaign:${campaignId}`,
		(signal) => getRolloutCampaign(campaignId, { signal }),
		{
			intervalMs: 3000,
			enabled: (): boolean =>
				!invalidId &&
				resource.data !== undefined &&
				isLive(resource.data.status) &&
				!campaignActions.isRollbackInFlight(campaignId),
			onSettled: (settled) => {
				if (settled.ok && settled.data) {
					campaignActions.noteStatus(campaignId, settled.data.status);
					// Копим метрики активной стадии: они понадобятся «Истории
					// переходов», когда стадия сменится (API хранит только текущие).
					stageStats.record(settled.data);
				}
			},
		},
	);

	const campaign = $derived(resource.data);
	const statusLabel = $derived(campaign ? CAMPAIGN_STATUS[campaign.status].label : undefined);
	const label = $derived(
		campaign
			? `${campaign.device_model} ${firmwareIndex.versionLabel(campaign.firmware_version_id) ?? ""}`.trimEnd()
			: "",
	);
	const firmware = $derived(campaign ? firmwareIndex.get(campaign.firmware_version_id) : undefined);
	const orderedStages = $derived(
		campaign ? [...campaign.rollout_stages].sort((a, b) => a.order_index - b.order_index) : [],
	);
	const activeStage = $derived(orderedStages.find((stage) => stage.status === "active"));
	const primaryAction = $derived(
		campaign ? allowedActions(campaign.status).find((action) => action !== "rollback") : undefined,
	);
	const canRollback = $derived(
		campaign ? allowedActions(campaign.status).includes("rollback") : false,
	);
	const inFlight = $derived(campaignActions.isRollbackInFlight(campaignId));
	const timedOut = $derived(campaignActions.isRollbackTimedOut(campaignId));
	const failure = $derived(campaignActions.failure(id));

	/** Конфликт по модели: живая кампания той же модели, если она уже загружена. */
	const conflicting = $derived.by(() => {
		if (!failure || failure.error.status !== 409 || !campaign) return undefined;
		return campaignCollection.rows.find(
			(row) =>
				row.id !== campaign.id && row.device_model === campaign.device_model && isLive(row.status),
		);
	});

	let confirmOpen = $state(false);

	$effect(() =>
		campaignActions.onChanged(async (updated) => {
			if (updated && updated.id === campaignId) {
				stageStats.record(updated);
				resource.write(updated);
				return;
			}
			await resource.refresh();
		}),
	);

	function perform(action: CampaignAction): void {
		if (!campaign) return;
		if (action === "rollback") {
			confirmOpen = true;
			return;
		}
		const args = [campaignId, label, statusLabel] as const;
		if (action === "start") void campaignActions.start(...args);
		if (action === "pause") void campaignActions.pause(...args);
		if (action === "resume") void campaignActions.resume(...args);
	}
</script>

<svelte:head>
	<title>
		{campaign ? `${label} - кампания` : "Кампания"} - OTA Firmware Orchestrator
	</title>
</svelte:head>

{#if invalidId}
	<EmptyState
		icon={FileXIcon}
		title="Некорректный идентификатор кампании"
		text="Проверьте ссылку или выберите кампанию из списка."
	>
		{#snippet action()}
			<Button variant="secondary" href="/campaigns">К списку кампаний</Button>
		{/snippet}
	</EmptyState>
{:else if !campaign && resource.error}
	{#if resource.error.status === 404}
		<EmptyState
			icon={FileDashedIcon}
			title="Кампания не найдена"
			text="Проверьте ссылку или выберите кампанию из списка."
		>
			{#snippet action()}
				<Button variant="secondary" href="/campaigns">К списку кампаний</Button>
			{/snippet}
		</EmptyState>
	{:else if resource.error.status === 400}
		<EmptyState
			icon={FileXIcon}
			title="Некорректный идентификатор кампании"
			text="Проверьте ссылку или выберите кампанию из списка."
		>
			{#snippet action()}
				<Button variant="secondary" href="/campaigns">К списку кампаний</Button>
			{/snippet}
		</EmptyState>
	{:else}
		<ErrorState error={resource.error} onretry={() => resource.refresh()} />
	{/if}
{:else if !campaign}
	<div class="grid gap-4">
		<Skeleton class="h-6 w-48" />
		<Skeleton class="h-16 w-full" />
		<div class="grid gap-4 xl:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
			<Skeleton class="h-64 w-full" />
			<Skeleton class="h-64 w-full" />
		</div>
	</div>
{:else}
	<div class="grid gap-5 pb-20 md:pb-0">
		<DetailHeader {campaign} versionLabel={firmware?.fw_version} onaction={perform} />

		{#if failure}
			<InlineBanner tone="danger" boxed>
				<span class="grid gap-1">
					<span class="font-medium text-fg-primary">
						{actionErrorHeadline(failure.action, failure.error, statusLabel)}
					</span>
					{#if failure.error.hint}
						<span>{failure.error.hint}</span>
					{/if}
					{#if conflicting}
						<a
							href="/campaigns/{conflicting.id}"
							class="underline underline-offset-4 hover:text-accent-text"
						>
							Открыть выполняющуюся кампанию этой модели
						</a>
					{/if}
					{#if failure.error.serverMessage}
						<details class="text-dense">
							<summary
								class="cursor-pointer list-none text-fg-secondary underline-offset-4 hover:underline"
							>
								Ответ сервера
							</summary>
							<span class="font-mono">{failure.error.serverMessage}</span>
						</details>
					{/if}
				</span>
			</InlineBanner>
		{/if}

		{#if timedOut}
			<InlineBanner tone="info" boxed>
				{#snippet action()}
					<Button variant="secondary" size="sm" onclick={() => resource.refresh()}>Обновить</Button>
				{/snippet}
				Запрос принят
			</InlineBanner>
		{/if}

		<IdentityStrip {campaign} {firmware} />

		<div class="grid gap-5 xl:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
			<div class="grid content-start gap-5">
				<Panel title="Воронка стадий">
					<div class="md:hidden">
						<StagePipeline stages={campaign.rollout_stages} orientation="vertical" />
					</div>
					<div class="hidden md:block">
						<StagePipeline stages={campaign.rollout_stages} />
					</div>
				</Panel>

				<StageTable {campaign} stats={campaign.stats} />
			</div>

			<MetricsPanel
				{campaign}
				stats={campaign.stats}
				{activeStage}
				refreshing={resource.refreshing}
				onrefresh={() => resource.refresh()}
				onstart={() => perform("start")}
			/>
		</div>

		<HistoryTimeline {campaign} />

		<details class="grid gap-2">
			<summary
				class="cursor-pointer list-none text-table text-fg-secondary underline-offset-4 hover:text-accent-text hover:underline focus-visible:outline-2 focus-visible:outline-accent"
			>
				Ответ API
			</summary>
			<JsonView value={campaign} label="GET /api/v1/campaigns/{id}" />
		</details>
	</div>

	<!-- Действия на узком экране: нижняя закреплённая полоса вместо шапки. -->
	{#if primaryAction || canRollback}
		<div
			class="fixed inset-x-0 bottom-0 z-30 flex items-center justify-end gap-2 border-t border-border-subtle bg-bg-surface px-4 py-3 md:hidden"
		>
			{#if canRollback}
				<Button
					variant="secondary"
					disabled={inFlight}
					loading={campaignActions.isPending(campaignId, "rollback")}
					onclick={() => perform("rollback")}
				>
					{actionMeta("rollback").label}
				</Button>
			{/if}
			{#if primaryAction}
				<Button
					variant="primary"
					disabled={inFlight}
					loading={campaignActions.isPending(campaignId, primaryAction)}
					onclick={() => perform(primaryAction)}
				>
					{actionMeta(primaryAction).label}
				</Button>
			{/if}
		</div>
	{/if}

	<ConfirmDialog
		open={confirmOpen}
		onOpenChange={(open) => (confirmOpen = open)}
		title={ROLLBACK_CONFIRM.title}
		body={ROLLBACK_CONFIRM.body}
		confirmLabel={ROLLBACK_CONFIRM.confirm}
		tone="danger"
		busy={campaignActions.isPending(campaignId, "rollback")}
		onconfirm={() => {
			confirmOpen = false;
			void campaignActions.rollback(campaignId, label, statusLabel);
		}}
	/>
{/if}
