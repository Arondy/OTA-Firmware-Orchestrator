<script lang="ts">
	import { percent } from "$lib/format/number";
	import { cn } from "./cn";

	/** Детерминированный прогресс 0..1; ширина - единственное динамическое значение. */
	let {
		value,
		label,
		tone = "accent",
		class: className,
	}: {
		value: number | undefined;
		label: string;
		tone?: "accent" | "success" | "danger" | "neutral";
		class?: string;
	} = $props();

	const clamped = $derived(value === undefined ? 0 : Math.min(1, Math.max(0, value)));
	const FILLS = {
		accent: "bg-accent",
		success: "bg-state-success",
		danger: "bg-state-danger",
		neutral: "bg-state-neutral",
	} as const;
</script>

<div class={cn("grid gap-1", className)}>
	<div class="flex items-baseline justify-between gap-2 text-dense">
		<span class="text-fg-muted">{label}</span>
		<span class="tabular-nums text-fg-secondary">
			{value === undefined ? "нет данных" : percent(clamped)}
		</span>
	</div>
	<div
		role="progressbar"
		aria-label={label}
		aria-valuemin={0}
		aria-valuemax={100}
		aria-valuenow={Math.round(clamped * 100)}
		class="h-1 overflow-hidden rounded-full bg-bg-inset"
	>
		<div class={cn("h-full rounded-full", FILLS[tone])} style="width: {clamped * 100}%"></div>
	</div>
</div>
