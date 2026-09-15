<script lang="ts">
	import ArchiveIcon from "phosphor-svelte/lib/ArchiveIcon";
	import ArrowClockwiseIcon from "phosphor-svelte/lib/ArrowClockwiseIcon";
	import PlugIcon from "phosphor-svelte/lib/PlugIcon";
	import RocketLaunchIcon from "phosphor-svelte/lib/RocketLaunchIcon";
	import type { Campaign, Stage, Stats } from "$lib/api/types";
	import { metricsVerdict } from "$lib/domain/metrics-verdict";
	import { TONE_CLASSES } from "$lib/domain/status";
	import { int, plural } from "$lib/format/number";
	import Button from "$lib/ui/Button.svelte";
	import EmptyState from "$lib/ui/EmptyState.svelte";
	import IconButton from "$lib/ui/IconButton.svelte";
	import MonoId from "$lib/ui/MonoId.svelte";
	import Panel from "$lib/ui/Panel.svelte";
	import Tooltip from "$lib/ui/Tooltip.svelte";
	import { cn } from "$lib/ui/cn";
	import Gauge from "$lib/viz/Gauge.svelte";
	import SampleProgress from "$lib/viz/SampleProgress.svelte";

	/**
	 * Регион метрик активной стадии. Три спроектированных случая: метрики
	 * есть; контроллер молчит; кампания не запущена. Ни один из них не
	 * показывает «0%» вместо отсутствия данных.
	 *
	 * Кнопка обновления - в правом верхнем углу шапки панели; подписи
	 * «обновлено только что» нет. Пояснение «идёт набор выборки» не пишется:
	 * счётчик наблюдений говорит всё сам, а вердикт показывается только когда
	 * выборка набрана (нейтральный вердикт про набор скрыт).
	 */
	let {
		campaign,
		stats,
		activeStage,
		refreshing,
		onrefresh,
		onstart,
	}: {
		campaign: Campaign;
		stats: Stats | undefined;
		/** Стадия со статусом `active` по данным Postgres. */
		activeStage: Stage | undefined;
		refreshing: boolean;
		onrefresh: () => Promise<void> | void;
		onstart: () => void;
	} = $props();

	// Спиннер только при ручном обновлении: фоновый поллинг каждые 3 с тоже
	// поднимает `refreshing`, но иконка при автообновлении обязана стоять.
	let manual = $state(false);

	async function refreshManually(): Promise<void> {
		manual = true;
		try {
			await onrefresh();
		} finally {
			manual = false;
		}
	}

	const verdict = $derived(
		stats && activeStage
			? metricsVerdict({
					sampleSize: stats.sample_size,
					minSampleSize: activeStage.min_sample_size,
					successRate: stats.success_rate,
					threshold: activeStage.success_threshold,
				})
			: undefined,
	);
	const sampleComplete = $derived(
		activeStage !== undefined &&
			stats !== undefined &&
			stats.sample_size >= activeStage.min_sample_size,
	);
	/** Контроллер считает не ту стадию, что помечена активной в Postgres. */
	const controllerMismatch = $derived(
		stats && activeStage && stats.active_stage_id !== activeStage.id,
	);
	const live = $derived(campaign.status === "running" || campaign.status === "paused");
</script>

<Panel title="Метрики активной стадии" class="xl:sticky xl:top-[72px]">
	{#snippet actions()}
		<Tooltip label="Обновить данные кампании">
			<span class="inline-flex">
				<IconButton
					glyph={ArrowClockwiseIcon}
					ariaLabel="Обновить данные кампании"
					size="sm"
					loading={manual && refreshing}
					onclick={() => void refreshManually()}
				/>
			</span>
		</Tooltip>
	{/snippet}

	{#if stats && activeStage}
		<div class="grid gap-4">
			<Gauge value={stats.success_rate} threshold={activeStage.success_threshold} />

			<div class="grid gap-1">
				<SampleProgress
					sampleSize={stats.sample_size}
					minSampleSize={activeStage.min_sample_size}
				/>
				<span class={cn("text-dense", sampleComplete ? "text-state-success" : "text-fg-secondary")}>
					{int(stats.sample_size)}/{int(activeStage.min_sample_size)}
					{plural(activeStage.min_sample_size, ["наблюдение", "наблюдения", "наблюдений"])}
				</span>
			</div>

			{#if verdict && verdict.tone !== "neutral"}
				<p class={cn("text-table", TONE_CLASSES[verdict.tone].metric)}>{verdict.text}</p>
			{/if}

			<div class="grid gap-1 border-t border-border-subtle pt-3">
				<span class="text-micro text-fg-muted">Метрики стадии</span>
				<MonoId id={stats.active_stage_id} copyKey="campaign-detail-active-stage" />
				{#if controllerMismatch}
					<span class="text-dense text-fg-secondary">Активна другая стадия.</span>
				{/if}
			</div>
		</div>
	{:else if live}
		<EmptyState icon={PlugIcon} title="Метрики недоступны">
			{#snippet action()}
				<Button
					variant="secondary"
					size="sm"
					loading={manual && refreshing}
					onclick={() => void refreshManually()}
				>
					Проверить снова
				</Button>
			{/snippet}
		</EmptyState>
	{:else if campaign.status === "draft"}
		<EmptyState icon={RocketLaunchIcon} title="Кампания не запущена">
			{#snippet action()}
				<Button variant="primary" size="sm" onclick={onstart}>Запустить</Button>
			{/snippet}
		</EmptyState>
	{:else}
		<!-- Терминальный статус: контроллер уже не агрегирует данные по стадиям. -->
		<EmptyState icon={ArchiveIcon} title="Метрик нет" />
	{/if}
</Panel>
