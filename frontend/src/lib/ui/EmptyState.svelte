<script lang="ts">
	import type { Snippet } from "svelte";
	import type { Glyph } from "./glyph";
	import Icon from "./Icon.svelte";
	import { cn } from "./cn";

	/** Пустое состояние называет причину; пояснение только если без него непонятно (§9). */
	let {
		icon,
		title,
		text,
		action,
		class: className,
	}: {
		icon: Glyph;
		title: string;
		text?: string;
		action?: Snippet;
		class?: string;
	} = $props();
</script>

<div class={cn("grid justify-items-center gap-2 px-4 py-10 text-center", className)}>
	<Icon glyph={icon} size={24} class="text-fg-muted" />
	<h3 class="text-ui font-semibold text-fg-primary">{title}</h3>
	{#if text}
		<p class="max-w-[46ch] text-table text-fg-secondary">{text}</p>
	{/if}
	{#if action}
		<div class="mt-2">{@render action()}</div>
	{/if}
</div>
