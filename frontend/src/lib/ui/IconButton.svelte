<script lang="ts">
	import type { Glyph } from "./glyph";
	import Icon from "./Icon.svelte";
	import Spinner from "./Spinner.svelte";
	import { cn } from "./cn";

	/**
	 * Квадратная кнопка под одну иконку; имя обязательно. Это единственный
	 * компонент для кнопок-иконок действий (пауза, откат, копирование-ссылка
	 * «новая кампания» и т.п.): hover-подсветка, размеры и подсказки живут
	 * только здесь, чтобы на разных экранах кнопки вели себя одинаково.
	 *
	 * Подсветка при наведении - `bg-bg-inset`: она темнее `bg-bg-raised`,
	 * поэтому остаётся видимой и поверх строки таблицы, которая сама
	 * подсвечивается при наведении.
	 *
	 * Цветные варианты (`success`, `warning`, `danger`) красят только иконку и
	 * остаются прозрачными: в колонке «Действия» несколько кнопок подряд, и
	 * залитые плашки превратили бы её в светофор. Жёлтый берётся из
	 * единственного тёплого сигнального оттенка палитры - `--state-progress`.
	 *
	 * Подсказку при наведении даёт обёртка `Tooltip` (задержка 1 с);
	 * нативного `title` здесь нет, чтобы две подсказки не всплывали
	 * одновременно. С `href` рендерится ссылкой с тем же видом.
	 */
	let {
		glyph,
		ariaLabel,
		size = "md",
		variant = "ghost",
		loading = false,
		disabled = false,
		active = false,
		href,
		onclick,
		class: className,
	}: {
		glyph: Glyph;
		ariaLabel: string;
		size?: "sm" | "md";
		variant?: "ghost" | "secondary" | "success" | "warning" | "danger";
		loading?: boolean;
		disabled?: boolean;
		active?: boolean;
		/** Превращает кнопку в ссылку с тем же видом: навигация, а не действие. */
		href?: string;
		onclick?: (event: MouseEvent) => void;
		class?: string;
	} = $props();

	const VARIANTS = {
		ghost:
			"border-transparent bg-transparent text-fg-secondary hover:bg-bg-inset hover:text-fg-primary",
		secondary: "border-border-strong bg-bg-raised text-fg-primary hover:bg-bg-inset",
		success: "border-transparent bg-transparent text-state-success hover:bg-bg-inset",
		warning: "border-transparent bg-transparent text-state-progress hover:bg-bg-inset",
		danger: "border-transparent bg-transparent text-state-danger hover:bg-bg-inset",
	} as const;

	const SIZES = { sm: "h-10 w-10 md:h-7 md:w-7", md: "h-10 w-10 md:h-9 md:w-9" } as const;

	const classes = $derived(
		cn(
			"inline-flex select-none items-center justify-center rounded-control border transition active:scale-[.98]",
			"disabled:pointer-events-none disabled:opacity-55",
			VARIANTS[variant],
			SIZES[size],
			active && "border-accent-line bg-accent-tint text-accent-text",
			className,
		),
	);
</script>

{#if href}
	<a {href} aria-label={ariaLabel} class={classes}>
		<Icon {glyph} size={size === "sm" ? 16 : 18} />
	</a>
{:else}
	<button
		type="button"
		{onclick}
		aria-label={ariaLabel}
		aria-busy={loading || undefined}
		aria-pressed={active || undefined}
		disabled={disabled || loading}
		class={classes}
	>
		{#if loading}
			<Spinner />
		{:else}
			<Icon {glyph} size={size === "sm" ? 16 : 18} />
		{/if}
	</button>
{/if}
