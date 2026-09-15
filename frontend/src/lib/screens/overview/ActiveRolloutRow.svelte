<script lang="ts">
	import ArrowCounterClockwiseIcon from "phosphor-svelte/lib/ArrowCounterClockwiseIcon";
	import PauseIcon from "phosphor-svelte/lib/PauseIcon";
	import PlayIcon from "phosphor-svelte/lib/PlayIcon";
	import type { CampaignRow } from "$lib/domain/campaign-row";
	import { activeStageOf } from "$lib/domain/campaign-row";
	import { allowedActions } from "$lib/domain/transitions";
	import { CAMPAIGN_STATUS } from "$lib/domain/status";
	import { ROLLBACK_CONFIRM, type CampaignAction } from "$lib/domain/transitions";
	import { campaignActions } from "$lib/state/campaign-actions.svelte";
	import ConfirmDialog from "$lib/ui/ConfirmDialog.svelte";
	import IconButton from "$lib/ui/IconButton.svelte";
	import StatusChip from "$lib/ui/StatusChip.svelte";
	import Tooltip from "$lib/ui/Tooltip.svelte";
	import Gauge from "$lib/viz/Gauge.svelte";
	import SampleProgress from "$lib/viz/SampleProgress.svelte";
	import StagePipeline from "$lib/viz/StagePipeline.svelte";

	let { row }: { row: CampaignRow } = $props();

	const label = $derived(row.version ? `${row.model} ${row.version}` : row.model);
	const stage = $derived(activeStageOf(row));
	const actions = $derived(allowedActions(row.status));
	const rollbackPending = $derived(campaignActions.isRollbackInFlight(row.id));

	let confirmOpen = $state(false);

	function actionPending(action: CampaignAction): boolean {
		return campaignActions.isPending(row.id, action) || (action === "rollback" && rollbackPending);
	}
</script>

<!--
  Действия живут в двух местах разметки, потому что на узком экране они стоят
  справа от шкал, а на >=768px - в общем ряду строки, рядом с датчиком. Это
  разные родители, и одним блоком их не описать; сам набор кнопок поэтому
  вынесен в сниппет, чтобы список действий оставался в одном месте.
-->
{#snippet actionButtons()}
	{#if actions.includes("pause")}
		<Tooltip label="Поставить на паузу">
			<span class="inline-flex">
				<IconButton
					glyph={PauseIcon}
					ariaLabel="Поставить на паузу"
					size="sm"
					loading={actionPending("pause")}
					onclick={() => campaignActions.pause(row.id, label, CAMPAIGN_STATUS[row.status].label)}
				/>
			</span>
		</Tooltip>
	{/if}
	{#if actions.includes("resume")}
		<Tooltip label="Возобновить">
			<span class="inline-flex">
				<IconButton
					glyph={PlayIcon}
					ariaLabel="Возобновить"
					size="sm"
					loading={actionPending("resume")}
					onclick={() => campaignActions.resume(row.id, label, CAMPAIGN_STATUS[row.status].label)}
				/>
			</span>
		</Tooltip>
	{/if}
	{#if actions.includes("rollback")}
		<Tooltip label="Откатить">
			<span class="inline-flex">
				<IconButton
					glyph={ArrowCounterClockwiseIcon}
					ariaLabel="Откатить"
					size="sm"
					variant="danger"
					loading={actionPending("rollback")}
					onclick={() => (confirmOpen = true)}
				/>
			</span>
		</Tooltip>
	{/if}
{/snippet}

<!--
  Узкая ширина (<768px): строка складывается в колонку, но шкалы, датчик и
  действия остаются одной строкой - кнопки относятся к раскатке целиком и
  рядом со шкалами читаются как её управление, а не как отдельный блок под
  ними. На >=768px - одна горизонтальная линия: статус с воронкой и выборкой
  слева, датчик процента успеха в свободной средней зоне, действия справа;
  ничего не нависает над соседями и не оставляет пустую полосу.
-->
<li
	class="flex flex-col gap-3 border-b border-border-subtle py-3 last:border-b-0 md:flex-row md:items-center md:gap-6"
>
	<div class="grid min-w-0 flex-1 gap-2">
		<div class="flex flex-wrap items-center gap-2">
			<StatusChip meta={CAMPAIGN_STATUS[row.status]} raw={row.status} size="sm" />
			<a
				href="/campaigns/{row.id}"
				class="text-ui font-medium text-fg-primary underline-offset-4 hover:text-accent-text hover:underline"
			>
				{row.model}
			</a>
			{#if row.version}
				<span class="font-mono text-table text-fg-secondary">{row.version}</span>
			{:else}
				<span class="text-table text-fg-muted">нет данных</span>
			{/if}
		</div>

		<!-- Шкалы, датчик и действия одной строкой. Шкалам задана минимальная
		     ширина, поэтому при нехватке места датчик переносится на свою
		     строку, а не сжимает воронку до нечитаемой полоски. На >=768px
		     отсюда уходят и датчик, и действия (md:hidden): там они встают
		     в общий ряд строки. -->
		<div class="flex flex-wrap items-center gap-3">
			<div class="grid min-w-[11rem] flex-1 gap-2">
				<StagePipeline stages={row.stages} mode="compact" />

				{#if row.stats && stage}
					<SampleProgress
						sampleSize={row.stats.sample_size}
						minSampleSize={stage.min_sample_size}
					/>
				{:else if !row.stats}
					<span class="text-table text-fg-muted">нет метрик</span>
				{/if}
			</div>

			{#if row.stats && stage && row.stats.sample_size > 0}
				<div class="shrink-0 md:hidden">
					<Gauge value={row.stats.success_rate} threshold={stage.success_threshold} />
				</div>
			{/if}

			<div class="flex shrink-0 items-center gap-1 md:hidden">{@render actionButtons()}</div>
		</div>
	</div>

	<!-- На узком экране обёртка не нужна вовсе (hidden): её содержимое уже
	     стоит в строке шкал, а пустой блок добавлял бы лишний вертикальный
	     отступ. На >=768px md:contents растворяет обёртку, и датчик с
	     действиями становятся непосредственными элементами строки. -->
	<div class="hidden md:contents">
		{#if row.stats && stage && row.stats.sample_size > 0}
			<Gauge value={row.stats.success_rate} threshold={stage.success_threshold} class="shrink-0" />
		{/if}

		<div class="flex shrink-0 items-center gap-1">{@render actionButtons()}</div>
	</div>
</li>

<ConfirmDialog
	open={confirmOpen}
	onOpenChange={(open) => (confirmOpen = open)}
	title={ROLLBACK_CONFIRM.title}
	body={ROLLBACK_CONFIRM.body}
	confirmLabel={ROLLBACK_CONFIRM.confirm}
	tone="danger"
	busy={rollbackPending}
	onconfirm={() => {
		confirmOpen = false;
		void campaignActions.rollback(row.id, label, CAMPAIGN_STATUS[row.status].label);
	}}
/>
