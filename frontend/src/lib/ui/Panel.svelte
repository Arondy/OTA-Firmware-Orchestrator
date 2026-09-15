<script lang="ts">
	import type { Snippet } from "svelte";
	import WarningIcon from "phosphor-svelte/lib/WarningIcon";
	import Icon from "./Icon.svelte";
	import { cn } from "./cn";

	/** Единственный контейнер приложения: вложенных «карточек» нет. */
	let {
		title = "",
		subtitle,
		actions,
		footer,
		dense = false,
		tone = "default",
		class: className,
		children,
	}: {
		/** Заголовок панели; пустой означает панель без шапки. */
		title?: string;
		subtitle?: string;
		actions?: Snippet;
		footer?: Snippet;
		dense?: boolean;
		tone?: "default" | "danger" | "accent";
		class?: string;
		children?: Snippet;
	} = $props();

	// Тон панели читается по иконке в шапке, а не по цветной полосе вдоль
	// контейнера: полосы выглядят шаблонным приёмом, а не сигналом состояния.
	const TONE_ICON = {
		default: undefined,
		danger: WarningIcon,
		accent: undefined,
	} as const;
</script>

<section class={cn("min-w-0 rounded-panel border border-border-subtle bg-bg-surface", className)}>
	{#if title || subtitle || actions}
		<header class="flex items-start justify-between gap-3 border-b border-border-subtle px-4 py-3">
			<div class="min-w-0">
				<h2
					class="flex items-center gap-2 text-panel font-semibold tracking-[-0.01em] text-fg-primary"
				>
					{#if TONE_ICON[tone]}
						<Icon glyph={TONE_ICON[tone]} size={18} class="text-state-danger" />
					{/if}
					{title}
				</h2>
				{#if subtitle}
					<p class="mt-0.5 text-dense text-fg-secondary">{subtitle}</p>
				{/if}
			</div>
			{#if actions}
				<div class="flex shrink-0 items-center gap-2">{@render actions()}</div>
			{/if}
		</header>
	{/if}

	<div class={dense ? "p-2" : "p-4"}>{@render children?.()}</div>

	{#if footer}
		<footer class="border-t border-border-subtle px-4 py-2">{@render footer()}</footer>
	{/if}
</section>
