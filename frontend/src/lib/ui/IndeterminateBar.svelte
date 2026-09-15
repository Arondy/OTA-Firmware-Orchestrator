<script lang="ts">
	import { cn } from "./cn";
	import { reducedMotion } from "$lib/state/motion.svelte";

	/**
	 * Индикатор «действие принято и ещё выполняется» для асинхронных мутаций.
	 * При prefers-reduced-motion движение заменяется статичной подписью:
	 * состояние остаётся объявленным, но ничего не движется (§8.6).
	 */
	let { label, class: className }: { label: string; class?: string } = $props();
</script>

{#if reducedMotion.current}
	<span class={cn("text-dense text-fg-secondary", className)} role="status">{label}</span>
{:else}
	<div
		role="progressbar"
		aria-label={label}
		class={cn("h-0.5 overflow-hidden bg-bg-inset", className)}
	>
		<div class="indeterminate-slide h-full w-1/3 bg-accent"></div>
	</div>
{/if}
