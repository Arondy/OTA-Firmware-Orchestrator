<script lang="ts">
	import { Switch } from "bits-ui";
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
	<Switch.Root
		{id}
		bind:checked
		{disabled}
		aria-describedby={describedBy}
		class="relative h-5 w-9 shrink-0 rounded-full border border-border-strong bg-bg-inset transition data-[state=checked]:border-accent-line data-[state=checked]:bg-accent-tint"
	>
		<Switch.Thumb
			class="block size-4 translate-x-0.5 rounded-full bg-border-strong transition-transform data-[state=checked]:translate-x-4 data-[state=checked]:bg-accent"
		/>
	</Switch.Root>
	<span>{label}</span>
</label>
