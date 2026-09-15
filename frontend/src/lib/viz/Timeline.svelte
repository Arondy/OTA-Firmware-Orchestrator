<script lang="ts">
	import RelativeTime from "$lib/ui/RelativeTime.svelte";
	import { cn } from "$lib/ui/cn";
	import { TONE_CLASSES, type Tone } from "$lib/domain/status";

	/**
	 * Вертикальная история событий: точка в тоне события, линия связи,
	 * заголовок и момент времени. События приходит готовым списком -
	 * компонент ничего не додумывает (§5.3).
	 */
	let {
		entries,
		class: className,
	}: {
		entries: { key: string; title: string; tone: Tone; at: string | undefined; note?: string }[];
		class?: string;
	} = $props();
</script>

<ol class={cn("grid", className)}>
	{#each entries as entry, index (entry.key)}
		<li class="relative grid grid-cols-[auto_minmax(0,1fr)] gap-x-3 pb-4 last:pb-0">
			{#if index < entries.length - 1}
				<span aria-hidden="true" class="absolute left-[5px] top-4 h-full w-px bg-border-subtle"
				></span>
			{/if}
			<span
				aria-hidden="true"
				class={cn("mt-1.5 size-2.5 rounded-full", TONE_CLASSES[entry.tone].dot)}
			></span>
			<div class="grid content-start gap-0.5">
				<span class="text-table text-fg-primary">{entry.title}</span>
				{#if entry.note}
					<span class="text-dense text-fg-secondary">{entry.note}</span>
				{/if}
				{#if entry.at}
					<RelativeTime iso={entry.at} />
				{/if}
			</div>
		</li>
	{/each}
</ol>
