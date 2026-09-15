<script lang="ts">
	import type { Snippet } from "svelte";
	import type { Glyph } from "./glyph";
	import Icon from "./Icon.svelte";
	import Spinner from "./Spinner.svelte";
	import { cn } from "./cn";

	/**
	 * Четыре варианта по семантике тона: solid-акцент только для основного
	 * действия, solid-danger только внутри диалога подтверждения (00-CONTEXT §8.2).
	 * Заливка плоская: градиент в приложении носит только бегунок
	 * сегментированного переключателя.
	 */
	let {
		variant = "secondary",
		size = "md",
		icon,
		iconPosition = "before",
		iconOnly = false,
		loading = false,
		disabled = false,
		type = "button",
		href,
		ariaLabel,
		onclick,
		class: className,
		children,
	}: {
		variant?: "primary" | "secondary" | "ghost" | "danger";
		size?: "sm" | "md";
		icon?: Glyph;
		iconPosition?: "before" | "after";
		iconOnly?: boolean;
		loading?: boolean;
		disabled?: boolean;
		type?: "button" | "submit";
		/** Превращает кнопку в ссылку с тем же видом: навигация, а не действие. */
		href?: string;
		/** Обязателен при `iconOnly`: кнопка без текста не может остаться без имени. */
		ariaLabel?: string;
		onclick?: (event: MouseEvent) => void;
		class?: string;
		children?: Snippet;
	} = $props();

	const VARIANTS: Record<typeof variant, string> = {
		primary: "border-transparent bg-accent text-accent-ink hover:bg-accent/85",
		secondary: "border-border-strong bg-bg-raised text-fg-primary hover:bg-bg-inset",
		ghost:
			"border-transparent bg-transparent text-fg-secondary hover:bg-bg-raised hover:text-fg-primary",
		danger: "border-transparent bg-state-danger text-bg-surface hover:bg-state-danger/85",
	};

	const SIZES: Record<typeof size, string> = {
		sm: "h-10 gap-1.5 px-2.5 text-dense md:h-7",
		md: "h-10 gap-2 px-3.5 text-ui md:h-9",
	};

	const ICON_ONLY: Record<typeof size, string> = {
		sm: "h-10 w-10 md:h-7 md:w-7",
		md: "h-10 w-10 md:h-9 md:w-9",
	};

	const classes = $derived(
		cn(
			"inline-flex select-none items-center justify-center whitespace-nowrap rounded-control border font-medium transition active:scale-[.98]",
			"disabled:pointer-events-none disabled:opacity-55",
			VARIANTS[variant],
			iconOnly ? ICON_ONLY[size] : SIZES[size],
			className,
		),
	);
</script>

{#if href}
	<a {href} aria-label={iconOnly ? ariaLabel : undefined} class={classes}>
		{#if icon && iconPosition === "before"}
			<Icon glyph={icon} size={size === "sm" ? 16 : 18} />
		{/if}
		{#if !iconOnly}
			<span class="whitespace-nowrap">{@render children?.()}</span>
		{/if}
		{#if icon && iconPosition === "after"}
			<Icon glyph={icon} size={size === "sm" ? 16 : 18} />
		{/if}
	</a>
{:else}
	<button
		{type}
		{onclick}
		disabled={disabled || loading}
		aria-label={iconOnly ? ariaLabel : undefined}
		aria-busy={loading || undefined}
		class={classes}
	>
		{#if loading}
			<Spinner />
		{:else if icon && iconPosition === "before"}
			<Icon glyph={icon} size={size === "sm" ? 16 : 18} />
		{/if}
		{#if !iconOnly}
			<span class="whitespace-nowrap">{@render children?.()}</span>
		{/if}
		{#if icon && iconPosition === "after" && !loading}
			<Icon glyph={icon} size={size === "sm" ? 16 : 18} />
		{/if}
	</button>
{/if}
