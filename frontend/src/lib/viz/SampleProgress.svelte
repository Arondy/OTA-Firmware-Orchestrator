<script lang="ts">
	import { int, plural } from "$lib/format/number";
	import { cn } from "$lib/ui/cn";

	/**
	 * Набор выборки одной строкой: полоса и счётчик наблюдений. Подписей
	 * «набор выборки» нет - счётчик и цвет полосы говорят всё сами.
	 */
	let {
		sampleSize,
		minSampleSize,
		class: className,
	}: {
		sampleSize: number;
		minSampleSize: number;
		class?: string;
	} = $props();

	const forms: [string, string, string] = ["наблюдение", "наблюдения", "наблюдений"];
	const complete = $derived(sampleSize >= minSampleSize);
	const ratio = $derived(minSampleSize === 0 ? 0 : Math.min(1, sampleSize / minSampleSize));
</script>

<div class={cn("flex items-center gap-2", className)}>
	<div
		role="progressbar"
		aria-label={`Набор выборки: ${int(sampleSize)} из ${int(minSampleSize)} ${plural(minSampleSize, forms)}`}
		aria-valuemin={0}
		aria-valuemax={minSampleSize}
		aria-valuenow={Math.min(sampleSize, minSampleSize)}
		class="h-1 min-w-0 flex-1 overflow-hidden rounded-full bg-bg-inset"
	>
		<div
			class={cn("h-full rounded-full", complete ? "bg-state-success" : "bg-state-neutral")}
			style="width: {ratio * 100}%"
		></div>
	</div>
	<span class="shrink-0 text-dense tabular-nums text-fg-secondary">
		{int(sampleSize)}/{int(minSampleSize)}
	</span>
</div>
