<script lang="ts">
	import ArrowCounterClockwiseIcon from "phosphor-svelte/lib/ArrowCounterClockwiseIcon";
	import PauseIcon from "phosphor-svelte/lib/PauseIcon";
	import PlayIcon from "phosphor-svelte/lib/PlayIcon";
	import RocketLaunchIcon from "phosphor-svelte/lib/RocketLaunchIcon";
	import type { Campaign } from "$lib/api/types";
	import { CAMPAIGN_STATUS } from "$lib/domain/status";
	import {
		actionMeta,
		allowedActions,
		isTerminal,
		type CampaignAction,
	} from "$lib/domain/transitions";
	import { absoluteDateTime } from "$lib/format/datetime";
	import Breadcrumb from "$lib/layout/Breadcrumb.svelte";
	import { campaignActions } from "$lib/state/campaign-actions.svelte";
	import IconButton from "$lib/ui/IconButton.svelte";
	import IndeterminateBar from "$lib/ui/IndeterminateBar.svelte";
	import StatusChip from "$lib/ui/StatusChip.svelte";
	import Tooltip from "$lib/ui/Tooltip.svelte";
	import type { Glyph } from "$lib/ui/glyph";

	/**
	 * Шапка детали кампании: крошка, заголовок, живой статус и действия.
	 * Невозможных кнопок нет: терминальные статусы показывают только строку
	 * факта, а подсказка на статусе объясняет жизненный цикл.
	 */
	let {
		campaign,
		versionLabel,
		onaction,
	}: {
		campaign: Campaign;
		versionLabel: string | undefined;
		/** Экран решает, куда отправить действие: диалог подтверждения или сразу. */
		onaction: (action: CampaignAction) => void;
	} = $props();

	const meta = $derived(CAMPAIGN_STATUS[campaign.status]);
	const actions = $derived(allowedActions(campaign.status));
	const inFlight = $derived(campaignActions.isRollbackInFlight(campaign.id));

	/** Первичное действие контекста: первое разрешённое, кроме отката. */
	const primary = $derived(actions.find((action) => action !== "rollback"));

	const ACTION_ICONS: Record<CampaignAction, Glyph> = {
		start: RocketLaunchIcon,
		pause: PauseIcon,
		resume: PlayIcon,
		rollback: ArrowCounterClockwiseIcon,
	};
</script>

<div class="grid gap-3">
	<Breadcrumb
		items={[
			{ href: "/campaigns", label: "Кампании" },
			{ label: versionLabel ? `${campaign.device_model} ${versionLabel}` : campaign.device_model },
		]}
	/>

	<div class="flex flex-wrap items-start justify-between gap-3">
		<div class="grid gap-1">
			<h1 class="text-page font-semibold tracking-[-0.01em] text-fg-primary">
				{campaign.device_model}
			</h1>
			{#if versionLabel}
				<span class="font-mono text-table tabular-nums text-fg-secondary">{versionLabel}</span>
			{/if}
			<div class="mt-1 flex flex-wrap items-center gap-2">
				<StatusChip {meta} raw={campaign.status} />
				{#if isTerminal(campaign.status)}
					<span class="text-dense text-fg-muted">
						{campaign.status === "completed" ? "Кампания завершена" : "Кампания откачена"}
						{#if campaign.completed_at}
							{absoluteDateTime(campaign.completed_at)}
						{/if}
					</span>
				{/if}
			</div>
		</div>

		{#if actions.length > 0}
			<div class="hidden items-center gap-2 md:flex">
				{#if primary}
					<Tooltip label={actionMeta(primary).label}>
						<span class="inline-flex">
							<IconButton
								glyph={ACTION_ICONS[primary]}
								ariaLabel={actionMeta(primary).label}
								variant="secondary"
								loading={campaignActions.isPending(campaign.id, primary)}
								disabled={inFlight}
								onclick={() => onaction(primary)}
							/>
						</span>
					</Tooltip>
				{/if}
				{#if actions.includes("rollback")}
					<Tooltip label="Откатить кампанию">
						<span class="inline-flex">
							<IconButton
								glyph={ArrowCounterClockwiseIcon}
								ariaLabel="Откатить"
								variant="danger"
								loading={campaignActions.isPending(campaign.id, "rollback")}
								disabled={inFlight}
								onclick={() => onaction("rollback")}
							/>
						</span>
					</Tooltip>
				{/if}
			</div>
		{/if}
	</div>

	{#if inFlight}
		<div class="grid gap-1.5">
			<IndeterminateBar label="Применяем откат" />
			<span class="text-dense text-fg-secondary">Применяем откат</span>
		</div>
	{/if}
</div>
