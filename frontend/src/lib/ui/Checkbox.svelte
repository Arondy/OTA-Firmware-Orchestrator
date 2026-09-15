<script lang="ts">
	import { Checkbox } from "bits-ui";
	import CheckIcon from "phosphor-svelte/lib/CheckIcon";
	import Icon from "./Icon.svelte";
	import { getFieldContext } from "./field-context";
	import { cn } from "./cn";

	let {
		id,
		checked = $bindable(false),
		label,
		disabled = false,
		class: className,
	}: {
		id: string;
		checked?: boolean;
		label: string;
		disabled?: boolean;
		class?: string;
	} = $props();

	const field = getFieldContext();
	const describedBy = $derived(
		[field?.hintId, field?.errorId].filter(Boolean).join(" ") || undefined,
	);
</script>

<label
	for={id}
	class={cn(
		"flex min-h-8 cursor-pointer items-center gap-2 text-ui text-fg-primary",
		disabled && "cursor-not-allowed opacity-55",
		className,
	)}
>
	<Checkbox.Root
		{id}
		bind:checked
		{disabled}
		aria-describedby={describedBy}
		class="flex size-4 shrink-0 items-center justify-center rounded-chip border border-border-strong bg-bg-raised transition data-[state=checked]:border-accent data-[state=checked]:bg-accent"
	>
		{#if checked}
			<Icon glyph={CheckIcon} size={16} class="text-accent-ink" />
		{/if}
	</Checkbox.Root>
	<span>{label}</span>
</label>
