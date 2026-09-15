<script lang="ts">
	import { RadioGroup } from "bits-ui";
	import { getFieldContext } from "./field-context";
	import { cn } from "./cn";

	let {
		name,
		value = $bindable(""),
		items,
		legend,
		disabled = false,
		class: className,
	}: {
		name: string;
		value?: string;
		items: { value: string; label: string }[];
		legend: string;
		disabled?: boolean;
		class?: string;
	} = $props();

	const field = getFieldContext();
	const describedBy = $derived(
		[field?.hintId, field?.errorId].filter(Boolean).join(" ") || undefined,
	);
</script>

<RadioGroup.Root
	bind:value
	{disabled}
	aria-describedby={describedBy}
	class={cn("grid gap-1.5", className)}
>
	<span class="text-dense font-medium text-fg-secondary">{legend}</span>
	{#each items as item (item.value)}
		<label
			for={`${name}-${item.value}`}
			class={cn(
				"flex min-h-8 cursor-pointer items-center gap-2 text-ui",
				disabled && "cursor-not-allowed opacity-55",
				value === item.value ? "text-fg-primary" : "text-fg-secondary",
			)}
		>
			<RadioGroup.Item
				value={item.value}
				id={`${name}-${item.value}`}
				class="flex size-4 shrink-0 items-center justify-center rounded-full border border-border-strong bg-bg-raised data-[state=checked]:border-accent"
			>
				{#if value === item.value}
					<span class="size-2 rounded-full bg-accent" aria-hidden="true"></span>
				{/if}
			</RadioGroup.Item>
			<span>{item.label}</span>
		</label>
	{/each}
</RadioGroup.Root>
