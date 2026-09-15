<script lang="ts">
	import type { Snippet } from "svelte";
	import { getFieldContext } from "./field-context";
	import { inputClasses } from "./input-classes";
	import { cn } from "./cn";

	let {
		id,
		value = $bindable<number | undefined>(undefined),
		step = 1,
		min,
		max,
		disabled = false,
		unit,
		placeholder,
		class: className,
	}: {
		id: string;
		value?: number | undefined;
		step?: number;
		min?: number;
		max?: number;
		disabled?: boolean;
		/** Единица измерения справа: %, устройств, наблюдений. */
		unit?: Snippet;
		placeholder?: string;
		class?: string;
	} = $props();

	const field = getFieldContext();
	const invalid = $derived(field?.invalid ?? false);
	const describedBy = $derived(
		[field?.hintId, field?.errorId].filter(Boolean).join(" ") || undefined,
	);
</script>

<div class={cn("relative", className)}>
	<input
		{id}
		type="number"
		{step}
		{min}
		{max}
		{disabled}
		{placeholder}
		bind:value
		aria-invalid={invalid || undefined}
		aria-describedby={describedBy}
		class={inputClasses(invalid, "font-mono tabular-nums", unit && "pr-16")}
	/>
	{#if unit}
		<span
			class="pointer-events-none absolute top-1/2 right-3 -translate-y-1/2 text-dense text-fg-muted"
		>
			{@render unit()}
		</span>
	{/if}
</div>
