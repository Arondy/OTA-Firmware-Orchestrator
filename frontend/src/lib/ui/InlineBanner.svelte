<script lang="ts">
	import type { Snippet } from "svelte";
	import InfoIcon from "phosphor-svelte/lib/InfoIcon";
	import WarningIcon from "phosphor-svelte/lib/WarningIcon";
	import XCircleIcon from "phosphor-svelte/lib/XCircleIcon";
	import Icon from "./Icon.svelte";
	import { cn } from "./cn";

	/**
	 * Полоса сообщения: «лентой» под топбаром (сбой поллинга, недоступный API)
	 * или штучной плашкой внутри контента (`boxed`): у плашки граница по всему
	 * периметру, иначе блок с одной линией выглядит обрывком ленты.
	 */
	let {
		tone: toneName = "info",
		boxed = false,
		action,
		class: className,
		children,
	}: {
		tone?: "info" | "warning" | "danger";
		boxed?: boolean;
		action?: Snippet;
		class?: string;
		children: Snippet;
	} = $props();

	const TONES = {
		info: {
			wrap: "border-border-subtle bg-bg-inset text-fg-secondary",
			icon: InfoIcon,
			glyph: "text-fg-muted",
		},
		warning: {
			wrap: "border-accent-line bg-accent-tint text-fg-primary",
			icon: WarningIcon,
			glyph: "text-accent-text",
		},
		danger: {
			wrap: "border-state-danger/35 bg-state-danger/12 text-fg-primary",
			icon: XCircleIcon,
			glyph: "text-state-danger",
		},
	} as const;

	const tone = $derived(TONES[toneName]);
</script>

<div
	role="status"
	class={cn(
		"flex items-center gap-2",
		boxed ? "rounded-panel border px-3 py-2" : "border-b px-4 py-2",
		tone.wrap,
		className,
	)}
>
	<Icon glyph={tone.icon} size={16} class={cn("shrink-0", tone.glyph)} />
	<div class="min-w-0 flex-1 text-table">{@render children()}</div>
	{#if action}
		<div class="shrink-0">{@render action()}</div>
	{/if}
</div>
