<script lang="ts">
	import { getFieldContext } from "./field-context";
	import { inputClasses } from "./input-classes";

	let {
		id,
		value = $bindable(""),
		rows = 3,
		placeholder,
		disabled = false,
		mono = false,
		maxlength,
		class: className,
	}: {
		id: string;
		value?: string;
		rows?: number;
		placeholder?: string;
		disabled?: boolean;
		mono?: boolean;
		maxlength?: number;
		class?: string;
	} = $props();

	const field = getFieldContext();
	const invalid = $derived(field?.invalid ?? false);
	const describedBy = $derived(
		[field?.hintId, field?.errorId].filter(Boolean).join(" ") || undefined,
	);
</script>

<textarea
	{id}
	{rows}
	{placeholder}
	{disabled}
	{maxlength}
	bind:value
	aria-invalid={invalid || undefined}
	aria-describedby={describedBy}
	class={inputClasses(invalid, mono && "font-mono", "h-auto py-2 leading-relaxed", className)}
></textarea>
