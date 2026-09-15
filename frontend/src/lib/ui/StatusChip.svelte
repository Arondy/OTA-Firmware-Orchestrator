<script lang="ts">
	import type { StatusMeta } from "$lib/domain/status";
	import { TONE_CLASSES } from "$lib/domain/status";
	import LiveDot from "./LiveDot.svelte";
	import { cn } from "./cn";

	/**
	 * Статус - точка тона и текстовая подпись, без подложки: цветные
	 * полупрозрачные плашки читаются как декорация, а не как данные
	 * (00-CONTEXT §8.2, §9). Одним цветом смысл не передаётся, поэтому точка
	 * всегда сопровождается подписью. `raw` уходит в title для отладки.
	 */
	let {
		meta,
		size = "md",
		raw,
		class: className,
	}: {
		meta: StatusMeta;
		size?: "sm" | "md";
		raw?: string;
		class?: string;
	} = $props();

	const tone = $derived(TONE_CLASSES[meta.tone]);
</script>

<span
	title={raw}
	class={cn(
		"inline-flex items-center gap-1.5 font-medium text-fg-secondary",
		size === "sm" ? "text-micro" : "text-dense",
		className,
	)}
>
	{#if meta.live}
		<LiveDot class={tone.dot} />
	{:else}
		<span class={cn("size-1.5 rounded-full", tone.dot)} aria-hidden="true"></span>
	{/if}
	{meta.label}
</span>
