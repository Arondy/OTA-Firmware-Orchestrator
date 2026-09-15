<script lang="ts">
	import { percent } from "$lib/format/number";
	import { cn } from "$lib/ui/cn";

	/**
	 * Полукруглая шкала процента успеха с маркером порога: ниже порога
	 * значение красится опасным тоном (00-CONTEXT §8.5).
	 */
	let {
		value,
		threshold,
		label = "процент успеха",
		size = "md",
		class: className,
	}: {
		/** Доля 0..1; `undefined`, когда метрик нет. */
		value: number | undefined;
		threshold: number;
		label?: string;
		/** `sm` - вставку в строку таблицы стадий. */
		size?: "md" | "sm";
		class?: string;
	} = $props();

	const RADIUS = 40;
	const ARC = Math.PI * RADIUS;

	const clamped = $derived(value === undefined ? 0 : Math.min(1, Math.max(0, value)));
	const below = $derived(value !== undefined && value < threshold);
	const color = $derived(below ? "var(--state-danger)" : "var(--state-success)");
</script>

<div class={cn("flex items-center gap-3", className)}>
	<svg
		viewBox="0 0 100 52"
		role="img"
		aria-label={`${label}: ${value === undefined ? "нет метрик" : percent(value)} при пороге ${percent(threshold)}`}
		class={size === "sm" ? "h-7 w-14 shrink-0" : "h-12 w-24 shrink-0"}
	>
		<path
			d="M 10 50 A 40 40 0 0 1 90 50"
			fill="none"
			stroke="var(--bg-inset)"
			stroke-width="8"
			stroke-linecap="round"
		/>
		{#if value !== undefined}
			<path
				d="M 10 50 A 40 40 0 0 1 90 50"
				fill="none"
				stroke={color}
				stroke-width="8"
				stroke-linecap="round"
				stroke-dasharray={`${clamped * ARC} ${ARC}`}
			/>
		{/if}
		<line
			x1={50 + Math.cos(Math.PI * (1 - threshold)) * (RADIUS + 6)}
			y1={50 - Math.sin(Math.PI * (1 - threshold)) * (RADIUS + 6)}
			x2={50 + Math.cos(Math.PI * (1 - threshold)) * (RADIUS - 6)}
			y2={50 - Math.sin(Math.PI * (1 - threshold)) * (RADIUS - 6)}
			stroke="var(--fg-muted)"
			stroke-width="2"
		>
			<title>порог успеха {percent(threshold)}</title>
		</line>
	</svg>

	<div class="grid">
		{#if size === "md"}
			<span class="text-micro font-medium text-fg-muted uppercase">{label}</span>
		{/if}
		<span
			class={cn(
				size === "sm"
					? "font-mono text-lead font-semibold tabular-nums"
					: "font-mono text-metric font-semibold tabular-nums",
				value === undefined ? "text-fg-muted" : below ? "text-state-danger" : "text-fg-primary",
			)}
		>
			{value === undefined ? "нет метрик" : percent(value)}
		</span>
		{#if size === "md"}
			<span class="text-dense text-fg-secondary">порог {percent(threshold)}</span>
		{/if}
	</div>
</div>
