<script lang="ts">
	import type { Stage } from "$lib/api/types";
	import { STAGE_STATUS, TONE_CLASSES, TONE_CSS_VAR } from "$lib/domain/status";
	import { stageNumber } from "$lib/domain/campaign-history";
	import { absoluteDateTimeTitle } from "$lib/format/datetime";
	import { int, percent, plural } from "$lib/format/number";
	import LiveDot from "$lib/ui/LiveDot.svelte";
	import Tooltip from "$lib/ui/Tooltip.svelte";
	import { cn } from "$lib/ui/cn";

	/**
	 * Воронка стадий: ширина сегмента пропорциональна приросту охвата стадии
	 * (минимум 8%, чтобы узкие канареечные стадии оставались читаемыми).
	 * Рисунок не доступен сам по себе, поэтому у него есть словесный
	 * `aria-label`, а каждая стадия продублирована подписью с подсказкой.
	 */
	let {
		stages,
		mode = "full",
		orientation = "horizontal",
		class: className,
	}: {
		stages: Stage[];
		mode?: "compact" | "full";
		/** Вертикальная воронка - для узкой колонки (<768px). */
		orientation?: "horizontal" | "vertical";
		class?: string;
	} = $props();

	const MIN_WIDTH = 8;

	const ordered = $derived([...stages].sort((a, b) => a.order_index - b.order_index));

	const widths = $derived.by(() => {
		const steps = ordered.map((stage, index) => {
			const previous = index === 0 ? 0 : ordered[index - 1].target_percent;
			return Math.max(1, stage.target_percent - previous);
		});
		const grown = steps.map((step) => Math.max(MIN_WIDTH, step));
		const total = grown.reduce((sum, value) => sum + value, 0);
		return grown.map((value) => (value / total) * 100);
	});

	const segments = $derived.by(() => {
		let cursor = 0;
		return ordered.map((stage, index) => {
			const at = cursor;
			cursor += widths[index];
			return { stage, at, width: widths[index] };
		});
	});

	const summary = $derived(
		`${ordered.length} ${plural(ordered.length, ["стадия", "стадии", "стадий"])}: ${ordered
			.map((stage) => `${STAGE_STATUS[stage.status].label} ${int(stage.target_percent)}%`)
			.join(", ")}`,
	);

	function stageHint(stage: Stage): string {
		const entered = stage.entered_at
			? `вошла ${absoluteDateTimeTitle(stage.entered_at)}`
			: "ещё не вошла в статус";
		return `порог ${percent(stage.success_threshold)}, мин. выборка ${int(stage.min_sample_size)}, ${entered}`;
	}
</script>

<div class={cn(orientation === "vertical" ? "flex gap-3" : "grid gap-2", className)}>
	{#if orientation === "horizontal"}
		<svg
			viewBox="0 0 100 6"
			preserveAspectRatio="none"
			role="img"
			aria-label={summary}
			class={mode === "compact" ? "h-1.5 w-full" : "h-2.5 w-full"}
		>
			{#each segments as segment (segment.stage.id)}
				<rect
					x={segment.at + 0.4}
					y="0"
					width={Math.max(0.8, segment.width - 0.8)}
					height="6"
					rx="1"
					fill={TONE_CSS_VAR[STAGE_STATUS[segment.stage.status].tone]}
					opacity={segment.stage.status === "pending" ? 0.45 : 1}
					stroke={segment.stage.status === "active" ? "var(--state-progress)" : "none"}
					stroke-width={segment.stage.status === "active" ? 0.6 : 0}
				>
					<title>
						{`Стадия ${stageNumber(segment.stage.order_index)}: ${STAGE_STATUS[segment.stage.status].label}, охват ${segment.stage.target_percent}%`}
					</title>
				</rect>
			{/each}
		</svg>
	{:else}
		<svg
			viewBox="0 0 6 100"
			preserveAspectRatio="none"
			role="img"
			aria-label={summary}
			class="h-40 w-1.5"
		>
			{#each segments as segment (segment.stage.id)}
				<rect
					y={segment.at + 0.4}
					x="0"
					height={Math.max(0.8, segment.width - 0.8)}
					width="6"
					rx="1"
					fill={TONE_CSS_VAR[STAGE_STATUS[segment.stage.status].tone]}
					opacity={segment.stage.status === "pending" ? 0.45 : 1}
					stroke={segment.stage.status === "active" ? "var(--state-progress)" : "none"}
					stroke-width={segment.stage.status === "active" ? 0.6 : 0}
				>
					<title>
						{`Стадия ${stageNumber(segment.stage.order_index)}: ${STAGE_STATUS[segment.stage.status].label}, охват ${segment.stage.target_percent}%`}
					</title>
				</rect>
			{/each}
		</svg>
	{/if}

	{#if mode === "full"}
		<ol class={orientation === "horizontal" ? "flex flex-wrap gap-x-4 gap-y-1" : "grid gap-1"}>
			{#each ordered as stage (stage.id)}
				<li class="flex items-center gap-1.5 text-dense">
					<Tooltip label={stageHint(stage)} tabbable>
						<span
							class="flex items-center gap-1.5 rounded-chip outline-none focus-visible:ring-2 focus-visible:ring-accent"
						>
							{#if STAGE_STATUS[stage.status].live}
								<LiveDot class={cn(TONE_CLASSES[STAGE_STATUS[stage.status].tone].dot)} />
							{:else}
								<span
									aria-hidden="true"
									class={cn(
										"size-1.5 rounded-full",
										TONE_CLASSES[STAGE_STATUS[stage.status].tone].dot,
									)}
								></span>
							{/if}
							<span class="text-fg-primary">
								Стадия {stageNumber(stage.order_index)}, {int(stage.target_percent)}%
							</span>
							<span class="text-fg-secondary">{STAGE_STATUS[stage.status].label}</span>
						</span>
					</Tooltip>
				</li>
			{/each}
		</ol>
	{/if}
</div>
